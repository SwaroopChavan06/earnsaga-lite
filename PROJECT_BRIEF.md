# Project Brief: Go Backend Service

## Overview
This is a Go-based backend service designed with a clean, domain-driven architecture. It handles user authentication, offer tracking, PubScale S2S reward callbacks, wallet management, a real-time leaderboard, and basic analytics. It uses PostgreSQL for persistence, Redis for the leaderboard cache, and integrates with the external PubScale API.

## Directory Structure
- `cmd/server/`: Contains the main entry point (`main.go`) for the application — this is the only place domains get wired together.
- `internal/`: Contains the core business logic, separated by domain.
    - `analytics/`: Event ingestion (impressions/clicks) and admin-facing reporting. Deliberately one package, not split by write-side/read-side — they're the same concept.
    - `auth/`: JWT issue/parse, Google ID token verification, auth + admin-gate HTTP middleware.
    - `callback/`: PubScale S2S reward callback — MD5 signature verification, idempotent wallet crediting.
    - `common/`: Shared utilities, specifically for HTTP response handling (`WriteJSON`, `WriteError`).
    - `config/`: Environment variable loading and configuration management.
    - `cache/`: Redis client construction.
    - `db/`: Database connection pooling using `pgx`.
    - `leaderboard/`: Redis sorted-set rankings (daily/weekly/all-time) + SSE real-time stream.
    - `models/`: Shared data structures (User, Offer, WalletTransaction, etc.) used across domains.
    - `offer/`: PubScale sync/upsert, list+search, detail, start-offer-once.
    - `pubscale/`: HTTP client for the external PubScale offer API.
    - `user/`: User lookup, Google find-or-create, admin check. Owns all `users` table access — `auth` never touches it directly.
    - `wallet/`: Wallet balance and transaction history.
- `migrations/`: SQL migration files for database schema management.

## Key Technologies
- **Language:** Go (Golang)
- **Database:** PostgreSQL (via `pgx/v5` and `pgxpool`)
- **Cache:** Redis (`go-redis/v9`) — leaderboard sorted sets only
- **Authentication:** JWT (JSON Web Tokens), Google ID Token validation
- **Configuration:** `godotenv` for environment variables
- **Architecture:** Domain-driven design with a clear separation of Handlers (HTTP), Services (business logic), and Repositories (data access).

## Conventions for AI / future contributors
1. **File naming per domain:** exactly `repository.go`, `service.go`, `handler.go` — no domain-name prefix (the package name already disambiguates `offer.Service` from `wallet.Service`). `auth/` is the one exception with extra files (`google.go`, `jwt.go`) since it has more than one concern.
2. **Clean layering:** Handlers only parse requests and call services. Services contain business logic and depend on a small, package-local, **unexported** `repository` interface (defined in `service.go`, not a separate file) — not the concrete `*Repository` struct — so a fake can be substituted in tests. Repositories only handle SQL/Redis calls. A domain never reaches into another domain's table directly (e.g. `auth` calls `user.Service` for anything touching `users`, it doesn't run its own SQL against that table).
3. **Error handling:** Use `common.WriteError`/`common.WriteJSON` for all HTTP responses — including inside `auth` middleware, which used to be the one inconsistent spot.
4. **Context:** Always pass `context.Context` through the layers to handle timeouts and cancellations.
5. **Database:** Use `pgxpool` for database operations. Ensure all queries are parameterized to prevent SQL injection. Check `pgx.ErrNoRows` with `errors.Is`, never by comparing `err.Error()` strings.
6. **Models:** Shared structs are located in `internal/models`. Domain-local response DTOs (e.g. `wallet.TransactionResponse`) live next to their service.
7. **Import direction:** `internal/models` and `internal/common` are leaf packages nothing else in the domain graph depends on cyclically. If a domain needs another domain's data (e.g. `auth` needing user lookups), define the narrowest interface it actually needs locally (see `auth.UserUpserter`, `auth.AdminChecker`) rather than importing that domain's package wholesale — several domain handlers already import `auth` for context extraction, so `auth` importing them back would cycle.
