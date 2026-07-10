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
- `Dockerfile` / `docker-compose.yml` / `docker/postgres/` — API image + local Postgres/Redis/API stack; migrations on first Postgres boot
- `internal/` — domain packages
  - `analytics/` — event ingestion (impressions/clicks) + admin reporting
  - `auth/` — JWT, Google ID token verification, auth + admin-gate middleware
  - `callback/` — PubScale S2S: MD5 signature verify, idempotent wallet credit, offer/goal attribution
  - `common/` — `WriteJSON` / `WriteError`
  - `config/` — env loading (`ALLOWED_ORIGINS`, PubScale, JWT, etc.)
  - `cache/` — Redis client
  - `db/` — `pgxpool`
  - `leaderboard/` — Redis rankings (daily/weekly/all-time) + SSE stream
  - `models/` — shared structs
  - `offer/` — PubScale sync/upsert, list+search, detail, start-once
  - `pubscale/` — PubScale HTTP client
  - `user/` — profile, Google find-or-create, `is_admin` (owns all `users` table access)
  - `wallet/` — balance + transaction history (with offer/goal when attributed)
- `migrations/` — goose-formatted SQL (Up applied by Docker init; Down stripped for init)

## Key Technologies

- **Language:** Go
- **HTTP:** chi
- **Database:** PostgreSQL (`pgx/v5`)
- **Cache:** Redis (`go-redis/v9`) — leaderboard only
- **Auth:** JWT + Google ID token validation
- **Config:** `godotenv` / env vars
- **Architecture:** Handler → Service → Repository per domain; services depend on unexported `repository` interfaces for tests

## Conventions

1. **File naming per domain:** `repository.go`, `service.go`, `handler.go` — no domain-name prefix. `auth/` may have extras (`google.go`, `jwt.go`).
2. **Layering:** Handlers parse HTTP and call services. Services hold business logic and depend on a package-local unexported `repository` interface. Repositories only do SQL/Redis. Domains do not query another domain's tables directly.
3. **HTTP responses:** Always `common.WriteError` / `common.WriteJSON` (including auth middleware).
4. **Context:** Pass `context.Context` through all layers.
5. **SQL:** Parameterized queries only; use `errors.Is(err, pgx.ErrNoRows)`, never string-compare errors.
6. **Models:** Shared types in `internal/models`; response DTOs live next to their service when domain-specific.
7. **Import direction:** Prefer narrow local interfaces (`auth.UserUpserter`, `auth.AdminChecker`) over cross-domain package imports that would cycle.
