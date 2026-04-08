package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"stanks/internal/game"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db *pgxpool.Pool
}

type Player struct {
	UserID             string `json:"user_id"`
	Email              string `json:"email"`
	Username           string `json:"username"`
	InviteCode         string `json:"invite_code"`
	BalanceMicros      int64  `json:"balance_micros"`
	PeakNetWorthMicros int64  `json:"peak_net_worth_micros"`
	ActiveBusinessID   *int64 `json:"active_business_id,omitempty"`
}

type Business struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Visibility        string `json:"visibility"`
	IsListed          bool   `json:"is_listed"`
	BaseRevenueMicros int64  `json:"base_revenue_micros"`
}

type Position struct {
	Symbol         string `json:"symbol"`
	DisplayName    string `json:"display_name"`
	QuantityUnits  int64  `json:"quantity_units"`
	AvgPriceMicros int64  `json:"avg_price_micros"`
}

type Stock struct {
	Symbol             string `json:"symbol"`
	DisplayName        string `json:"display_name"`
	CurrentPriceMicros int64  `json:"current_price_micros"`
	AnchorPriceMicros  int64  `json:"anchor_price_micros"`
	ListedPublic       bool   `json:"listed_public"`
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) ActiveSeasonID(ctx context.Context) (int64, error) {
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

func (s *Service) ListPlayers(ctx context.Context, query string) ([]Player, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
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

	var out []Player
	for rows.Next() {
		var row Player
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

func (s *Service) PlayerByID(ctx context.Context, userID string) (Player, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
	if err != nil {
		return Player{}, err
	}
	var row Player
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
			return Player{}, fmt.Errorf("player %q not found", userID)
		}
		return Player{}, err
	}
	return row, nil
}

func (s *Service) ensureWallet(ctx context.Context, tx pgx.Tx, userID string, seasonID int64) error {
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

func (s *Service) ChangeBalance(ctx context.Context, userID string, deltaMicros int64) (Player, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
	if err != nil {
		return Player{}, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Player{}, err
	}
	defer tx.Rollback(ctx)

	userID = strings.TrimSpace(userID)
	if err := s.ensureWallet(ctx, tx, userID, seasonID); err != nil {
		return Player{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = balance_micros + $3,
		    updated_at = now()
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID, deltaMicros); err != nil {
		return Player{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Player{}, err
	}
	return s.PlayerByID(ctx, userID)
}

func (s *Service) SetBalance(ctx context.Context, userID string, amountMicros int64) (Player, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
	if err != nil {
		return Player{}, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Player{}, err
	}
	defer tx.Rollback(ctx)

	userID = strings.TrimSpace(userID)
	if err := s.ensureWallet(ctx, tx, userID, seasonID); err != nil {
		return Player{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = $3,
		    updated_at = now()
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID, amountMicros); err != nil {
		return Player{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Player{}, err
	}
	return s.PlayerByID(ctx, userID)
}

func (s *Service) ChangePeak(ctx context.Context, userID string, deltaMicros int64) (Player, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
	if err != nil {
		return Player{}, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Player{}, err
	}
	defer tx.Rollback(ctx)

	userID = strings.TrimSpace(userID)
	if err := s.ensureWallet(ctx, tx, userID, seasonID); err != nil {
		return Player{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET peak_net_worth_micros = peak_net_worth_micros + $3,
		    updated_at = now()
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID, deltaMicros); err != nil {
		return Player{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Player{}, err
	}
	return s.PlayerByID(ctx, userID)
}

func (s *Service) SetPeak(ctx context.Context, userID string, amountMicros int64) (Player, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
	if err != nil {
		return Player{}, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Player{}, err
	}
	defer tx.Rollback(ctx)

	userID = strings.TrimSpace(userID)
	if err := s.ensureWallet(ctx, tx, userID, seasonID); err != nil {
		return Player{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET peak_net_worth_micros = $3,
		    updated_at = now()
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID, amountMicros); err != nil {
		return Player{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Player{}, err
	}
	return s.PlayerByID(ctx, userID)
}

func (s *Service) SetActiveBusiness(ctx context.Context, userID string, businessID int64) (Player, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
	if err != nil {
		return Player{}, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Player{}, err
	}
	defer tx.Rollback(ctx)

	userID = strings.TrimSpace(userID)
	if err := s.ensureWallet(ctx, tx, userID, seasonID); err != nil {
		return Player{}, err
	}
	if businessID == 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE game.wallets
			SET active_business_id = NULL, updated_at = now()
			WHERE user_id = $1 AND season_id = $2
		`, userID, seasonID); err != nil {
			return Player{}, err
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
			return Player{}, err
		}
		if !exists {
			return Player{}, fmt.Errorf("business %d not found for player %s", businessID, userID)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE game.wallets
			SET active_business_id = $3, updated_at = now()
			WHERE user_id = $1 AND season_id = $2
		`, userID, seasonID, businessID); err != nil {
			return Player{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Player{}, err
	}
	return s.PlayerByID(ctx, userID)
}

func (s *Service) ListBusinessesByUser(ctx context.Context, userID string) ([]Business, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
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

	var out []Business
	for rows.Next() {
		var row Business
		if err := rows.Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Service) SetBusinessName(ctx context.Context, businessID int64, name string) (Business, error) {
	var row Business
	err := s.db.QueryRow(ctx, `
		UPDATE game.businesses
		SET name = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, name, visibility, is_listed, base_revenue_micros
	`, businessID, strings.TrimSpace(name)).Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Business{}, fmt.Errorf("business %d not found", businessID)
		}
		return Business{}, err
	}
	return row, nil
}

func (s *Service) SetBusinessVisibility(ctx context.Context, businessID int64, visibility string) (Business, error) {
	if visibility != "private" && visibility != "public" {
		return Business{}, fmt.Errorf("visibility must be private or public")
	}
	var row Business
	err := s.db.QueryRow(ctx, `
		UPDATE game.businesses
		SET visibility = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, name, visibility, is_listed, base_revenue_micros
	`, businessID, visibility).Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Business{}, fmt.Errorf("business %d not found", businessID)
		}
		return Business{}, err
	}
	return row, nil
}

func (s *Service) SetBusinessListed(ctx context.Context, businessID int64, listed bool) (Business, error) {
	var row Business
	err := s.db.QueryRow(ctx, `
		UPDATE game.businesses
		SET is_listed = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, name, visibility, is_listed, base_revenue_micros
	`, businessID, listed).Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Business{}, fmt.Errorf("business %d not found", businessID)
		}
		return Business{}, err
	}
	return row, nil
}

func (s *Service) SetBusinessRevenue(ctx context.Context, businessID int64, amountMicros int64) (Business, error) {
	var row Business
	err := s.db.QueryRow(ctx, `
		UPDATE game.businesses
		SET base_revenue_micros = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, name, visibility, is_listed, base_revenue_micros
	`, businessID, amountMicros).Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Business{}, fmt.Errorf("business %d not found", businessID)
		}
		return Business{}, err
	}
	return row, nil
}

func (s *Service) DeleteBusiness(ctx context.Context, businessID int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM game.businesses WHERE id = $1`, businessID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("business %d not found", businessID)
	}
	return nil
}

func (s *Service) ListPositionsByUser(ctx context.Context, userID string) ([]Position, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
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

	var out []Position
	for rows.Next() {
		var row Position
		if err := rows.Scan(&row.Symbol, &row.DisplayName, &row.QuantityUnits, &row.AvgPriceMicros); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Service) SetPosition(ctx context.Context, userID, symbol string, quantityUnits, avgPriceMicros int64) (Position, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
	if err != nil {
		return Position{}, err
	}
	if err := game.ValidateSymbol(symbol); err != nil {
		return Position{}, err
	}
	if quantityUnits <= 0 {
		return Position{}, fmt.Errorf("quantity units must be > 0")
	}
	if avgPriceMicros <= 0 {
		return Position{}, fmt.Errorf("avg price must be > 0")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Position{}, err
	}
	defer tx.Rollback(ctx)

	userID = strings.TrimSpace(userID)
	if err := s.ensureWallet(ctx, tx, userID, seasonID); err != nil {
		return Position{}, err
	}

	var stockID int64
	var row Position
	err = tx.QueryRow(ctx, `
		SELECT id, symbol, display_name
		FROM game.stocks
		WHERE season_id = $1 AND symbol = $2
	`, seasonID, symbol).Scan(&stockID, &row.Symbol, &row.DisplayName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Position{}, fmt.Errorf("stock %s not found", symbol)
		}
		return Position{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO game.positions (user_id, season_id, stock_id, quantity_units, avg_price_micros, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (user_id, season_id, stock_id)
		DO UPDATE SET quantity_units = EXCLUDED.quantity_units,
		              avg_price_micros = EXCLUDED.avg_price_micros,
		              updated_at = now()
	`, userID, seasonID, stockID, quantityUnits, avgPriceMicros); err != nil {
		return Position{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Position{}, err
	}
	row.QuantityUnits = quantityUnits
	row.AvgPriceMicros = avgPriceMicros
	return row, nil
}

func (s *Service) DeletePosition(ctx context.Context, userID, symbol string) error {
	seasonID, err := s.ActiveSeasonID(ctx)
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
	return nil
}

func (s *Service) ListStocks(ctx context.Context) ([]Stock, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
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

	var out []Stock
	for rows.Next() {
		var row Stock
		if err := rows.Scan(&row.Symbol, &row.DisplayName, &row.CurrentPriceMicros, &row.AnchorPriceMicros, &row.ListedPublic); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Service) SetStockPrice(ctx context.Context, symbol string, priceMicros int64) (Stock, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
	if err != nil {
		return Stock{}, err
	}
	if err := game.ValidateSymbol(symbol); err != nil {
		return Stock{}, err
	}
	if priceMicros <= 0 {
		return Stock{}, fmt.Errorf("price must be > 0")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Stock{}, err
	}
	defer tx.Rollback(ctx)

	var stockID int64
	var row Stock
	err = tx.QueryRow(ctx, `
		SELECT id, symbol, display_name, listed_public
		FROM game.stocks
		WHERE season_id = $1 AND symbol = $2
		FOR UPDATE
	`, seasonID, symbol).Scan(&stockID, &row.Symbol, &row.DisplayName, &row.ListedPublic)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Stock{}, fmt.Errorf("stock %s not found", symbol)
		}
		return Stock{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.stocks
		SET current_price_micros = $2,
		    anchor_price_micros = $2,
		    updated_at = now()
		WHERE id = $1
	`, stockID, priceMicros); err != nil {
		return Stock{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO game.stock_prices (stock_id, tick_at, price_micros)
		VALUES ($1, $2, $3)
	`, stockID, time.Now().UTC(), priceMicros); err != nil {
		return Stock{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Stock{}, err
	}
	row.CurrentPriceMicros = priceMicros
	row.AnchorPriceMicros = priceMicros
	return row, nil
}
