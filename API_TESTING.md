# API Testing Guide — EarnSaga Lite

Copy-pasteable `curl` workflow for the whole backend, in the order you'd actually exercise it:
auth → sync offers → browse/start an offer → simulate the S2S reward callback → check the wallet
→ leaderboard → analytics event → admin report.

Assumes the API is reachable at `http://localhost:8080` and `ENV=development` (needed for step 2's
dev token route). Start it with either:

```bash
docker compose up --build          # recommended — Postgres + Redis + API
# or: go run ./cmd/server          # with postgres/redis already up
```

**Windows / PowerShell users:** the commands below are bash (works as-is in WSL or Git Bash, both
common on Windows). In plain PowerShell, use `curl.exe` instead of `curl` (PowerShell aliases
`curl` to `Invoke-WebRequest`, which takes different flags), and see the PowerShell snippet in
step 6 for computing the MD5 signature instead of `openssl`.

Every response below is illustrative — actual `id`/`token` values will differ per run.

## 0. Set up shell variables

```bash
BASE_URL="http://localhost:8080"
```

## 1. Health check

```bash
curl -s "$BASE_URL/health"
```

```json
{"status":"ok"}
```

## 2. Get a JWT

**Dev shortcut** (only available when `ENV=development`) — mints a token without going through
Google at all:

```bash
curl -s "$BASE_URL/api/v1/dev/token?email=you@example.com"
```

```json
{"token":"eyJhbGciOi...","user":{"id":"...","email":"you@example.com","name":"Dev User","is_admin":false,...}}
```

Save the token for every subsequent request:

```bash
TOKEN=$(curl -s "$BASE_URL/api/v1/dev/token?email=you@example.com" | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
USER_ID=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/users/profile" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
```

**Real Google login** (same response shape): obtain a Google `id_token` from a Google OAuth client,
then:

```bash
curl -s -X POST "$BASE_URL/api/v1/auth/google" \
  -H "Content-Type: application/json" \
  -d '{"id_token": "<google-id-token>"}'
```

Both routes return: `{"token": "...", "user": {...}}`.

## 3. Sync offers from PubScale (admin)

Sync is admin-gated, so the dev user needs `is_admin = true` first — this is a manual one-time
SQL step by design (see [`README.md`](README.md), there's deliberately no admin-promotion
endpoint):

```bash
# with Docker stack:
docker compose exec -T postgres psql -U postgres -d earnsaga \
  -c "UPDATE users SET is_admin = true WHERE email = 'you@example.com';"

# or against a host DATABASE_URL:
# psql "$DATABASE_URL" -c "UPDATE users SET is_admin = true WHERE email = 'you@example.com';"
```

No need to re-login: `is_admin` isn't baked into the JWT, it's looked up fresh from the DB on every
admin request (`auth.RequireAdmin` → `user.Service.IsAdmin`), so the existing `$TOKEN` from step 2
already works.

```bash
curl -s -X POST -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/admin/sync-offers"
```

```json
{"synced": 42}
```

## 4. List / search offers, get offer detail

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/offers"

curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/offers?search=survey"
```

Grab an offer id from the list response, then:

```bash
OFFER_ID="<id-from-the-list-above>"

curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/offers/$OFFER_ID"
```

```json
{
  "id": "...", "name": "...", "icon_url": "...", "description": "...", "total_payout": 5.00,
  "goals": [{"id":"...", "title":"...", "instructions":"...", "reward": 5.00, "sort_order": 0}],
  "status": "not_started"
}
```

## 5. Start an offer

```bash
curl -s -X POST -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/offers/$OFFER_ID/start"
```

```json
{"status": "in_progress", "redirect_url": "https://advertiser.example/click?uid=<your-user-id>", "already_started": false}
```

`redirect_url` has PubScale's `{your_user_id}` placeholder already substituted with the real user
id — open that URL to continue the advertiser flow. Calling `/start` again on the same offer is
safe and idempotent — it returns `already_started: true` with the current status instead of
erroring or inserting a duplicate row.

## 6. Simulate a PubScale S2S reward callback

This is what PubScale's servers call directly once the user completes an offer. It's public (no
JWT) — authenticated instead by an MD5 signature over `secret_key.user_id.value.token`, where
`value` is truncated to an integer in the signature formula (see
`callback.Service.VerifySignature`).

```bash
SECRET="$PUBSCALE_SECRET_KEY"   # from your .env
VALUE="5"                        # must match int64(value) in the signature below
CB_TOKEN="test-token-$(date +%s)" # unique per "callback" — this is what idempotency keys off

SIGNATURE=$(echo -n "$SECRET.$USER_ID.$VALUE.$CB_TOKEN" | openssl md5 | awk '{print $NF}')

curl -s "$BASE_URL/callbacks/pubscale?user_id=$USER_ID&value=$VALUE&token=$CB_TOKEN&signature=$SIGNATURE"
```

```json
{"status": "credited"}
```

**PowerShell equivalent for the signature** (no `openssl` needed):

```powershell
$secret = $env:PUBSCALE_SECRET_KEY
$userId = "<user-id>"
$value = 5
$cbToken = "test-token-$(Get-Date -UFormat %s)"
$str = "$secret.$userId.$value.$cbToken"
$md5 = [System.Security.Cryptography.MD5]::Create()
$hashBytes = $md5.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($str))
$signature = ($hashBytes | ForEach-Object { $_.ToString("x2") }) -join ""
```

Replay the exact same request (same `$CB_TOKEN`) to confirm idempotency — PubScale retries
callbacks, so this must not double-credit:

```bash
curl -s "$BASE_URL/callbacks/pubscale?user_id=$USER_ID&value=$VALUE&token=$CB_TOKEN&signature=$SIGNATURE"
```

```json
{"status": "duplicate_ignored"}
```

Tamper with the signature to confirm rejection:

```bash
curl -s "$BASE_URL/callbacks/pubscale?user_id=$USER_ID&value=$VALUE&token=$CB_TOKEN&signature=deadbeef"
```

```json
{"error": "invalid signature"}
```
_(HTTP 401)_

## 7. Check wallet balance + transactions

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/users/wallet"
```

```json
{"balance": 5.00}
```

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/users/wallet/transactions"
```

```json
[
  {
    "id": "...", "amount": 5.00, "type": "credit",
    "offer_id": "...", "goal_id": "...", "offer_name": "...",
    "created_at": "2026-07-11T00:00:00Z"
  }
]
```

`offer_id`/`goal_id`/`offer_name` being present (not omitted) confirms the callback in step 6 was
attributed to the offer started in step 5 — this is the reward-value matching from
`callback.selectAttributionMatch` (see [`README.md`](README.md) known-gaps for how this heuristic
works and its limits).

## 8. Leaderboard

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/leaderboard?range=daily"
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/leaderboard?range=weekly"
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/leaderboard?range=alltime"
```

```json
[{"rank": 1, "user_id": "...", "name": "Dev User", "avatar_url": "", "coins": 5.00}]
```

Real-time stream (Server-Sent Events — `-N` disables curl's output buffering so frames show up as
they arrive; leave this running and trigger another callback from another terminal to watch a new
frame push):

```bash
curl -sN -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/leaderboard/stream?range=alltime"
```

## 9. Track an impression/click event

```bash
curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d "{\"type\": \"impression\", \"offer_id\": \"$OFFER_ID\"}" \
  "$BASE_URL/api/v1/events"

curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d "{\"type\": \"click\", \"offer_id\": \"$OFFER_ID\"}" \
  "$BASE_URL/api/v1/events"
```

```json
{"status": "tracked"}
```
_(HTTP 201)_

## 10. Admin analytics report

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/admin/analytics"

# Filtered to a specific offer and date range
curl -s -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/api/v1/admin/analytics?from=2026-07-01&to=2026-07-11&offer_id=$OFFER_ID"
```

```json
{
  "by_date_offer": [
    {"date": "2026-07-11", "offer_id": "...", "offer_name": "...", "impressions": 1, "clicks": 1, "revenue": 5.00}
  ],
  "by_date": [
    {"date": "2026-07-11", "dau": 1}
  ]
}
```

A malformed filter is rejected before touching the database:

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/admin/analytics?from=not-a-date"
```

```json
{"error": "invalid from date, expected YYYY-MM-DD: parsing time \"not-a-date\" ..."}
```
_(HTTP 400)_

---

## Final PDF alignment check

Re-checked against every core requirement in `Fullstack-FTE Assignment.pdf`:

| # | Requirement | Status | Where |
|---|---|---|---|
| 1 | Google sign-in, JWT session | Done | Step 2, `internal/auth` |
| 2 | Fetch & store PubScale offers, re-syncable without duplicates | Done | Step 3, `internal/offer` (upsert on `pubscale_id`) |
| 3 | List offers + search | Done | Step 4, `GET /offers?search=` |
| 4 | Offer detail (name, icon, description, payout, goals) | Done | Step 4, `GET /offers/{id}` |
| 5 | Start an offer once, redirect with tracking URL | Done | Step 5, idempotent `/start` |
| 6 | S2S callback: receive, verify signature, credit wallet, idempotent, 2xx always | Done | Step 6 |
| 7 | Wallet balance + transaction history with offer/goal reference | Done | Step 7 |
| 8 | Leaderboard daily/weekly/all-time, real-time updates | Done | Step 8, Redis + SSE |
| 9 | Analytics: impressions/clicks/revenue/DAU, admin-only | Done | Steps 9–10 |
| 10 | Automated tests | Done | `go test ./...`, see [`README.md`](README.md#testing) |

Backend feature work for the assignment is complete and curl-verified above.

**Still outside this backend codebase** (assignment submission / process, not missing API features):

- **Public deploy** — local Docker stack is ready (`docker compose up --build`); a publicly
  accessible URL is still required for submission.
- **Repo submission** — private GitHub repo + add the collaborators named in the PDF, then email
  repo link + live URL.
