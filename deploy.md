# Deployment Guide: Supabase + Railway (API + Worker)

This project does **not** auto-provision Supabase.  
Follow this document to set up manually, then deploy to Railway.

## 1. Supabase setup (manual)

1. Create a new Supabase project.
2. In `Project Settings -> API`, copy:
   - `Project URL` -> `SUPABASE_URL`
   - `anon public key` -> `SUPABASE_ANON_KEY`
3. In `Project Settings -> Database`, copy:
   - Connection string -> `DATABASE_URL`
4. In `Authentication -> Providers`, keep Email enabled.
5. In `Authentication -> Email`, configure verification policy.

## 2. Apply database migration

Run:

```bash
psql "$DATABASE_URL" -f migrations/0001_init.sql
psql "$DATABASE_URL" -f migrations/0002_business_expansion.sql
psql "$DATABASE_URL" -f migrations/0003_loan_consequences.sql
psql "$DATABASE_URL" -f migrations/0004_business_depth.sql
```

Verify tables exist:

```sql
SELECT table_schema, table_name
FROM information_schema.tables
WHERE table_schema IN ('users', 'game')
ORDER BY table_schema, table_name;
```

## 3. Railway project layout

Create one Railway project with two services:

- `stanks-api`
- `stanks-worker`

Both services use the same repo and env vars.

## 4. Deploy API service (`stanks-api`)

1. New Railway service from repo.
2. Set Dockerfile path to `Dockerfile.api`.
3. Optionally use template file: `deploy/railway-api.toml`.
4. Set environment variables:

```bash
DATABASE_URL=postgres://...
SUPABASE_URL=https://<project>.supabase.co
SUPABASE_ANON_KEY=<anon-key>
STANKS_API_ADDR=:8080
STANKS_MARKET_TICK_EVERY=5m
STANKS_INTEREST_APR=0.18
STANKS_STARTUP_SEED_STOCKS=true
```

5. Set healthcheck path to `/healthz`.
6. Deploy.

## 5. Deploy worker service (`stanks-worker`)

1. New Railway service from repo.
2. Set Dockerfile path to `Dockerfile.worker`.
3. Optionally use template file: `deploy/railway-worker.toml`.
4. Set same env vars as API:

```bash
DATABASE_URL=postgres://...
SUPABASE_URL=https://<project>.supabase.co
SUPABASE_ANON_KEY=<anon-key>
STANKS_MARKET_TICK_EVERY=5m
STANKS_INTEREST_APR=0.18
STANKS_STARTUP_SEED_STOCKS=true
```

5. Deploy.

## 6. Worker schedule options

### Option A: always-on worker loop (current default)

- Worker process runs continuously and ticks every `STANKS_MARKET_TICK_EVERY`.
- No external cron required.

### Option B: Railway cron with run-once mode

Set:

```bash
STANKS_WORKER_RUN_ONCE=true
```

Then run service on a Railway cron (every 5 minutes) instead of always-on.

## 7. Railway secrets checklist

Set these as Railway Variables (never commit secrets):

- `DATABASE_URL`
- `SUPABASE_URL`
- `SUPABASE_ANON_KEY`

Optional tuning:

- `STANKS_API_ADDR` (API only)
- `STANKS_MARKET_TICK_EVERY`
- `STANKS_INTEREST_APR`
- `STANKS_STARTUP_SEED_STOCKS`
- `STANKS_WORKER_RUN_ONCE` (cron mode only)

## 8. Post-deploy verification

1. Hit API health:

```bash
curl https://<api-domain>/healthz
```

Expect:

```json
{"ok":true}
```

2. Run CLI signup/login against deployed API:

```bash
export STK_API_BASE_URL=https://<api-domain>
stk signup
stk login
stk dash
```

3. Check worker logs for market ticks.

## 9. Rollback strategy

1. Keep prior Railway deployment available.
2. If regression occurs:
   - Roll back Railway service to previous deployment.
   - Disable worker temporarily to stop new ticks.
3. If schema issue exists:
   - Hotfix with forward migration (preferred) rather than destructive rollback.

## 10. Common failure modes

- `401 invalid token`: wrong Supabase URL/key or expired token.
- `db connect failed`: invalid `DATABASE_URL` or network restrictions.
- Missing tables: migration not applied.
- Orders failing with idempotency conflict: duplicate request key replayed.

## 11. Minimal production hardening checklist

- Enable Supabase auth email verification rules intentionally.
- Enforce strong passwords in auth policy.
- Add Railway alerting on API/worker failures.
- Add log drains and metrics/trace backend.
- Restrict who can change Railway variables.
