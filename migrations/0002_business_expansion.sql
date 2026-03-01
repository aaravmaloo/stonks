CREATE TABLE IF NOT EXISTS game.business_machinery (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES game.businesses(id) ON DELETE CASCADE,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    machine_type TEXT NOT NULL,
    level INT NOT NULL DEFAULT 1 CHECK (level >= 1),
    output_bonus_micros BIGINT NOT NULL CHECK (output_bonus_micros >= 0),
    upkeep_micros BIGINT NOT NULL CHECK (upkeep_micros >= 0),
    reliability_bps INT NOT NULL DEFAULT 9400 CHECK (reliability_bps >= 0 AND reliability_bps <= 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (business_id, machine_type)
);

CREATE TABLE IF NOT EXISTS game.business_loans (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES game.businesses(id) ON DELETE CASCADE,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    owner_user_id TEXT NOT NULL,
    principal_micros BIGINT NOT NULL CHECK (principal_micros > 0),
    outstanding_micros BIGINT NOT NULL CHECK (outstanding_micros >= 0),
    interest_bps INT NOT NULL CHECK (interest_bps > 0 AND interest_bps <= 10000),
    missed_ticks INT NOT NULL DEFAULT 0 CHECK (missed_ticks >= 0),
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'repaid', 'sold_off')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game.business_sale_history (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    owner_user_id TEXT NOT NULL,
    gross_valuation_micros BIGINT NOT NULL,
    adjustment_factor NUMERIC(10,4) NOT NULL,
    loan_payoff_micros BIGINT NOT NULL DEFAULT 0,
    payout_micros BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game.fund_positions (
    user_id TEXT NOT NULL,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    fund_code TEXT NOT NULL,
    units BIGINT NOT NULL CHECK (units >= 0),
    avg_nav_micros BIGINT NOT NULL CHECK (avg_nav_micros > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, season_id, fund_code)
);

CREATE INDEX IF NOT EXISTS idx_business_machinery_owner ON game.business_machinery (season_id, business_id);
CREATE INDEX IF NOT EXISTS idx_business_loans_owner ON game.business_loans (season_id, owner_user_id, status);
CREATE INDEX IF NOT EXISTS idx_fund_positions_user ON game.fund_positions (season_id, user_id);
