ALTER TABLE game.business_loans
ADD COLUMN IF NOT EXISTS missed_ticks INT NOT NULL DEFAULT 0 CHECK (missed_ticks >= 0);
