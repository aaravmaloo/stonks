package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"stanks/internal/game"

	"github.com/jackc/pgx/v5"
)

func (s *adminStore) activeSeasonID(ctx context.Context) (int64, error) {
	var seasonID int64
	err := s.db.QueryRow(ctx, `
		SELECT id
		FROM game.seasons
		WHERE status = 'active'
		ORDER BY starts_at DESC, id DESC
		LIMIT 1
	`).Scan(&seasonID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("no active season found")
		}
		return 0, err
	}
	return seasonID, nil
}

func (s *adminStore) listPlayers(ctx context.Context, query string) ([]playerRow, error) {
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return nil, err
	}
	like := "%"
	if q := strings.TrimSpace(query); q != "" {
		like = "%" + q + "%"
	}
	rows, err := s.db.Query(ctx, `
		SELECT p.user_id, p.email, p.username, p.invite_code,
		       COALESCE(w.balance_micros, 0),
		       COALESCE(w.peak_net_worth_micros, 0),
		       w.active_business_id
		FROM users.profiles p
		LEFT JOIN game.wallets w
		  ON w.user_id = p.user_id AND w.season_id = $1
		WHERE $2 = '%'
		   OR p.user_id ILIKE $2
		   OR p.email ILIKE $2
		   OR p.username ILIKE $2
		   OR p.invite_code ILIKE $2
		ORDER BY p.username, p.user_id
		LIMIT 250
	`, seasonID, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []playerRow
	for rows.Next() {
		var row playerRow
		if err := rows.Scan(
			&row.UserID,
			&row.Email,
			&row.Username,
			&row.InviteCode,
			&row.BalanceMicros,
			&row.PeakNetWorthMicros,
			&row.ActiveBusinessID,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *adminStore) playerByID(ctx context.Context, userID string) (playerRow, error) {
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return playerRow{}, err
	}
	var row playerRow
	err = s.db.QueryRow(ctx, `
		SELECT p.user_id, p.email, p.username, p.invite_code,
		       COALESCE(w.balance_micros, 0),
		       COALESCE(w.peak_net_worth_micros, 0),
		       w.active_business_id
		FROM users.profiles p
		LEFT JOIN game.wallets w
		  ON w.user_id = p.user_id AND w.season_id = $2
		WHERE p.user_id = $1
	`, strings.TrimSpace(userID), seasonID).Scan(
		&row.UserID,
		&row.Email,
		&row.Username,
		&row.InviteCode,
		&row.BalanceMicros,
		&row.PeakNetWorthMicros,
		&row.ActiveBusinessID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return playerRow{}, fmt.Errorf("player %q not found", userID)
		}
		return playerRow{}, err
	}
	return row, nil
}

func (s *adminStore) ensureWallet(ctx context.Context, tx pgx.Tx, userID string, seasonID int64) error {
	tag, err := tx.Exec(ctx, `
		INSERT INTO game.wallets (user_id, season_id, balance_micros, peak_net_worth_micros)
		SELECT p.user_id, $2, 0, 0
		FROM users.profiles p
		WHERE p.user_id = $1
		ON CONFLICT (user_id, season_id) DO NOTHING
	`, userID, seasonID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users.profiles WHERE user_id = $1)`, userID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("player %q not found", userID)
		}
	}
	return nil
}

func (s *adminStore) changeBalance(ctx context.Context, userID string, deltaMicros int64) (playerRow, error) {
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return playerRow{}, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return playerRow{}, err
	}
	defer tx.Rollback(ctx)

	userID = strings.TrimSpace(userID)
	if err := s.ensureWallet(ctx, tx, userID, seasonID); err != nil {
		return playerRow{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = balance_micros + $3,
		    updated_at = now()
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID, deltaMicros); err != nil {
		return playerRow{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return playerRow{}, err
	}
	return s.playerByID(ctx, userID)
}

func (s *adminStore) setBalance(ctx context.Context, userID string, amountMicros int64) (playerRow, error) {
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return playerRow{}, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return playerRow{}, err
	}
	defer tx.Rollback(ctx)

	userID = strings.TrimSpace(userID)
	if err := s.ensureWallet(ctx, tx, userID, seasonID); err != nil {
		return playerRow{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = $3,
		    updated_at = now()
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID, amountMicros); err != nil {
		return playerRow{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return playerRow{}, err
	}
	return s.playerByID(ctx, userID)
}

func (s *adminStore) changePeak(ctx context.Context, userID string, deltaMicros int64) (playerRow, error) {
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return playerRow{}, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return playerRow{}, err
	}
	defer tx.Rollback(ctx)

	userID = strings.TrimSpace(userID)
	if err := s.ensureWallet(ctx, tx, userID, seasonID); err != nil {
		return playerRow{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET peak_net_worth_micros = peak_net_worth_micros + $3,
		    updated_at = now()
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID, deltaMicros); err != nil {
		return playerRow{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return playerRow{}, err
	}
	return s.playerByID(ctx, userID)
}

func (s *adminStore) setPeak(ctx context.Context, userID string, amountMicros int64) (playerRow, error) {
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return playerRow{}, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return playerRow{}, err
	}
	defer tx.Rollback(ctx)

	userID = strings.TrimSpace(userID)
	if err := s.ensureWallet(ctx, tx, userID, seasonID); err != nil {
		return playerRow{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET peak_net_worth_micros = $3,
		    updated_at = now()
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID, amountMicros); err != nil {
		return playerRow{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return playerRow{}, err
	}
	return s.playerByID(ctx, userID)
}

func (s *adminStore) setActiveBusiness(ctx context.Context, userID string, businessID int64) (playerRow, error) {
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return playerRow{}, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return playerRow{}, err
	}
	defer tx.Rollback(ctx)

	userID = strings.TrimSpace(userID)
	if err := s.ensureWallet(ctx, tx, userID, seasonID); err != nil {
		return playerRow{}, err
	}
	if businessID == 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE game.wallets
			SET active_business_id = NULL, updated_at = now()
			WHERE user_id = $1 AND season_id = $2
		`, userID, seasonID); err != nil {
			return playerRow{}, err
		}
	} else {
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM game.businesses
				WHERE id = $1 AND owner_user_id = $2 AND season_id = $3
			)
		`, businessID, userID, seasonID).Scan(&exists); err != nil {
			return playerRow{}, err
		}
		if !exists {
			return playerRow{}, fmt.Errorf("business %d not found for player %s", businessID, userID)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE game.wallets
			SET active_business_id = $3, updated_at = now()
			WHERE user_id = $1 AND season_id = $2
		`, userID, seasonID, businessID); err != nil {
			return playerRow{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return playerRow{}, err
	}
	return s.playerByID(ctx, userID)
}

func (s *adminStore) listBusinessesByUser(ctx context.Context, userID string) ([]businessRow, error) {
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, name, visibility, is_listed, base_revenue_micros
		FROM game.businesses
		WHERE owner_user_id = $1 AND season_id = $2
		ORDER BY id
	`, strings.TrimSpace(userID), seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []businessRow
	for rows.Next() {
		var row businessRow
		if err := rows.Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *adminStore) setBusinessName(ctx context.Context, businessID int64, name string) (businessRow, error) {
	var row businessRow
	err := s.db.QueryRow(ctx, `
		UPDATE game.businesses
		SET name = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, name, visibility, is_listed, base_revenue_micros
	`, businessID, strings.TrimSpace(name)).Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros)
	if err != nil {
		if err == pgx.ErrNoRows {
			return businessRow{}, fmt.Errorf("business %d not found", businessID)
		}
		return businessRow{}, err
	}
	return row, nil
}

func (s *adminStore) setBusinessVisibility(ctx context.Context, businessID int64, visibility string) (businessRow, error) {
	if visibility != "private" && visibility != "public" {
		return businessRow{}, fmt.Errorf("visibility must be private or public")
	}
	var row businessRow
	err := s.db.QueryRow(ctx, `
		UPDATE game.businesses
		SET visibility = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, name, visibility, is_listed, base_revenue_micros
	`, businessID, visibility).Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros)
	if err != nil {
		if err == pgx.ErrNoRows {
			return businessRow{}, fmt.Errorf("business %d not found", businessID)
		}
		return businessRow{}, err
	}
	return row, nil
}

func (s *adminStore) setBusinessListed(ctx context.Context, businessID int64, listed bool) (businessRow, error) {
	var row businessRow
	err := s.db.QueryRow(ctx, `
		UPDATE game.businesses
		SET is_listed = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, name, visibility, is_listed, base_revenue_micros
	`, businessID, listed).Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros)
	if err != nil {
		if err == pgx.ErrNoRows {
			return businessRow{}, fmt.Errorf("business %d not found", businessID)
		}
		return businessRow{}, err
	}
	return row, nil
}

func (s *adminStore) setBusinessRevenue(ctx context.Context, businessID int64, amountMicros int64) (businessRow, error) {
	var row businessRow
	err := s.db.QueryRow(ctx, `
		UPDATE game.businesses
		SET base_revenue_micros = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, name, visibility, is_listed, base_revenue_micros
	`, businessID, amountMicros).Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros)
	if err != nil {
		if err == pgx.ErrNoRows {
			return businessRow{}, fmt.Errorf("business %d not found", businessID)
		}
		return businessRow{}, err
	}
	return row, nil
}

func (s *adminStore) deleteBusiness(ctx context.Context, businessID int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM game.businesses WHERE id = $1`, businessID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("business %d not found", businessID)
	}
	fmt.Printf("Business %d deleted.\n", businessID)
	return nil
}

func (s *adminStore) listPositionsByUser(ctx context.Context, userID string) ([]positionRow, error) {
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT st.symbol, st.display_name, p.quantity_units, p.avg_price_micros
		FROM game.positions p
		JOIN game.stocks st ON st.id = p.stock_id
		WHERE p.user_id = $1 AND p.season_id = $2
		ORDER BY st.symbol
	`, strings.TrimSpace(userID), seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []positionRow
	for rows.Next() {
		var row positionRow
		if err := rows.Scan(&row.Symbol, &row.DisplayName, &row.QuantityUnits, &row.AvgPriceMicros); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *adminStore) setPosition(ctx context.Context, userID, symbol string, shares float64, avgPriceMicros int64) (positionRow, error) {
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return positionRow{}, err
	}
	if err := game.ValidateSymbol(symbol); err != nil {
		return positionRow{}, err
	}
	units, err := game.SharesToUnits(shares)
	if err != nil {
		return positionRow{}, err
	}
	if avgPriceMicros <= 0 {
		return positionRow{}, fmt.Errorf("avg price must be > 0")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return positionRow{}, err
	}
	defer tx.Rollback(ctx)

	userID = strings.TrimSpace(userID)
	if err := s.ensureWallet(ctx, tx, userID, seasonID); err != nil {
		return positionRow{}, err
	}

	var stockID int64
	var row positionRow
	err = tx.QueryRow(ctx, `
		SELECT id, symbol, display_name
		FROM game.stocks
		WHERE season_id = $1 AND symbol = $2
	`, seasonID, symbol).Scan(&stockID, &row.Symbol, &row.DisplayName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return positionRow{}, fmt.Errorf("stock %s not found", symbol)
		}
		return positionRow{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO game.positions (user_id, season_id, stock_id, quantity_units, avg_price_micros, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (user_id, season_id, stock_id)
		DO UPDATE SET quantity_units = EXCLUDED.quantity_units,
		              avg_price_micros = EXCLUDED.avg_price_micros,
		              updated_at = now()
	`, userID, seasonID, stockID, units, avgPriceMicros); err != nil {
		return positionRow{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return positionRow{}, err
	}
	row.QuantityUnits = units
	row.AvgPriceMicros = avgPriceMicros
	return row, nil
}

func (s *adminStore) deletePosition(ctx context.Context, userID, symbol string) error {
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return err
	}
	if err := game.ValidateSymbol(symbol); err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `
		DELETE FROM game.positions p
		USING game.stocks st
		WHERE p.stock_id = st.id
		  AND p.user_id = $1
		  AND p.season_id = $2
		  AND st.season_id = $2
		  AND st.symbol = $3
	`, strings.TrimSpace(userID), seasonID, symbol)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("position %s for %s not found", symbol, userID)
	}
	fmt.Printf("Position %s deleted for %s.\n", symbol, userID)
	return nil
}

func (s *adminStore) listStocks(ctx context.Context) ([]stockRow, error) {
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT symbol, display_name, current_price_micros, anchor_price_micros, listed_public
		FROM game.stocks
		WHERE season_id = $1
		ORDER BY symbol
	`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []stockRow
	for rows.Next() {
		var row stockRow
		if err := rows.Scan(&row.Symbol, &row.DisplayName, &row.CurrentPriceMicros, &row.AnchorPriceMicros, &row.ListedPublic); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *adminStore) setStockPrice(ctx context.Context, symbol string, priceMicros int64) (stockRow, error) {
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return stockRow{}, err
	}
	if err := game.ValidateSymbol(symbol); err != nil {
		return stockRow{}, err
	}
	if priceMicros <= 0 {
		return stockRow{}, fmt.Errorf("price must be > 0")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return stockRow{}, err
	}
	defer tx.Rollback(ctx)

	var stockID int64
	var row stockRow
	err = tx.QueryRow(ctx, `
		SELECT id, symbol, display_name, listed_public
		FROM game.stocks
		WHERE season_id = $1 AND symbol = $2
		FOR UPDATE
	`, seasonID, symbol).Scan(&stockID, &row.Symbol, &row.DisplayName, &row.ListedPublic)
	if err != nil {
		if err == pgx.ErrNoRows {
			return stockRow{}, fmt.Errorf("stock %s not found", symbol)
		}
		return stockRow{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.stocks
		SET current_price_micros = $2,
		    anchor_price_micros = $2,
		    updated_at = now()
		WHERE id = $1
	`, stockID, priceMicros); err != nil {
		return stockRow{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO game.stock_prices (stock_id, tick_at, price_micros)
		VALUES ($1, $2, $3)
	`, stockID, time.Now().UTC(), priceMicros); err != nil {
		return stockRow{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return stockRow{}, err
	}
	row.CurrentPriceMicros = priceMicros
	row.AnchorPriceMicros = priceMicros
	return row, nil
}
