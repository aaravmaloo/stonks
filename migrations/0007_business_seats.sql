ALTER TABLE game.businesses
ADD COLUMN IF NOT EXISTS seat_capacity BIGINT NOT NULL DEFAULT 60000
CHECK (seat_capacity > 0 AND seat_capacity <= 250000);

UPDATE game.businesses
SET seat_capacity = 60000
WHERE seat_capacity IS NULL OR seat_capacity <= 0;
