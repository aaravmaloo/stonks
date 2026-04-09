CREATE TABLE IF NOT EXISTS game.business_stakes (
    business_id BIGINT NOT NULL REFERENCES game.businesses(id) ON DELETE CASCADE,
    season_id BIGINT NOT NULL REFERENCES game.seasons(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    stake_bps INT NOT NULL CHECK (stake_bps > 0 AND stake_bps <= 10000),
    cost_basis_micros BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (business_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_business_stakes_user ON game.business_stakes (user_id, season_id);

INSERT INTO game.business_stakes (business_id, season_id, user_id, stake_bps, cost_basis_micros)
SELECT b.id, b.season_id, b.owner_user_id, 10000, 0
FROM game.businesses b
WHERE NOT EXISTS (
    SELECT 1
    FROM game.business_stakes s
    WHERE s.business_id = b.id
      AND s.user_id = b.owner_user_id
);
