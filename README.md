# Stanks (`stk`)

Stanks is a true CLI stock sandbox game written in Go.

- Game name: `stanks`
- CLI command: `stk`
- Currency: `stonky`
- Backend: Local auth + Postgres
- Runtime: Go API + Go worker

## What this repository includes

- `stk` CLI with auth, dashboard, world-state, trading, business, stocks, social, and sync commands.
- `stanks-discord-bot` slash-command Discord bot with embeds and modal-based auth.
- `stanks-api` HTTP backend for game logic.
- `stanks-worker` scheduled market + economy engine.
- Postgres migration for all core schemas/tables.
- Dockerfiles for API and worker.
- Railway deployment config examples.

## Core gameplay

1. `stk signup`
2. `stk login`
3. You start with `25,000 stonky` plus a `2,000 stonky` signup bonus.
4. Read the world before you trade:
   - `stk world`
   - Track the active catalyst, political climate, region drift, and current risk/reward bias.
5. Browse market and trade:
   - `stk stocks list all`
   - `stk stocks list COBOLT`
   - `stk stocks buy COBOLT` (then enter shares in prompt)
   - `stk stocks sell COBOLT` (then enter shares in prompt)
6. Build businesses:
   - `stk business create "Acme Labs"` (then choose visibility in prompt)
   - `stk business visibility <id> public`
   - `stk business ipo <id>` (then enter symbol and price in prompts)
7. Hire and train professionals for business revenue:
   - `stk business employees candidates`
   - `stk business employees hire <business_id> <candidate_id>`
   - `stk business employees hire-many <business_id> <count> <best_value|high_output|low_risk>`
   - `stk business employees train <business_id> <employee_id>`
8. Scale with machines and financing:
   - `stk business machinery buy <business_id> assembly_line`
   - `stk business loans list <business_id>`
   - `stk business loans take <business_id> 50000`
   - `stk business loans repay <business_id> 10000`
9. Build progression:
   - Stack profitable ticks to earn automatic streak rewards.
   - Grow reputation so your empire looks stronger to the market.
   - Lean into risk when the world is hot, and pull back when politics/global markets turn.
10. Own private companies through stakes:
   - `stk stakes`
   - `stk stakes give <business_id> <username> <percent>`
   - Business revenue and bank-sale payouts now flow to every stake holder by ownership percentage.
11. Diversify with mutual funds:
   - `stk funds list`
   - `stk funds buy CORE20 10`
   - `stk funds sell TECH6X 2`
12. Exit a company via bank buyout:
   - `stk business sell <business_id>`
13. Create and list your own stock:
   - `stk stocks create ACMELB` (then enter display name and business id in prompts)
   - `stk stocks ipo ACMELB` (then enter price in prompt)
14. Follow players and compare rankings:
   - `stk friends add <invite_code>`
   - `stk leaderboard global`
   - `stk leaderboard friends`
15. Replay offline writes:
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

- World-state rotation with:
  - Political climates (`steady_hand`, `stimulus_wave`, `tariff_cycle`, `antitrust_wave`, `election_heat`)
  - Mid-term catalysts with a visible tick countdown
  - Global regional drift across Americas / Europe / Asia
  - A risk/reward bias that makes aggressive play pay more or hurt more depending on the moment
- Business revenue credits/debits to owner wallets, now including:
  - Employee salary costs
  - Professional risk drag
  - Machinery output + machinery upkeep
  - Business-loan interest accrual
  - Auto debt servicing every tick (2% of outstanding, floor 250 stonky)
  - Late fees when due amount cannot be paid
  - Delinquency consequences:
    - `>=5` missed ticks: machinery repossession + employee productivity haircut
    - `>=9` missed ticks: forced liquidation (business closed, payout 0)
- Debt interest accrual on negative balances (APR configurable, default `18%`).
- Employee candidate replenishment every tick (`EMPLOYEE_PER_TICK`).
- Optional random stock spawning every tick (`NEW_STOCKS_PER_TICK`) with starting prices below `100 stonky`.

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
- `internal/auth`: Local Postgres-backed auth service.
- `internal/game`: gameplay engine.
- `internal/api`: HTTP handlers + auth middleware.
- `migrations/0001_init.sql`: core DB schema.
- `migrations/0002_business_expansion.sql`: machinery, loans, bank-sale history, fund positions.
- `migrations/0003_loan_consequences.sql`: delinquency tracking column for loan consequences.
- `migrations/0004_business_depth.sql`: strategy/upgrades/brand/health/reserve columns.
- `migrations/0005_active_business.sql`: active business pointer on wallets.
- `migrations/0006_widen_market_price_columns.sql`: repair legacy `INTEGER` market-price columns to `BIGINT`.
- `migrations/0007_business_seats.sql`: per-business seat capacity for employee scaling.
- `migrations/0011_world_progression.sql`: world-state, streak rewards, reputation, and business narrative systems.
- `migrations/0012_business_stakes.sql`: transferable business ownership and passive stake payouts.
- `migrations/0013_hire_many_perf.sql`: indexes for faster bulk-hiring queries.

## Local setup

### Prerequisites

- Go 1.25+
- PostgreSQL server

### Environment variables

Set for API and worker:

```bash
DATABASE_URL=postgres://...
STANKS_API_ADDR=:8080
STANKS_MARKET_TICK_EVERY=5m
EMPLOYEE_PER_TICK=1
NEW_STOCKS_PER_TICK=0
STANKS_INTEREST_APR=0.18
STANKS_STARTUP_SEED_STOCKS=true
```

Set for CLI:

```bash
STK_API_BASE_URL=http://localhost:8080
```

Set for Discord bot:

```bash
DISCORD_BOT_TOKEN=your_bot_token
DISCORD_GUILD_ID=optional_dev_guild_id
```

### Run migration

Apply SQL file:

```bash
psql "$DATABASE_URL" -f migrations/0001_init.sql
psql "$DATABASE_URL" -f migrations/0002_business_expansion.sql
psql "$DATABASE_URL" -f migrations/0003_loan_consequences.sql
psql "$DATABASE_URL" -f migrations/0004_business_depth.sql
psql "$DATABASE_URL" -f migrations/0005_active_business.sql
psql "$DATABASE_URL" -f migrations/0006_widen_market_price_columns.sql
psql "$DATABASE_URL" -f migrations/0007_business_seats.sql
psql "$DATABASE_URL" -f migrations/0008_business_employee_count.sql
psql "$DATABASE_URL" -f migrations/0009_discord_sessions.sql
psql "$DATABASE_URL" -f migrations/0010_auth_users_local_columns.sql
psql "$DATABASE_URL" -f migrations/0011_world_progression.sql
psql "$DATABASE_URL" -f migrations/0012_business_stakes.sql
psql "$DATABASE_URL" -f migrations/0013_hire_many_perf.sql
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

### Run Discord Bot

```bash
go run ./cmd/stanks-discord-bot
```

Notes:

- The bot reads `DISCORD_BOT_TOKEN` and `STK_API_BASE_URL`.
- Set `DISCORD_GUILD_ID` to register commands instantly in one server during development.
- The bot always registers a global command set so slash commands can also be used in DMs with the bot.
- If `DISCORD_GUILD_ID` is empty, commands register globally only.

## CLI command reference

### Auth/session

- `stk signup` (interactive prompts)
- `stk login` (interactive prompts)
- `stk logout`

### Dashboard/sync

- `stk dash`
- `stk world`
- `stk stakes`
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
- `stk business upgrades buy [business_id] [marketing|rd|automation|compliance|seats]`
- `stk business reserve deposit [business_id] [stonky]`
- `stk business reserve withdraw [business_id] [stonky]`

### Stakes

- `stk stakes`
- `stk stakes give [business_id] [username] [percent]`

### Business depth mechanics

- Strategy modes with different risk/reward profiles.
- Upgrade tree (`marketing`, `rd`, `automation`, `compliance`, `seats`) with escalating costs.
- `seats` increases per-business employee capacity in blocks instead of keeping a flat cap.
- Cash reserve treasury per business.
- Reserve yield per tick (higher with R&D levels).
- Reserve auto-shield when business cycle turns negative.
- Brand score and operational-health score affect revenue multipliers.
- Region exposure and company narrative arcs drive core business volatility.
- Narrative focus (`product`, `brand`, `supply`, `talent`, `regulatory`, `finance`) changes how politics and catalysts hit each business.
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

### World + progression mechanics

- `stk world` shows:
  - Current market regime
  - Political climate and policy focus
  - Mid-term catalyst with remaining ticks
  - Global market drift for Americas / Europe / Asia
  - Current risk/reward bias
- `stk dash` now also shows:
  - Reputation title + score
  - Current and best profitable-tick streak
  - Risk appetite score
  - Last risk payout and streak reward
- Streak rewards trigger automatically at profitable streak thresholds.
- Reputation rises on clean profitable runs and falls faster during bad, high-risk stretches.
- Risk appetite is derived from concentration, leverage, listed exposure, and aggressive business posture.
- `stk stakes` shows:
  - Your percentage in each business
  - Revenue share per tick
  - Estimated stake value
  - Cost basis and unrealized P/L
- Stake transfers are controller-driven:
  - Only the controlling owner can give company percentage away
  - The controller must keep some stake
  - Passive holders receive business-cycle payouts and sale proceeds automatically

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

- This repo assumes online-first gameplay; local queue is best-effort retry.
- Auth records are stored in `auth.users` with bcrypt password hashes and bearer tokens.
- Production ops details are in `deploy.md`.
- Business candidate pool now seeds up to 60,000 professionals per season and can grow every tick.
- Discord `/setup` now points players toward `/world`, `/dashboard`, and the new progression loop.
