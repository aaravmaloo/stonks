CREATE SCHEMA IF NOT EXISTS users;
CREATE SCHEMA IF NOT EXISTS game;

CREATE TABLE IF NOT EXISTS users.profiles (
    user_id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    username TEXT NOT NULL UNIQUE,
    invite_code TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game.seasons (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'completed')),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game.wallets (
    user_id TEXT NOT NULL,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    balance_micros BIGINT NOT NULL,
    peak_net_worth_micros BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, season_id)
);

CREATE TABLE IF NOT EXISTS game.businesses (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    visibility TEXT NOT NULL CHECK (visibility IN ('private', 'public')),
    is_listed BOOLEAN NOT NULL DEFAULT false,
    stock_symbol CHAR(6),
    base_revenue_micros BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game.stocks (
    id BIGSERIAL PRIMARY KEY,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    symbol CHAR(6) NOT NULL,
    display_name TEXT NOT NULL,
    listed_public BOOLEAN NOT NULL DEFAULT false,
    current_price_micros BIGINT NOT NULL,
    anchor_price_micros BIGINT NOT NULL,
    created_by_user_id TEXT,
    business_id BIGINT REFERENCES game.businesses(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (season_id, symbol)
);

CREATE TABLE IF NOT EXISTS game.stock_prices (
    id BIGSERIAL PRIMARY KEY,
    stock_id BIGINT NOT NULL REFERENCES game.stocks(id) ON DELETE CASCADE,
    tick_at TIMESTAMPTZ NOT NULL,
    price_micros BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS game.positions (
    user_id TEXT NOT NULL,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    stock_id BIGINT NOT NULL REFERENCES game.stocks(id) ON DELETE CASCADE,
    quantity_units BIGINT NOT NULL CHECK (quantity_units > 0),
    avg_price_micros BIGINT NOT NULL CHECK (avg_price_micros > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, season_id, stock_id)
);

CREATE TABLE IF NOT EXISTS game.orders (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    stock_id BIGINT NOT NULL REFERENCES game.stocks(id) ON DELETE CASCADE,
    side TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    quantity_units BIGINT NOT NULL CHECK (quantity_units > 0),
    price_micros BIGINT NOT NULL CHECK (price_micros > 0),
    fee_micros BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game.employee_candidates (
    id BIGSERIAL PRIMARY KEY,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    full_name TEXT NOT NULL,
    role TEXT NOT NULL,
    trait TEXT NOT NULL,
    hire_cost_micros BIGINT NOT NULL,
    revenue_per_tick_micros BIGINT NOT NULL,
    risk_bps INT NOT NULL CHECK (risk_bps >= 0 AND risk_bps <= 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game.business_employees (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES game.businesses(id) ON DELETE CASCADE,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    source_candidate_id BIGINT REFERENCES game.employee_candidates(id) ON DELETE SET NULL,
    full_name TEXT NOT NULL,
    role TEXT NOT NULL,
    trait TEXT NOT NULL,
    revenue_per_tick_micros BIGINT NOT NULL,
    risk_bps INT NOT NULL CHECK (risk_bps >= 0 AND risk_bps <= 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (business_id, source_candidate_id)
);

CREATE TABLE IF NOT EXISTS game.friend_follows (
    follower_user_id TEXT NOT NULL,
    followee_user_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (follower_user_id, followee_user_id)
);

CREATE TABLE IF NOT EXISTS game.idempotency_keys (
    user_id TEXT NOT NULL,
    key TEXT NOT NULL,
    action TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, key)
);

CREATE TABLE IF NOT EXISTS game.ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    tx_group_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    account TEXT NOT NULL,
    delta_micros BIGINT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game.market_state (
    season_id BIGINT PRIMARY KEY REFERENCES game.seasons(id) ON DELETE CASCADE,
    regime TEXT NOT NULL CHECK (regime IN ('bull', 'neutral', 'bear')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game.sync_replays (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game.moderation_flags (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    reason TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_stocks_season_listed ON game.stocks (season_id, listed_public);
CREATE INDEX IF NOT EXISTS idx_stock_prices_stock_tick ON game.stock_prices (stock_id, tick_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_user_season ON game.orders (user_id, season_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_business_owner_season ON game.businesses (owner_user_id, season_id);
CREATE INDEX IF NOT EXISTS idx_wallets_season_balance ON game.wallets (season_id, balance_micros DESC);
CREATE INDEX IF NOT EXISTS idx_positions_user_season ON game.positions (user_id, season_id);
