# Stanks (`stk`)

Stanks is a true CLI stock sandbox game written in Go.

- Game name: `stanks`
- CLI command: `stk`
- Currency: `stonky`
- Backend: Supabase Auth + Supabase Postgres
- Runtime: Go API + Go worker

## What this repository includes

- `stk` CLI with auth, dashboard, trading, business, stocks, social, and sync commands.
- `stanks-api` HTTP backend for game logic.
- `stanks-worker` scheduled market + economy engine.
- Postgres migration for all core schemas/tables.
- Dockerfiles for API and worker.
- Railway deployment config examples.

## Core gameplay

1. `stk signup`
2. `stk login`
3. You start with `25,000 stonky`.
4. Browse market and trade:
   - `stk stocks list all`
   - `stk stocks list COBOLT`
   - `stk stocks buy COBOLT` (then enter shares in prompt)
   - `stk stocks sell COBOLT` (then enter shares in prompt)
5. Build businesses:
   - `stk business create "Acme Labs"` (then choose visibility in prompt)
   - `stk business visibility <id> public`
   - `stk business ipo <id>` (then enter symbol and price in prompts)
6. Hire employees for business revenue:
   - `stk business employees candidates`
   - `stk business employees hire <business_id> <candidate_id>`
7. Create and list your own stock:
   - `stk stocks create ACMELB` (then enter display name and business id in prompts)
   - `stk stocks ipo ACMELB` (then enter price in prompt)
8. Follow players and compare rankings:
   - `stk friends add <invite_code>`
   - `stk leaderboard global`
   - `stk leaderboard friends`
9. Replay offline writes:
   - `stk sync`

## Game rules and constraints

- Ticker symbol format is strict: exactly 6 uppercase chars (`[A-Z]{6}`).
- Trading is spot-only in v1 (no leverage/short/options).
- Business creation unlocks at net worth `>= 250,000 stonky`.
- Debt is allowed but bounded:
  - `debt_limit = clamp(5000, 100000, 35% of peak_net_worth)` in stonky.
- Duplicate mutating requests are blocked with idempotency keys.

## Market algorithm (implemented)

The worker runs every `5m` by default and updates listed stocks with:

- Regime drift (`bull`, `neutral`, `bear`)
- Stochastic noise
- Mean reversion toward anchor price
- Low-probability shock events
- Per-tick clamp at `[-8%, +8%]`

Implemented return shape:

`return = drift(regime) + noise + meanReversion + optionalShock`

Then:

`price_next = max(min_price, price_now * (1 + return))`

Economy tick also applies:

- Business revenue credits to owner wallets.
- Debt interest accrual on negative balances (APR configurable, default `18%`).

## Anti-exploit safety model

- Server-authoritative mutations only.
- Serializable DB transactions for critical actions (orders, hiring, IPO, business create).
- Idempotency table (`game.idempotency_keys`) protects against replay duplicates.
- Money uses fixed-point integer micros (`1 stonky = 1_000_000 micros`).
- Shares use fixed units (`1 share = 10_000 units`).
- Ledger entries recorded for economic mutations.
- CLI offline queue uses idempotency keys and replays via `stk sync`.

## Architecture

- `cmd/stk`: CLI binary.
- `cmd/stanks-api`: HTTP API server.
- `cmd/stanks-worker`: market/economy worker.
- `internal/auth`: Supabase Auth REST integration.
- `internal/game`: gameplay engine.
- `internal/api`: HTTP handlers + auth middleware.
- `migrations/0001_init.sql`: core DB schema.

## Local setup

### Prerequisites

- Go 1.25+
- Supabase project
- Postgres URL from Supabase

### Environment variables

Set for API and worker:

```bash
DATABASE_URL=postgres://...
SUPABASE_URL=https://<project>.supabase.co
SUPABASE_ANON_KEY=<anon-key>
STANKS_API_ADDR=:8080
STANKS_MARKET_TICK_EVERY=5m
STANKS_INTEREST_APR=0.18
STANKS_STARTUP_SEED_STOCKS=true
```

Set for CLI:

```bash
STK_API_BASE_URL=http://localhost:8080
```

### Run migration

Apply SQL file:

```bash
psql "$DATABASE_URL" -f migrations/0001_init.sql
```

### Run services

```bash
go run ./cmd/stanks-api
go run ./cmd/stanks-worker
```

### Run CLI

```bash
go run ./cmd/stk
```

## CLI command reference

### Auth/session

- `stk signup` (interactive prompts)
- `stk login` (interactive prompts)
- `stk logout`

### Dashboard/sync

- `stk dash`
- `stk sync`

### Stocks

- `stk stocks list [all|SYMBOL]`
- `stk stocks buy [symbol]` (interactive quantity prompt)
- `stk stocks sell [symbol]` (interactive quantity prompt)
- `stk stocks create [symbol]` (interactive display name + business id prompts)
- `stk stocks ipo [symbol]` (interactive price prompt)

Alias:

- `stk stock ...` works as alias for `stk stocks`.

### Business

- `stk business create [name]` (interactive visibility prompt)
- `stk business state [business_id]`
- `stk business visibility [business_id] [private|public]`
- `stk business ipo [business_id]` (interactive symbol + price prompts)
- `stk business employees list [business_id]`
- `stk business employees candidates`
- `stk business employees hire [business_id] [candidate_id]`

Alias:

- `stk bussin ...` works as alias for `stk business`.

### Social/competition

- `stk leaderboard global`
- `stk leaderboard friends`
- `stk friends add [invite_code]`
- `stk friends remove [invite_code]`

## Persistence and sync behavior

- Session token stored in `~/.stk/session.json`.
- Offline queued mutations stored in `~/.stk/queue.json`.
- On network failure (non-API failure), mutating commands are queued automatically.
- `stk sync` retries queued commands in order.

## Included stock universe (seeded)

`COBOLT`, `NIMBUS`, `RUSTIC`, `PYLONS`, `JAVOLT`, `SWIFTR`, `KOTLIN`, `NODEON`, `RUBYIX`, `ELIXIR`, `QUARKX`, `VECTRA`, `DATUMX`, `CYBRON`, `FUSION`, `NEBULA`, `ORBITZ`, `ZENITH`, `ARCANE`, `LUMINA`.

## Testing

Run:

```bash
go test ./...
go vet ./...
```

Current test coverage includes:

- Symbol validation rules.
- Debt limit boundary logic.
- Notional calculation fixed-point behavior.

## Important notes

- Supabase project/server provisioning is intentionally manual.
- This repo assumes online-first gameplay; local queue is best-effort retry.
- Email verification behavior follows Supabase project auth configuration.
- Production ops details are in `deploy.md`.
