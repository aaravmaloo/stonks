-- Add active_business_id to game.wallets to track the user's primary business.
ALTER TABLE game.wallets
ADD COLUMN active_business_id BIGINT REFERENCES game.businesses(id) ON DELETE SET NULL;
