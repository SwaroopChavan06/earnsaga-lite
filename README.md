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

### Option A — full stack with Docker (recommended)

```bash
cp .env.example .env
# fill in JWT_SECRET, GOOGLE_CLIENT_ID, PUBSCALE_SECRET_KEY
# (DATABASE_URL / REDIS_URL in .env are for host-side Go runs; compose overrides them)

docker compose up --build
```

This starts Postgres, Redis, and the API. Migrations apply automatically on the **first**
Postgres boot (empty volume). API: `http://localhost:8080`.

```bash
docker compose up -d          # detached
docker compose logs -f api    # follow API logs
docker compose down           # stop (keeps DB volume)
docker compose down -v        # stop and wipe DB volume (re-runs migrations next up)
```

### Option B — Go on the host, deps in Docker

```bash
cp .env.example .env
# fill in DATABASE_URL, JWT_SECRET, GOOGLE_CLIENT_ID, PUBSCALE_SECRET_KEY
# DATABASE_URL should use localhost:5433 (compose maps Postgres there)

docker compose up -d postgres redis
# then apply migrations once (if this is a fresh DB):
#   docker compose exec -T postgres psql -U postgres -d earnsaga < migrations/0001_init.sql
#   (or use goose / the init script after wiping the volume)

go mod tidy
go run ./cmd/server
```

Health check: `GET http://localhost:8080/health` → `{"status":"ok"}`

See [`API_TESTING.md`](API_TESTING.md) for a full copy-pasteable curl workflow exercising every
route in order, including how to compute the PubScale S2S callback signature.

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
| GET | `/api/v1/admin/analytics` | JWT + is_admin | `?from=&to=&offer_id=`, dates `YYYY-MM-DD` |
| POST | `/api/v1/admin/sync-offers` | JWT + is_admin | no request timeout, 5min internal cap |

Making a user an admin: `UPDATE users SET is_admin = true WHERE email = '...';` (manual, per the
assignment spec — there's no admin-promotion endpoint by design).

## Config

See [`.env.example`](.env.example) for every variable. `PUBSCALE_APP_ID`/`PUBSCALE_PUB_KEY` default
to the sandbox credentials from the assignment; `PUBSCALE_SECRET_KEY` (S2S signature secret, different
from the pub key) must come from the PubScale dashboard's S2S config screen and has no default.

## Testing

```bash
go test ./...
```

Unit tests cover every domain's `Service` against a fake `repository` (the unexported interface
each service already depends on — see the comment at the top of this file), no test DB required:
signature verification and callback idempotency (`internal/callback`), offer/goal attribution
matching (`internal/callback`), start-offer idempotency and status mapping (`internal/offer`), JWT
issue/parse/expiry and auth middleware (`internal/auth`), user upsert/admin lookup
(`internal/user`), analytics date-range parsing/validation (`internal/analytics`), and leaderboard
ranking/enrichment (`internal/leaderboard`). Repository-level tests that would need real
Postgres/Redis (e.g. the `ON CONFLICT` idempotency at the SQL level itself) are intentionally out
of scope here — the logic they'd cover is unit-tested at the service layer via fakes instead.

## Known gaps (next steps)

- **Wallet offer/goal attribution** — improved but still a heuristic. PubScale's S2S callback
  payload only carries `user_id/value/token/signature`, not which offer/goal earned the reward.
  `callback.selectAttributionMatch` now matches the callback's `value` against the reward amount
  of the user's in-progress offer goals first (a real signal), falling back to "oldest in-progress
  offer" only when nothing matches by value. A fully precise fix would need an offer/goal
  identifier round-tripped through the tracking URL, which PubScale's sandbox doesn't support here.
- **Frontend** — not started.
- **Cloud deployment** — local Docker stack is ready (`Dockerfile` + `docker compose up`); a
  public URL (Railway/Render/etc.) is still needed for assignment submission.
