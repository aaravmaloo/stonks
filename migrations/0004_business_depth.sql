ALTER TABLE game.businesses
ADD COLUMN IF NOT EXISTS strategy TEXT NOT NULL DEFAULT 'balanced' CHECK (strategy IN ('aggressive', 'balanced', 'defensive'));

ALTER TABLE game.businesses
ADD COLUMN IF NOT EXISTS marketing_level INT NOT NULL DEFAULT 0 CHECK (marketing_level >= 0);

ALTER TABLE game.businesses
ADD COLUMN IF NOT EXISTS rd_level INT NOT NULL DEFAULT 0 CHECK (rd_level >= 0);

ALTER TABLE game.businesses
ADD COLUMN IF NOT EXISTS automation_level INT NOT NULL DEFAULT 0 CHECK (automation_level >= 0);

ALTER TABLE game.businesses
ADD COLUMN IF NOT EXISTS compliance_level INT NOT NULL DEFAULT 0 CHECK (compliance_level >= 0);

ALTER TABLE game.businesses
ADD COLUMN IF NOT EXISTS brand_bps INT NOT NULL DEFAULT 10000 CHECK (brand_bps >= 5000 AND brand_bps <= 20000);

ALTER TABLE game.businesses
ADD COLUMN IF NOT EXISTS operational_health_bps INT NOT NULL DEFAULT 10000 CHECK (operational_health_bps >= 5000 AND operational_health_bps <= 15000);

ALTER TABLE game.businesses
ADD COLUMN IF NOT EXISTS cash_reserve_micros BIGINT NOT NULL DEFAULT 0 CHECK (cash_reserve_micros >= 0);

ALTER TABLE game.businesses
ADD COLUMN IF NOT EXISTS last_event TEXT NOT NULL DEFAULT '';
