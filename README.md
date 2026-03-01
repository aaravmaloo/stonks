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
6. Hire and train professionals for business revenue:
   - `stk business employees candidates`
   - `stk business employees hire <business_id> <candidate_id>`
   - `stk business employees train <business_id> <employee_id>`
7. Scale with machines and financing:
   - `stk business machinery buy <business_id> assembly_line`
   - `stk business loans list <business_id>`
   - `stk business loans take <business_id> 50000`
   - `stk business loans repay <business_id> 10000`
8. Diversify with mutual funds:
   - `stk funds list`
   - `stk funds buy CORE20 10`
   - `stk funds sell TECH6X 2`
9. Exit a company via bank buyout:
   - `stk business sell <business_id>`
10. Create and list your own stock:
   - `stk stocks create ACMELB` (then enter display name and business id in prompts)
   - `stk stocks ipo ACMELB` (then enter price in prompt)
11. Follow players and compare rankings:
   - `stk friends add <invite_code>`
   - `stk leaderboard global`
   - `stk leaderboard friends`
12. Replay offline writes:
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
- Drifting anchor prices (not fixed to seed)
- Mean reversion toward the moving anchor
- Jump shocks and extreme tail shocks
- One-sided downside guardrail per tick (to avoid hard-zero crashes), with no hard upside clamp

Implemented return shape:

`return = drift(regime) + noise + meanReversion + optionalShock + optionalExtremeShock`

Then:

`price_next = clamp(min_price, max_price, price_now * exp(return))`

Economy tick also applies:

- Business revenue credits/debits to owner wallets, now including:
  - Professional risk drag
  - Machinery output + machinery upkeep
  - Business-loan interest accrual
  - Auto debt servicing every tick (2% of outstanding, floor 250 stonky)
  - Late fees when due amount cannot be paid
  - Delinquency consequences:
    - `>=5` missed ticks: machinery repossession + employee productivity haircut
    - `>=9` missed ticks: forced liquidation (business closed, payout 0)
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
- `migrations/0002_business_expansion.sql`: machinery, loans, bank-sale history, fund positions.
- `migrations/0003_loan_consequences.sql`: delinquency tracking column for loan consequences.
- `migrations/0004_business_depth.sql`: strategy/upgrades/brand/health/reserve columns.

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
psql "$DATABASE_URL" -f migrations/0002_business_expansion.sql
psql "$DATABASE_URL" -f migrations/0003_loan_consequences.sql
psql "$DATABASE_URL" -f migrations/0004_business_depth.sql
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

### Funds

- `stk funds` (guided flow; prompts action and inputs)
- `stk funds list`
- `stk funds buy [TECH6X|CORE20|VOLT10|DIVMAX|AIEDGE|STABLE] [shares]`
- `stk funds sell [TECH6X|CORE20|VOLT10|DIVMAX|AIEDGE|STABLE] [shares]`

### Business

- `stk business` (guided flow; prompts action and inputs)
- `stk business create [name]` (interactive visibility prompt)
- `stk business state [business_id]`
- `stk business visibility [business_id] [private|public]`
- `stk business ipo [business_id]` (interactive symbol + price prompts)
- `stk business sell [business_id]`
- `stk business employees list [business_id]`
- `stk business employees candidates`
- `stk business employees hire [business_id] [candidate_id]`
- `stk business employees train [business_id] [employee_id]`
- `stk business machinery list [business_id]`
- `stk business machinery buy [business_id] [machine_type]`
- `stk business loans take [business_id] [stonky]`
- `stk business loans repay [business_id] [stonky]`
- `stk business loans list [business_id]`
- `stk business strategy [business_id] [aggressive|balanced|defensive]`
- `stk business upgrades buy [business_id] [marketing|rd|automation|compliance]`
- `stk business reserve deposit [business_id] [stonky]`
- `stk business reserve withdraw [business_id] [stonky]`

### Business depth mechanics

- Strategy modes with different risk/reward profiles.
- Upgrade tree (`marketing`, `rd`, `automation`, `compliance`) with escalating costs.
- Cash reserve treasury per business.
- Reserve yield per tick (higher with R&D levels).
- Reserve auto-shield when business cycle turns negative.
- Brand score and operational-health score affect revenue multipliers.
- Public visibility bonus and listed-company prestige bonus.
- Employee diminishing returns at high headcount.
- Upgrade burn-rate operating costs each tick.
- Aggressive-mode burnout chance reducing employee output.
- Low-brand poaching chance (random employee attrition).
- Viral breakout events boosting brand, health, and tick revenue.
- PR crisis events reducing brand, health, and tick revenue.
- Compliance reduces effective risk penalty.
- Automation boosts machine output and lowers upkeep.
- Marketing and R&D revenue multipliers.
- Loan payment auto-debit each tick.
- Missed-loan late fees and delinquency tracking.
- Delinquency escalation: repossession/productivity haircut/forced liquidation.

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
- Business candidate pool now seeds up to 50 professionals per season.
