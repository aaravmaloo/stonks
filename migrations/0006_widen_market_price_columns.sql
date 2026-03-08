DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'game'
          AND table_name = 'stocks'
          AND column_name = 'current_price_micros'
          AND data_type = 'integer'
    ) THEN
        ALTER TABLE game.stocks
        ALTER COLUMN current_price_micros TYPE BIGINT
        USING current_price_micros::BIGINT;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'game'
          AND table_name = 'stocks'
          AND column_name = 'anchor_price_micros'
          AND data_type = 'integer'
    ) THEN
        ALTER TABLE game.stocks
        ALTER COLUMN anchor_price_micros TYPE BIGINT
        USING anchor_price_micros::BIGINT;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'game'
          AND table_name = 'stock_prices'
          AND column_name = 'price_micros'
          AND data_type = 'integer'
    ) THEN
        ALTER TABLE game.stock_prices
        ALTER COLUMN price_micros TYPE BIGINT
        USING price_micros::BIGINT;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'game'
          AND table_name = 'positions'
          AND column_name = 'avg_price_micros'
          AND data_type = 'integer'
    ) THEN
        ALTER TABLE game.positions
        ALTER COLUMN avg_price_micros TYPE BIGINT
        USING avg_price_micros::BIGINT;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'game'
          AND table_name = 'orders'
          AND column_name = 'price_micros'
          AND data_type = 'integer'
    ) THEN
        ALTER TABLE game.orders
        ALTER COLUMN price_micros TYPE BIGINT
        USING price_micros::BIGINT;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'game'
          AND table_name = 'orders'
          AND column_name = 'fee_micros'
          AND data_type = 'integer'
    ) THEN
        ALTER TABLE game.orders
        ALTER COLUMN fee_micros TYPE BIGINT
        USING fee_micros::BIGINT;
    END IF;
END $$;
