# EarnSaga Lite — Backend

Go backend for a simplified rewards platform: Google sign-in, PubScale offers, start/complete flow,
S2S reward callbacks, wallet, real-time leaderboard, and admin analytics. Built against
[`Fullstack-FTE Assignment.pdf`](Fullstack-FTE%20Assignment.pdf).

**Backend status: complete** against PDF requirements 1–10 (see checklist below). Local stack runs
via Docker; cloud deploy + private-repo submission are separate process steps for later.

## Architecture

Domain-driven: one folder per business capability under `internal/`, each following the same
`repository.go` (SQL/Redis) → `service.go` (business logic) → `handler.go` (HTTP) layering.
Every `Service` depends on a small, package-local `repository` interface rather than a concrete
struct, so a fake can be substituted in tests without touching how `main.go` wires things.

```
cmd/server/main.go   entrypoint — loads config, opens DB/Redis, wires every domain, starts chi router
Dockerfile           multi-stage build for the API image
docker-compose.yml   postgres + redis + api (one-command local stack)
docker/postgres/     first-boot migration init script
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
migrations/            plain SQL, goose-formatted; applied automatically on first Docker Postgres boot
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
# fill in JWT_SECRET, GOOGLE_CLIENT_ID, PUBSCALE_SECRET_KEY
# DATABASE_URL should use localhost:5433 (compose maps Postgres there)

docker compose up -d postgres redis
go mod tidy
go run ./cmd/server
```

If the Postgres volume is brand new and you are not using the full compose stack (which runs
`docker/postgres/init-migrations.sh`), apply migrations once with goose or by piping the Up
sections of `migrations/*.sql` into `psql`. Easiest reset: `docker compose down -v` then
`docker compose up --build` so init runs again.

Health check: `GET http://localhost:8080/health` → `{"status":"ok"}`

See [`API_TESTING.md`](API_TESTING.md) for a full copy-pasteable curl workflow exercising every
route in order, including how to compute the PubScale S2S callback signature.

## Auth flow

1. Client obtains a Google `id_token` (Google Identity Services / OAuth client)
2. `POST /api/v1/auth/google` with `{"id_token": "..."}`
3. Backend verifies the token against `GOOGLE_CLIENT_ID`, finds-or-creates the user, returns `{token, user}`
4. Client sends `Authorization: Bearer <token>` on every subsequent request
5. In `ENV=development`, `GET /api/v1/dev/token?email=you@example.com` mints a JWT without Google —
   useful for curling protected routes during backend testing

## Route map

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/health` | — | |
| GET | `/callbacks/pubscale` | signature only | PubScale S2S reward callback |
| POST | `/api/v1/auth/google` | — | |
| GET | `/api/v1/dev/token` | — | `ENV=development` only |
| GET | `/api/v1/users/profile` | JWT | |
| GET | `/api/v1/users/wallet` | JWT | |
| GET | `/api/v1/users/wallet/transactions` | JWT | includes offer/goal when attributed |
| GET | `/api/v1/offers` | JWT | `?search=` |
| GET | `/api/v1/offers/{id}` | JWT | name, icon, description, payout, goals, status |
| POST | `/api/v1/offers/{id}/start` | JWT | idempotent; returns substituted tracking URL |
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
`ALLOWED_ORIGINS` is comma-separated CORS origins (defaults to local Vite/CRA ports if unset).

## Testing

No Postgres/Redis needed. Unit tests use fakes.

### Run (readable output)

```powershell
.\scripts\test.ps1          # Windows
```

```bash
./scripts/test.sh           # macOS / Linux / Git Bash
```

Format:

```
-: TestName                 # test started
-> PASS  TestName  (0.00s)  # passed (green)
-> FAIL  TestName  (0.00s)  << FAILED   # failed (red) — easy to spot
```

End of run: `Summary: N passed, M failed` + list of any failures.

### Run (plain Go)

```powershell
go test -v ./...            # all packages
go test -v ./internal/auth  # one package
go test -v ./internal/auth -run TestIssueAndParseToken_RoundTrip  # one test
```

`go test` always uses Go’s own format (`=== RUN` / `--- PASS`).  
`? package [no test files]` = that package has no tests. Not a failure. Ignore it.  
Only `FAIL` / non-zero exit code means something broke.

The scripts wrap `go test` and rewrite the log. Running `go test` directly will **not** show `-:` / `->`.

### What is covered

| Package | Coverage |
|---|---|
| `internal/callback` | MD5 signature; idempotent credit; leaderboard failure after credit; attribution match + fallback |
| `internal/offer` | Start once + URL substitute; repeat start; completed preserved; detail status |
| `internal/auth` | JWT round-trip / reject bad tokens; middleware; admin gate |
| `internal/user` | Profile mapping; FindOrCreate; IsAdmin |
| `internal/analytics` | Date range parse; offer_id validation; track event |
| `internal/leaderboard` | Rank + enrich; invalid range default; RecordEarning |

Live API curls: [`API_TESTING.md`](API_TESTING.md).

## PDF alignment (backend)

| # | Requirement | Backend status |
|---|---|---|
| 1 | Google sign-in, find-or-create user, JWT, protected routes | Done — `internal/auth`, `internal/user` |
| 2 | Fetch/store PubScale offers, re-sync without duplicates | Done — upsert on `pubscale_id` |
| 3 | Active offers list + backend name search | Done — `GET /api/v1/offers?search=` |
| 4 | Offer detail (name, icon, description, payout, goals) + status for CTA | Done — `GET /api/v1/offers/{id}` |
| 5 | Start once, substitute `{your_user_id}`, return tracking URL, no error on repeat | Done — `POST .../start` |
| 6 | S2S callback: receive, MD5 verify, idempotent token, credit wallet, mark goal, 2xx | Done — `GET /callbacks/pubscale` |
| 7 | Wallet balance + transaction history (amount, offer/goal, timestamp, type) | Done — wallet endpoints |
| 8 | Leaderboard daily/weekly/all-time, Redis, top 50, real-time SSE | Done — leaderboard endpoints |
| 9 | Admin analytics: date/offer dims, impressions/clicks/revenue/DAU, filters | Done — events + `/admin/analytics` |
| 10 | Automated tests | Done — `.\scripts\test.ps1` / `./scripts/test.sh` |

Optional PDF items not implemented (explicitly optional): PubScale IP whitelist; wallet
debits/filters/pagination.

## Known limitations

- **Wallet offer/goal attribution** — best-effort heuristic. PubScale's S2S callback payload only
  carries `user_id/value/token/signature`, not which offer/goal earned the reward.
  `callback.selectAttributionMatch` matches the callback `value` to an in-progress goal reward
  first, then falls back to the oldest in-progress offer. A fully precise fix would need an
  offer/goal id round-tripped through the tracking URL.
- **Cloud deploy / submission** — local Docker is ready; a public URL and private-repo collaborator
  invites are still required by the assignment submission section (not backend feature work).
