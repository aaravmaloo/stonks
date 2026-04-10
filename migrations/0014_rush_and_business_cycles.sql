ALTER TABLE game.businesses
ADD COLUMN IF NOT EXISTS cycle_phase TEXT NOT NULL DEFAULT 'stable'
    CHECK (cycle_phase IN ('stable', 'boom', 'slump', 'recovery', 'squeeze')),
ADD COLUMN IF NOT EXISTS cycle_ticks_remaining INT NOT NULL DEFAULT 4
    CHECK (cycle_ticks_remaining >= 0),
ADD COLUMN IF NOT EXISTS cycle_impact_bps INT NOT NULL DEFAULT 0
    CHECK (cycle_impact_bps BETWEEN -6000 AND 6000);

CREATE TABLE IF NOT EXISTS game.rush_progress (
    user_id TEXT NOT NULL,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    current_streak INT NOT NULL DEFAULT 0 CHECK (current_streak >= 0),
    best_streak INT NOT NULL DEFAULT 0 CHECK (best_streak >= 0),
    round_count INT NOT NULL DEFAULT 0 CHECK (round_count >= 0),
    win_count INT NOT NULL DEFAULT 0 CHECK (win_count >= 0),
    vault_points INT NOT NULL DEFAULT 0 CHECK (vault_points >= 0),
    vault_level INT NOT NULL DEFAULT 0 CHECK (vault_level >= 0),
    last_mode TEXT NOT NULL DEFAULT 'steady'
        CHECK (last_mode IN ('steady', 'surge', 'apex')),
    last_multiplier_bps INT NOT NULL DEFAULT 10000 CHECK (last_multiplier_bps >= 0),
    last_payout_micros BIGINT NOT NULL DEFAULT 0,
    last_vault_reward_micros BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, season_id)
);
