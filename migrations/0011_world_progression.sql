ALTER TABLE game.market_state
ADD COLUMN IF NOT EXISTS political_climate TEXT NOT NULL DEFAULT 'steady_hand'
    CHECK (political_climate IN ('steady_hand', 'stimulus_wave', 'tariff_cycle', 'antitrust_wave', 'election_heat')),
ADD COLUMN IF NOT EXISTS policy_focus TEXT NOT NULL DEFAULT 'broad_market'
    CHECK (policy_focus IN ('broad_market', 'technology', 'energy', 'finance', 'consumer', 'healthcare')),
ADD COLUMN IF NOT EXISTS catalyst_name TEXT NOT NULL DEFAULT 'Earnings Window',
ADD COLUMN IF NOT EXISTS catalyst_summary TEXT NOT NULL DEFAULT 'The market is waiting for the next earnings cycle.',
ADD COLUMN IF NOT EXISTS catalyst_ticks_remaining INT NOT NULL DEFAULT 6 CHECK (catalyst_ticks_remaining >= 0),
ADD COLUMN IF NOT EXISTS headline TEXT NOT NULL DEFAULT 'Markets are calm.',
ADD COLUMN IF NOT EXISTS americas_bps INT NOT NULL DEFAULT 0 CHECK (americas_bps BETWEEN -4000 AND 4000),
ADD COLUMN IF NOT EXISTS europe_bps INT NOT NULL DEFAULT 0 CHECK (europe_bps BETWEEN -4000 AND 4000),
ADD COLUMN IF NOT EXISTS asia_bps INT NOT NULL DEFAULT 0 CHECK (asia_bps BETWEEN -4000 AND 4000),
ADD COLUMN IF NOT EXISTS risk_reward_bias_bps INT NOT NULL DEFAULT 0 CHECK (risk_reward_bias_bps BETWEEN -4000 AND 4000);

ALTER TABLE game.businesses
ADD COLUMN IF NOT EXISTS primary_region TEXT NOT NULL DEFAULT 'americas'
    CHECK (primary_region IN ('americas', 'europe', 'asia')),
ADD COLUMN IF NOT EXISTS narrative_arc TEXT NOT NULL DEFAULT 'steady'
    CHECK (narrative_arc IN ('steady', 'breakout', 'expansion', 'fragile', 'turnaround', 'defensive')),
ADD COLUMN IF NOT EXISTS narrative_focus TEXT NOT NULL DEFAULT 'product'
    CHECK (narrative_focus IN ('product', 'brand', 'supply', 'talent', 'regulatory', 'finance')),
ADD COLUMN IF NOT EXISTS narrative_pressure_bps INT NOT NULL DEFAULT 0 CHECK (narrative_pressure_bps BETWEEN 0 AND 12000);

CREATE TABLE IF NOT EXISTS game.player_progress (
    user_id TEXT NOT NULL,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    reputation_score INT NOT NULL DEFAULT 5000 CHECK (reputation_score BETWEEN 0 AND 10000),
    current_profit_streak INT NOT NULL DEFAULT 0 CHECK (current_profit_streak >= 0),
    best_profit_streak INT NOT NULL DEFAULT 0 CHECK (best_profit_streak >= 0),
    streak_reward_tier INT NOT NULL DEFAULT 0 CHECK (streak_reward_tier >= 0),
    risk_appetite_bps INT NOT NULL DEFAULT 0 CHECK (risk_appetite_bps BETWEEN 0 AND 10000),
    last_net_worth_micros BIGINT NOT NULL DEFAULT 0,
    last_net_worth_delta_micros BIGINT NOT NULL DEFAULT 0,
    last_risk_payout_micros BIGINT NOT NULL DEFAULT 0,
    last_streak_reward_micros BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, season_id)
);

CREATE TABLE IF NOT EXISTS game.world_events (
    id BIGSERIAL PRIMARY KEY,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    category TEXT NOT NULL,
    headline TEXT NOT NULL,
    impact_summary TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_world_events_season_created ON game.world_events (season_id, created_at DESC);
