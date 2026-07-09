# EarnSaga Lite — Backend Scaffold

## What's here
- `cmd/server/main.go` — entrypoint, chi router, middleware, route wiring
- `internal/config` — env var loading
- `internal/db` — Postgres pool (pgx)
- `internal/auth` — Google ID token verification, JWT issue/parse, auth middleware
- `internal/handlers` — HTTP handlers (auth done, offers/wallet/leaderboard/callback are next)
- `internal/models` — shared structs
- `migrations/0001_init.sql` — full schema (users, offers, offer_goals, user_offers,
  user_offer_goals, wallet_transactions, events) — written for [goose](https://github.com/pressly/goose)
  migration format, but the SQL is plain enough to run manually if you'd rather not add goose as a dep.

## Local setup

```bash
cp .env.example .env
# fill in DATABASE_URL, JWT_SECRET, GOOGLE_CLIENT_ID, PUBSCALE_SECRET_KEY

go mod tidy   # fetches all dependencies — do this first, sandbox here has no module proxy access

# run the migration (either via goose, or paste the SQL into psql directly)
goose -dir migrations postgres "$DATABASE_URL" up

go run ./cmd/server
```

Health check: `GET http://localhost:8080/health` → `{"status":"ok"}`

## Auth flow implemented
1. Frontend gets a Google `id_token` via `@react-oauth/google` (or Google Identity Services JS SDK)
2. `POST /auth/google` with `{"id_token": "..."}`
3. Backend verifies the token against `GOOGLE_CLIENT_ID`, upserts the user, returns `{token, user}`
4. Frontend stores the JWT and sends `Authorization: Bearer <token>` on every subsequent request
5. `GET /me` is a protected example route — copy this pattern for offers/wallet/etc.

## Things you still need to set for real
- `GOOGLE_CLIENT_ID` — create an OAuth 2.0 Client ID in Google Cloud Console (Web application type),
  add your frontend origin to Authorized JavaScript origins
- `PUBSCALE_SECRET_KEY` — get this from the PubScale dashboard S2S config screen (per the assignment doc),
  not the same as the Pub-Key used for the offer API
- `DATABASE_URL` / `REDIS_URL` — Railway will inject these automatically once you provision
  the Postgres and Redis addons; for local dev point at a local instance or a Railway dev DB

## Next pieces to build (in this order, per the roadmap)
1. `internal/handlers/offers_handler.go` — sync from PubScale, list+search, detail
2. `internal/handlers/offer_actions_handler.go` — start offer (unique constraint already in schema)
3. `internal/handlers/callback_handler.go` — S2S callback, MD5 signature check, idempotency via `callback_token`
4. `internal/handlers/wallet_handler.go` — balance + transaction history
5. `internal/leaderboard` — Redis sorted set writes on credit, SSE endpoint for real-time push
6. `internal/handlers/analytics_handler.go` — `/api/events` ingestion + `/admin/analytics` aggregation

Say the word and I'll build the next one (offers sync + list/search/detail is the natural next step
since start-offer and analytics both depend on it).
