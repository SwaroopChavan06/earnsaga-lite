# Demo scripts — interview use only, NOT part of the submission

Everything in this folder exists purely to make a live demo look good with
realistic data. None of it is referenced by the actual application, its
migrations, or any test suite — it's disposable tooling, safe to delete
entirely before submitting.

## The 5 demo users

Trivially identifiable and removable — every row this tooling creates is
reachable from `users.google_sub LIKE 'demo-sub-%'`, and nothing here ever
touches a real account.

| Name       | Email                   | google_sub       |
| ---------- | ------------------------ | ---------------- |
| Alice Demo | alice.demo@example.com  | demo-sub-alice   |
| Bob Demo   | bob.demo@example.com    | demo-sub-bob     |
| Carol Demo | carol.demo@example.com  | demo-sub-carol   |
| Dave Demo  | dave.demo@example.com   | demo-sub-dave    |
| Eve Demo   | eve.demo@example.com    | demo-sub-eve     |

Each user: one real offer marked `completed`, one real offer `in_progress`,
and 3 `wallet_transactions` spread across **today / this week / 2+ weeks
ago** — so the daily, weekly, and all-time leaderboards each show a
genuinely different ranking, not three identical copies of the same order.

## Prerequisites

1. Postgres + Redis running from the repo root:
   ```bash
   docker compose up -d postgres redis
   ```
   Postgres is reachable on the **host** at `localhost:5433` (remapped, see
   `docker-compose.yml`); these scripts talk to it via `docker compose exec`
   instead, so you don't need `psql`/`redis-cli` installed locally.
2. Migrations already applied (the API applies them itself on every boot — see
   `backend/internal/db/db.go`) and **at least 2 active
   offers already synced** from PubScale at some point:
   ```bash
   TOKEN=... # admin JWT, see API_TESTING.md
   curl -s -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/admin/sync-offers
   ```
   `seed_demo_data.sql` picks 2-3 *real* active offers from your existing
   `offers` table — it refuses to run (raises an error, no partial writes)
   if fewer than 2 exist.
3. Bash for the `.sh` scripts — Git Bash or WSL on Windows. They only use
   `docker compose exec` under the hood, no other host dependencies.

**Windows/PowerShell note:** PowerShell doesn't support `<` file
redirection the way bash does — every `... -f - < demo/*.sql` command below
will fail with `RedirectionNotSupported` if pasted straight into
PowerShell. Either run it from Git Bash directly, or from PowerShell use:
```powershell
Get-Content -Raw demo/seed_demo_data.sql | docker compose exec -T postgres psql -U postgres -d earnsaga -f -
```
The `.sh` scripts themselves just work from PowerShell via
`& "C:\Program Files\Git\bin\bash.exe" demo/seed_demo_leaderboard.sh`
(or `bash demo/seed_demo_leaderboard.sh` if Git's `bin` is on `PATH`).

## Run order

From the repo root:

```bash
# 1. Seed Postgres — 5 users, offers, and wallet_transactions.
docker compose exec -T postgres psql -U postgres -d earnsaga -f - < demo/seed_demo_data.sql

# 2. Mirror those transactions into Redis (the leaderboard is Redis-only —
#    step 1 alone leaves lb:daily:*/lb:weekly:*/lb:alltime empty).
./demo/seed_demo_leaderboard.sh
```

Both steps are safe to re-run as many times as you want before a demo —
re-running just recomputes the same end state.

## The "watch it change live" moment

With the leaderboard open in the frontend (or the raw SSE stream:
`GET /api/v1/leaderboard/stream?range=alltime`), run:

```bash
./demo/trigger_demo_credit.sh Alice 750
```

This inserts one real `wallet_transactions` row for Alice, pushes the same
amount into Redis, and prints her new balance + all-time rank. If the SSE
stream is open in a browser tab, her rank updates there within ~3 seconds
with **no page refresh** — that's the moment to narrate.

## Resetting between demo runs

```bash
./demo/reset_demo_leaderboard.sh
docker compose exec -T postgres psql -U postgres -d earnsaga -f - < demo/reset_demo_data.sql
```

- `reset_demo_leaderboard.sh` looks up demo user IDs from Postgres by
  `google_sub`, so run it **before** the SQL reset (or note the fallback
  instructions it prints if you run it after).
- `reset_demo_data.sql` only ever deletes rows reachable from
  `users.google_sub LIKE 'demo-sub-%'`.

## Suggested narration script

1. **Show the lifecycle.** Query Postgres directly to show one completed
   offer and one in-progress offer for the same user:
   ```bash
   docker compose exec -T postgres psql -U postgres -d earnsaga -c \
     "SELECT o.name, uo.status, uo.completed_at FROM user_offers uo
      JOIN offers o ON o.id = uo.offer_id JOIN users u ON u.id = uo.user_id
      WHERE u.email = 'alice.demo@example.com';"
   ```
2. **Show Alice's wallet via the real API.** Mint her a token with the
   dev-only endpoint (`ENV=development`) and hit the actual wallet route:
   ```bash
   TOKEN=$(curl -s "http://localhost:8080/api/v1/dev/token?email=alice.demo@example.com" | jq -r .token)
   curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/users/wallet/summary | jq
   ```
3. **Show the leaderboard differs by timeframe** — hit `?range=daily`,
   `?range=weekly`, and `?range=alltime` back to back and point out the
   order changes. This is *why* the seed data is deliberately spread
   across three time buckets instead of everything being "today".
4. **The live moment.** With the SSE stream open, run
   `./demo/trigger_demo_credit.sh Alice 750` and watch her rank move in
   real time, no refresh.
5. **Reset** with the two commands above before the next run-through.

## Notes / caveats

- The weekly bucket is anchored to the most recent **Sunday UTC**
  (`weeklyKey()` in `backend/internal/leaderboard/repository.go`), not
  Go's/ISO's Monday-anchored week. Every timestamp these scripts compute
  uses the same anchor, so the buckets line up with what the real app
  shows.
- If you run `seed_demo_data.sql` in the first few minutes after Sunday
  00:00 UTC, the "today" and "this week" buckets can briefly coincide —
  cosmetic only, not a bug.
- `trigger_demo_credit.sh` restricts its name argument to letters and its
  amount argument to a plain number. This is local demo tooling for a
  trusted operator, not hardened for untrusted input.
- Nothing here modifies `backend/internal/db/migrations/`, the app's schema, or any
  real seed path used by the actual application.
