# Project Brief: EarnSaga Lite Backend

## Overview

Go backend for EarnSaga Lite: Google auth (JWT), PubScale offer sync, start/complete flow, S2S
reward callbacks, wallet, Redis leaderboard with SSE, and admin analytics. Persistence is
PostgreSQL; leaderboard rankings are Redis sorted sets. Local stack is fully Dockerized
(`Dockerfile` + `docker-compose.yml`).

See [`README.md`](README.md) for setup/routes and [`API_TESTING.md`](API_TESTING.md) for curl
workflows. Assignment source: [`Fullstack-FTE Assignment.pdf`](Fullstack-FTE%20Assignment.pdf).

## Directory Structure

- `cmd/server/` — entrypoint (`main.go`); only place domains are wired together
- `Dockerfile` / `docker-compose.yml` — API image + local Postgres/Redis/API stack
- `render.yaml` (repo root) — Render Blueprint; deploys API + frontend + Postgres + Redis from one file
- `internal/` — domain packages
  - `analytics/` — event ingestion (impressions/clicks) + admin reporting
  - `auth/` — JWT, Google ID token verification, auth + admin-gate middleware
  - `callback/` — PubScale S2S: MD5 signature verify, idempotent wallet credit, offer/goal attribution
  - `common/` — `WriteJSON` / `WriteError`
  - `config/` — env loading (`ALLOWED_ORIGINS`, PubScale, JWT, etc.)
  - `cache/` — Redis client
  - `db/` — `pgxpool`
  - `leaderboard/` — Redis rankings (daily/weekly/all-time) + SSE stream; `broadcaster.go` fans one Redis poll out to all SSE clients
  - `models/` — shared structs
  - `offer/` — PubScale sync/upsert, paginated list+search, detail (incl. category/platform/offer_type), start-once
  - `pubscale/` — PubScale HTTP client
  - `user/` — profile, Google find-or-create, `is_admin` (owns all `users` table access)
  - `wallet/` — balance + transaction history (with offer/goal when attributed) + a collocated summary endpoint fetching both concurrently
  - `db/migrations/` — goose-formatted SQL, embedded into the binary (`go:embed`); `db.RunMigrations`
    applies pending ones on every startup via goose's own version-tracking table, in any environment

## Key Technologies

- **Language:** Go
- **HTTP:** chi
- **Database:** PostgreSQL (`pgx/v5`)
- **Cache:** Redis (`go-redis/v9`) — leaderboard only
- **Auth:** JWT + Google ID token validation (also accepted via `?token=` for SSE, which can't set headers)
- **Config:** `godotenv` / env vars
- **Architecture:** Handler → Service → Repository per domain; services depend on unexported `repository` interfaces for tests
- **Concurrency:** `golang.org/x/sync/errgroup` (parallel DB sub-queries in analytics/offer/wallet), `sync.WaitGroup` + buffered channels (offer sync worker pool), `sync/atomic` (race-free counters), `sync.Mutex` + goroutine broadcaster (SSE fan-out)
- **Pagination:** `GET /offers` is offset-based (`page`/`limit`, `COUNT(*) OVER()` for total in the same query)
- **API versioning:** per-endpoint, not global — frontend imports a version constant (`API_V1`) per domain file rather than hardcoding one prefix everywhere, so a breaking change to one endpoint doesn't force-bump the rest
- **Batching:** frontend queues impression/click events and flushes via `POST /events/batch` (bulk `unnest()` insert) instead of one request/insert per event

## Conventions

1. **File naming per domain:** `repository.go`, `service.go`, `handler.go` — no domain-name prefix. `auth/` may have extras (`google.go`, `jwt.go`).
2. **Layering:** Handlers parse HTTP and call services. Services hold business logic and depend on a package-local unexported `repository` interface. Repositories only do SQL/Redis. Domains do not query another domain's tables directly.
3. **HTTP responses:** Always `common.WriteError` / `common.WriteJSON` (including auth middleware).
4. **Context:** Pass `context.Context` through all layers.
5. **SQL:** Parameterized queries only; use `errors.Is(err, pgx.ErrNoRows)`, never string-compare errors.
6. **Models:** Shared types in `internal/models`; response DTOs live next to their service when domain-specific.
7. **Import direction:** Prefer narrow local interfaces (`auth.UserUpserter`, `auth.AdminChecker`) over cross-domain package imports that would cycle.
