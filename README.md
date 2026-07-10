# EarnSaga Lite — Backend

Go backend for a simplified rewards platform: Google sign-in, PubScale offers, start/complete flow,
S2S reward callbacks, wallet, real-time leaderboard, and basic analytics. Built against
[`Fullstack-FTE Assignment.pdf`](Fullstack-FTE%20Assignment.pdf).

## Architecture

Domain-driven: one folder per business capability under `internal/`, each following the same
`repository.go` (SQL/Redis) → `service.go` (business logic) → `handler.go` (HTTP) layering.
Every `Service` depends on a small, package-local `repository` interface rather than a concrete
struct, so a fake can be substituted in tests without touching how `main.go` wires things.

```
cmd/server/main.go   entrypoint — loads config, opens DB/Redis, wires every domain, starts chi router
internal/
  auth/               Google ID token verification, JWT issue/parse, auth + admin-gate middleware
  user/               user lookup, Google find-or-create, is_admin check
  offer/              PubScale sync/upsert, list+search, detail, start-offer-once
  callback/           PubScale S2S callback — signature verification, idempotent wallet credit
  wallet/             balance + transaction history
  leaderboard/        Redis sorted sets (daily/weekly/all-time) + SSE stream
  analytics/          event ingestion (impression/click) + admin reporting
  pubscale/           HTTP client for the PubScale offer API
  cache/               Redis client
  db/                  Postgres pool (pgx)
  config/              env var loading
  models/              shared structs used across domains
migrations/            plain SQL, goose-formatted but runnable manually
```

## Local setup

```bash
cp .env.example .env
# fill in DATABASE_URL, JWT_SECRET, GOOGLE_CLIENT_ID, PUBSCALE_SECRET_KEY

go mod tidy
go run ./cmd/server
```

Postgres + Redis for local dev: `docker-compose up -d` (see [`docker-compose.yml`](docker-compose.yml)).

Migrations: run `migrations/0001_init.sql` then `migrations/0002_offer_goals_unique.sql` against
`DATABASE_URL` (via `goose -dir migrations postgres "$DATABASE_URL" up`, or paste the SQL into `psql`
directly — it's plain enough to not need the tool).

Health check: `GET http://localhost:8080/health` → `{"status":"ok"}`

## Auth flow

1. Frontend gets a Google `id_token` (`@react-oauth/google` or Google Identity Services JS SDK)
2. `POST /api/v1/auth/google` with `{"id_token": "..."}`
3. Backend verifies the token against `GOOGLE_CLIENT_ID`, finds-or-creates the user, returns `{token, user}`
4. Frontend sends `Authorization: Bearer <token>` on every subsequent request
5. In `ENV=development`, `GET /api/v1/dev/token?email=you@example.com` mints a JWT without going
   through Google, for curling protected routes before a frontend exists

## Route map

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/health` | — | |
| GET | `/callbacks/pubscale` | signature only | PubScale S2S reward callback |
| POST | `/api/v1/auth/google` | — | |
| GET | `/api/v1/dev/token` | — | dev-env only |
| GET | `/api/v1/users/profile` | JWT | |
| GET | `/api/v1/users/wallet` | JWT | |
| GET | `/api/v1/users/wallet/transactions` | JWT | |
| GET | `/api/v1/offers` | JWT | `?search=` |
| GET | `/api/v1/offers/{id}` | JWT | |
| POST | `/api/v1/offers/{id}/start` | JWT | idempotent |
| GET | `/api/v1/leaderboard` | JWT | `?range=daily\|weekly\|alltime` |
| GET | `/api/v1/leaderboard/stream` | JWT | SSE, no request timeout |
| POST | `/api/v1/events` | JWT | impression/click ingestion |
| GET | `/api/v1/admin/analytics` | JWT + is_admin | |
| POST | `/api/v1/admin/sync-offers` | JWT + is_admin | no request timeout, 5min internal cap |

Making a user an admin: `UPDATE users SET is_admin = true WHERE email = '...';` (manual, per the
assignment spec — there's no admin-promotion endpoint by design).

## Config

See [`.env.example`](.env.example) for every variable. `PUBSCALE_APP_ID`/`PUBSCALE_PUB_KEY` default
to the sandbox credentials from the assignment; `PUBSCALE_SECRET_KEY` (S2S signature secret, different
from the pub key) must come from the PubScale dashboard's S2S config screen and has no default.

## Known gaps (next steps)

- **Analytics dimensions/metrics** — `/admin/analytics` currently returns coarse platform totals
  (users, revenue, active offers). The assignment asks for grouping by date/offer and separate
  impressions/clicks/revenue/DAU metrics — the `events` table already captures the raw data needed,
  this is a query/aggregation build-out, not a schema change.
- **Wallet offer/goal attribution** — improved but still a heuristic. PubScale's S2S callback
  payload only carries `user_id/value/token/signature`, not which offer/goal earned the reward.
  `callback.selectAttributionMatch` now matches the callback's `value` against the reward amount
  of the user's in-progress offer goals first (a real signal), falling back to "oldest in-progress
  offer" only when nothing matches by value. A fully precise fix would need an offer/goal
  identifier round-tripped through the tracking URL, which PubScale's sandbox doesn't support here.
- **Automated tests** — repositories are now behind interfaces specifically to unblock this;
  no test files exist yet.
- **Frontend** — not started.
- **Deployment** — no Dockerfile for the app itself yet (only `docker-compose.yml` for local
  Postgres/Redis).
