# EarnSaga Lite

Fullstack rewards platform: Google sign-in, PubScale offers, start/complete flow,
S2S reward callbacks, wallet, real-time leaderboard, and admin analytics. Built against
`[Fullstack-FTE Assignment.pdf](Fullstack-FTE%20Assignment.pdf)`.

**Status: complete** (backend + frontend). Full stack runs via a single `docker compose up --build`.

## Repository structure

```
earnsaga-lite/
  backend/             Go API (see below for domain layout)
  frontend/            React 19 + TypeScript + Vite + Tailwind CSS
  docker-compose.yml   postgres + redis + api + frontend — one command boots everything
  .env / .env.example  shared env vars (read by docker-compose)
```

### Backend (`backend/`)

Domain-driven: one folder per business capability under `internal/`, each following the same
`repository.go` (SQL/Redis) → `service.go` (business logic) → `handler.go` (HTTP) layering.
Every `Service` depends on a small, package-local `repository` interface rather than a concrete
struct, so a fake can be substituted in tests without touching how `main.go` wires things.

```
backend/
  cmd/server/main.go   entrypoint — loads config, opens DB/Redis, wires every domain, starts chi router
  Dockerfile           multi-stage build for the API image (golang:1.24-alpine → alpine:3.20)
  docker/postgres/     first-boot migration init script
  internal/
    auth/               Google ID token verification, JWT issue/parse, auth + admin-gate middleware
    user/               user lookup, Google find-or-create, is_admin check
    offer/              PubScale sync/upsert, list+search, detail, start-offer-once
    callback/           PubScale S2S callback — signature verification, idempotent wallet credit
    wallet/             balance + transaction history
    leaderboard/        Redis sorted sets (daily/weekly/all-time) + SSE stream + broadcaster
    analytics/          event ingestion (impression/click) + admin reporting
    pubscale/           HTTP client for the PubScale offer API
    cache/              Redis client
    db/                 Postgres pool (pgx)
    config/             env var loading
    models/             shared structs used across domains
  migrations/           plain SQL, goose-formatted; applied automatically on first Docker Postgres boot
```

### Frontend (`frontend/`)

```
frontend/
  src/
    api/        typed fetch wrappers per domain (auth, offers, wallet, leaderboard, analytics, events)
    components/ Navbar, AuthProvider, ProtectedRoute, Spinner, ErrorCard
    pages/      Login, Offers, OfferDetail, Wallet, Leaderboard, AdminAnalytics
    hooks/      useAuth (AuthContext consumer), useSSE (EventSource wrapper)
    types/      TypeScript interfaces mirroring backend DTOs
  Dockerfile    Node 24 build → nginx:1.27-alpine serve
  nginx.conf    SPA fallback + /api proxy to api:8080
```

## Local setup

### Option A — full stack with Docker (recommended)

```bash
cp .env.example .env
# fill in JWT_SECRET, GOOGLE_CLIENT_ID, PUBSCALE_SECRET_KEY, VITE_GOOGLE_CLIENT_ID
# (DATABASE_URL / REDIS_URL in .env are for host-side Go runs; compose overrides them)

docker compose up --build
```

This starts Postgres, Redis, the Go API, and the React frontend. Migrations apply automatically
on the **first** Postgres boot (empty volume).


| Service  | URL                      |
| -------- | ------------------------ |
| Frontend | `http://localhost:3000` |
| API      | `http://localhost:8080`  |


```bash
docker compose up -d               # detached
docker compose logs -f api         # follow API logs
docker compose logs -f frontend    # follow nginx logs
docker compose down                # stop (keeps DB volume)
docker compose down -v             # stop and wipe DB volume (re-runs migrations next up)
```

### Option B — frontend dev server + backend in Docker

```bash
cp .env.example .env

# Start backend deps + API in Docker
docker compose up -d postgres redis api

# Start frontend locally (hot-reload)
cd frontend
cp .env.example .env   # or: echo "VITE_API_URL=http://localhost:8080" > .env
                       #     echo "VITE_GOOGLE_CLIENT_ID=your-id" >> .env
npm install
npm run dev            # → http://localhost:5173
```

The Vite dev server proxies `/api`, `/callbacks`, `/health` to `http://localhost:8080`.

### Option C — Go on the host, deps in Docker

```bash
cp .env.example .env
# DATABASE_URL should use localhost:5433 (compose maps Postgres there)

docker compose up -d postgres redis
cd backend
go mod tidy
go run ./cmd/server
```

Health check: `GET http://localhost:8080/health` → `{"status":"ok"}`

See `[API_TESTING.md](API_TESTING.md)` for a full copy-pasteable curl workflow exercising every
route in order, including how to compute the PubScale S2S callback signature.

## Auth flow

1. User opens `http://localhost:3000` → redirected to `/login`
2. Google One-Tap / Sign-In button returns a Google `id_token` to the frontend
3. Frontend `POST /api/v1/auth/google` with `{"id_token": "..."}`
4. Backend verifies the token against `GOOGLE_CLIENT_ID`, finds-or-creates the user, returns `{token, user}`
5. Frontend stores JWT in `localStorage`; all subsequent API calls send `Authorization: Bearer <token>`
6. `ProtectedRoute` redirects unauthenticated visitors back to `/login`; admin pages additionally check `user.is_admin`
7. In `ENV=development`, `GET /api/v1/dev/token?email=you@example.com` mints a JWT without Google —
  useful for curling protected routes during backend testing
8. `EventSource` (used for the leaderboard SSE stream) can't set custom headers, so the frontend
  passes the JWT as `?token=` instead of `Authorization: Bearer`. `auth.Middleware` checks the
  header first and falls back to the query param only for that case — every other route still
  requires the header.

## Route map


| Method | Path                                | Auth           | Notes                                          |
| ------ | ----------------------------------- | -------------- | ---------------------------------------------- |
| GET    | `/health`                           | —              |                                                |
| GET    | `/callbacks/pubscale`               | signature only | PubScale S2S reward callback                   |
| POST   | `/api/v1/auth/google`               | —              |                                                |
| GET    | `/api/v1/dev/token`                 | —              | `ENV=development` only                         |
| GET    | `/api/v1/users/profile`             | JWT            |                                                |
| GET    | `/api/v1/users/wallet`              | JWT            | balance only                                   |
| GET    | `/api/v1/users/wallet/transactions` | JWT            | includes offer/goal when attributed            |
| GET    | `/api/v1/users/wallet/summary`      | JWT            | balance + transactions in one call (see below) |
| GET    | `/api/v1/offers`                    | JWT            | `?search=&page=&limit=` (paginated, see below) |
| GET    | `/api/v1/offers/{id}`               | JWT            | name, icon, description, payout, category/platform/offer_type, goals, status |
| POST   | `/api/v1/offers/{id}/start`         | JWT            | idempotent; returns substituted tracking URL   |
| GET    | `/api/v1/leaderboard`               | JWT            | `?range=daily|weekly|alltime`                  |
| GET    | `/api/v1/leaderboard/stream`        | JWT            | SSE, no request timeout, JWT via `?token=`     |
| POST   | `/api/v1/events`                    | JWT            | single impression/click ingestion              |
| POST   | `/api/v1/events/batch`              | JWT            | bulk ingestion — what the frontend actually calls (see below) |
| GET    | `/api/v1/admin/analytics`           | JWT + is_admin | `?from=&to=&offer_id=`, dates `YYYY-MM-DD`     |
| POST   | `/api/v1/admin/sync-offers`         | JWT + is_admin | no request timeout, 5min internal cap          |


Making a user an admin: `UPDATE users SET is_admin = true WHERE email = '...';` (manual, per the
assignment spec — there's no admin-promotion endpoint by design).

## API design notes

**Pagination.** `GET /api/v1/offers` is offset-based: `?page=1&limit=20` (both optional, `limit`
clamped to 1–100, default 20). The response is `{offers, total, page, limit, pages}` — the total
count comes from a `COUNT(*) OVER()` window function in the same query, so listing a page never
costs an extra round-trip. The frontend's Offers page uses this for Prev/Next controls instead of
fetching everything at once.

**API versioning is per-endpoint, not global.** `frontend/src/api/client.ts` exports version
constants (`API_V1`, and a commented-out `API_V2` template) instead of a single hardcoded prefix.
Each domain file (`offers.ts`, `wallet.ts`, etc.) imports the version it needs and builds its own
path. A breaking change to one endpoint only bumps that file's import — e.g. if `/events` needed a
`v2`, only `events.ts` would change; `offers.ts`/`wallet.ts` stay on `v1`. Non-breaking changes
(new optional fields, new optional query params) never require a version bump at all.

**Event batching.** The frontend queues impression/click events client-side (`src/api/events.ts`)
and flushes them together instead of firing one HTTP request per event — a page with 20 offer
cards used to mean 20 separate `POST /events` calls. Flush triggers: every 5s, when the queue hits
10 events, on tab hide, and on page unload (via `navigator.sendBeacon`, which survives the page
closing mid-request). The backend ingests the batch with `POST /api/v1/events/batch`, which does a
single bulk `INSERT ... SELECT * FROM unnest(...)` instead of one INSERT per event. `POST /events`
(singular) still exists for one-off callers.

**Wallet summary collocation.** The Wallet page always needs balance + transaction history
together, so `GET /api/v1/users/wallet/summary` returns both in one response instead of two
round-trips. `wallet.Service.GetSummary` fetches both with `errgroup` so the two DB queries run
concurrently — latency is `max(balance, transactions)`, not `balance + transactions`. The two
original single-purpose endpoints (`/wallet`, `/wallet/transactions`) are kept for callers that
only need one.

**Offer metadata.** PubScale's sync payload includes category tags (`ctg`), platform (`os`), and
offer type (`off_type`) — these were previously parsed by the PubScale client and discarded.
They're now persisted (`offers.category`/`platform`/`offer_type`, migration `0003`) and returned
on the detail endpoint, shown as badges on the offer detail page. The list endpoint intentionally
omits them to keep list payloads small — they're not needed until a user opens an offer.

## Config

See `[.env.example](.env.example)` for every variable. `PUBSCALE_APP_ID`/`PUBSCALE_PUB_KEY` default
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


| Package                | Coverage                                                                                         |
| ---------------------- | ------------------------------------------------------------------------------------------------ |
| `internal/callback`    | MD5 signature; idempotent credit; leaderboard failure after credit; attribution match + fallback |
| `internal/offer`       | Start once + URL substitute; repeat start; completed preserved; detail status                    |
| `internal/auth`        | JWT round-trip / reject bad tokens; middleware; admin gate                                       |
| `internal/user`        | Profile mapping; FindOrCreate; IsAdmin                                                           |
| `internal/analytics`   | Date range parse; offer_id validation; track event                                               |
| `internal/leaderboard` | Rank + enrich; invalid range default; RecordEarning                                              |


Live API curls: `[API_TESTING.md](API_TESTING.md)`.

## Concurrency

Places where sequential I/O was replaced with structured Go concurrency:


| Location                                | Pattern                                                                                                                                                | Effect                                                                         |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `analytics/repository.go` — `GetReport` | `errgroup` — two independent DB queries in parallel                                                                                                    | Latency = `max(q1, q2)` instead of `q1 + q2`                                   |
| `offer/service.go` — `GetDetail`        | `errgroup` — `ListGoals` + `GetUserOfferStatus` after `GetByID`                                                                                        | Same; `pgx.ErrNoRows` absorbed inside goroutine                                |
| `offer/service.go` — `SyncFromPubScale` | 10-worker pool via `sync.WaitGroup` + buffered channel; `sync/atomic.Int64` for the counter                                                            | Page of upserts runs in parallel instead of one-by-one                         |
| `leaderboard/broadcaster.go`            | Single background goroutine polls Redis every 3 s; fans out JSON to all SSE clients via buffered `chan []byte`; `sync.Mutex` guards the subscriber map | O(active ranges) Redis reads regardless of client count; was O(clients) before |
| `wallet/service.go` — `GetSummary`      | `errgroup` — balance + transaction history queried in parallel                                                                                         | Latency = `max(balance, transactions)` instead of sequential                  |


## PDF alignment (backend)


| #   | Requirement                                                                        | Backend status                                    |
| --- | ---------------------------------------------------------------------------------- | ------------------------------------------------- |
| 1   | Google sign-in, find-or-create user, JWT, protected routes                         | Done — `internal/auth`, `internal/user`           |
| 2   | Fetch/store PubScale offers, re-sync without duplicates                            | Done — upsert on `pubscale_id`                    |
| 3   | Active offers list + backend name search                                           | Done — `GET /api/v1/offers?search=&page=&limit=`  |
| 4   | Offer detail (name, icon, description, payout, goals) + status for CTA             | Done — `GET /api/v1/offers/{id}` (also returns category/platform/offer_type) |
| 5   | Start once, substitute `{your_user_id}`, return tracking URL, no error on repeat   | Done — `POST .../start`                           |
| 6   | S2S callback: receive, MD5 verify, idempotent token, credit wallet, mark goal, 2xx | Done — `GET /callbacks/pubscale`                  |
| 7   | Wallet balance + transaction history (amount, offer/goal, timestamp, type)         | Done — wallet endpoints                           |
| 8   | Leaderboard daily/weekly/all-time, Redis, top 50, real-time SSE                    | Done — leaderboard endpoints                      |
| 9   | Admin analytics: date/offer dims, impressions/clicks/revenue/DAU, filters          | Done — events + `/admin/analytics`                |
| 10  | Automated tests                                                                    | Done — `.\scripts\test.ps1` / `./scripts/test.sh` |


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

