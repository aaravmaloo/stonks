package game

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	mathrand "math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var usernameRE = regexp.MustCompile(`^[a-zA-Z0-9_]{3,24}$`)

var blockedNameFragments = []string{
	"admin",
	"mod",
	"support",
	"shit",
	"fuck",
	"bitch",
	"nazi",
}

type Service struct {
	db   *pgxpool.Pool
	log  *slog.Logger
	mu   sync.Mutex
	rand *mathrand.Rand
}

func NewService(db *pgxpool.Pool, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		db:   db,
		log:  logger,
		rand: mathrand.New(mathrand.NewSource(time.Now().UnixNano())),
	}
}

func (s *Service) ActiveSeasonID(ctx context.Context) (int64, error) {
	var seasonID int64
	err := s.db.QueryRow(ctx, `
		SELECT id
		FROM game.seasons
		WHERE status = 'active'
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&seasonID)
	if err == nil {
		return seasonID, nil
	}
	if err != pgx.ErrNoRows {
		return 0, err
	}

	err = s.db.QueryRow(ctx, `
		INSERT INTO game.seasons (name, status, starts_at, ends_at)
		VALUES ($1, 'active', now(), now() + interval '90 days')
		RETURNING id
	`, "Season 1").Scan(&seasonID)
	if err != nil {
		return 0, err
	}
	return seasonID, nil
}

func (s *Service) EnsurePlayer(ctx context.Context, userID, email, username string) error {
	seasonID, err := s.ActiveSeasonID(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(username) == "" {
		username = usernameFromEmail(email)
	}
	username = strings.TrimSpace(username)
	if !usernameRE.MatchString(username) {
		username = sanitizeUsername(usernameFromEmail(email))
	}
	inviteCode, err := generateInviteCode()
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO users.profiles (user_id, email, username, invite_code)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO NOTHING
	`, userID, email, username, inviteCode)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO game.wallets (user_id, season_id, balance_micros, peak_net_worth_micros)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT (user_id, season_id) DO NOTHING
	`, userID, seasonID, StarterBalanceMicros)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) SeedDefaults(ctx context.Context, seasonID int64) error {
	var count int
	if err := s.db.QueryRow(ctx, `SELECT COUNT(1) FROM game.stocks WHERE season_id = $1`, seasonID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	seed := []struct {
		Symbol string
		Name   string
		Price  int64
	}{
		{"COBOLT", "Cobalt Dynamics", 130 * MicrosPerStonky},
		{"NIMBUS", "Nimbus Labs", 95 * MicrosPerStonky},
		{"RUSTIC", "Rustic Systems", 115 * MicrosPerStonky},
		{"PYLONS", "Pylon Networks", 80 * MicrosPerStonky},
		{"JAVOLT", "Javolt Cloud", 105 * MicrosPerStonky},
		{"SWIFTR", "Swiftr Mobile", 150 * MicrosPerStonky},
		{"KOTLIN", "Kotlin Forge", 90 * MicrosPerStonky},
		{"NODEON", "Nodeon Runtime", 120 * MicrosPerStonky},
		{"RUBYIX", "Rubyix Core", 70 * MicrosPerStonky},
		{"ELIXIR", "Elixir Ops", 125 * MicrosPerStonky},
		{"QUARKX", "Quarkx Compute", 135 * MicrosPerStonky},
		{"VECTRA", "Vectra AI", 165 * MicrosPerStonky},
		{"DATUMX", "Datumx Data", 85 * MicrosPerStonky},
		{"CYBRON", "Cybron Secure", 140 * MicrosPerStonky},
		{"FUSION", "Fusion Grid", 110 * MicrosPerStonky},
		{"NEBULA", "Nebula Energy", 92 * MicrosPerStonky},
		{"ORBITZ", "Orbitz Space", 180 * MicrosPerStonky},
		{"ZENITH", "Zenith Retail", 75 * MicrosPerStonky},
		{"ARCANE", "Arcane Finance", 145 * MicrosPerStonky},
		{"LUMINA", "Lumina Health", 102 * MicrosPerStonky},
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, row := range seed {
		_, err := tx.Exec(ctx, `
			INSERT INTO game.stocks (season_id, symbol, display_name, listed_public, current_price_micros, anchor_price_micros, created_by_user_id)
			VALUES ($1, $2, $3, true, $4, $4, NULL)
		`, seasonID, row.Symbol, row.Name, row.Price)
		if err != nil {
			return err
		}
	}

	candidates := []struct {
		Name    string
		Role    string
		Trait   string
		Cost    int64
		Revenue int64
		RiskBps int32
	}{
		{"Maya Lee", "operator", "disciplined", 500 * MicrosPerStonky, 35 * MicrosPerStonky, 20},
		{"Arun Vale", "engineer", "innovative", 800 * MicrosPerStonky, 55 * MicrosPerStonky, 40},
		{"Iris Knox", "sales", "charismatic", 600 * MicrosPerStonky, 60 * MicrosPerStonky, 55},
		{"Noah Pike", "finance", "conservative", 700 * MicrosPerStonky, 45 * MicrosPerStonky, 15},
		{"Tara Sol", "product", "visionary", 900 * MicrosPerStonky, 75 * MicrosPerStonky, 70},
		{"Kian Moss", "ops", "resilient", 550 * MicrosPerStonky, 42 * MicrosPerStonky, 25},
	}
	for _, c := range candidates {
		_, err := tx.Exec(ctx, `
			INSERT INTO game.employee_candidates (season_id, full_name, role, trait, hire_cost_micros, revenue_per_tick_micros, risk_bps)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, seasonID, c.Name, c.Role, c.Trait, c.Cost, c.Revenue, c.RiskBps)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *Service) Dashboard(ctx context.Context, userID string, seasonID int64) (Dashboard, error) {
	var out Dashboard
	out.SeasonID = seasonID

	err := s.db.QueryRow(ctx, `
		SELECT balance_micros, peak_net_worth_micros
		FROM game.wallets
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID).Scan(&out.BalanceMicros, &out.PeakNetWorthMicros)
	if err != nil {
		return out, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT s.symbol, s.display_name, p.quantity_units, p.avg_price_micros, s.current_price_micros
		FROM game.positions p
		JOIN game.stocks s ON s.id = p.stock_id
		WHERE p.user_id = $1 AND p.season_id = $2
		ORDER BY s.symbol
	`, userID, seasonID)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	var holdings int64
	for rows.Next() {
		var pos PositionView
		if err := rows.Scan(&pos.Symbol, &pos.DisplayName, &pos.QuantityUnits, &pos.AvgPriceMicros, &pos.CurrentPriceMicros); err != nil {
			return out, err
		}
		marketValue, err := notionalMicros(pos.CurrentPriceMicros, pos.QuantityUnits)
		if err != nil {
			return out, err
		}
		costValue, err := notionalMicros(pos.AvgPriceMicros, pos.QuantityUnits)
		if err != nil {
			return out, err
		}
		pos.UnrealizedMicros = marketValue - costValue
		holdings += marketValue
		out.Positions = append(out.Positions, pos)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}

	bRows, err := s.db.Query(ctx, `
		SELECT b.id, b.name, b.visibility, b.is_listed,
		       COUNT(be.id) AS employee_count,
		       COALESCE(b.base_revenue_micros + SUM(be.revenue_per_tick_micros), b.base_revenue_micros) AS revenue
		FROM game.businesses b
		LEFT JOIN game.business_employees be ON be.business_id = b.id
		WHERE b.owner_user_id = $1 AND b.season_id = $2
		GROUP BY b.id
		ORDER BY b.id
	`, userID, seasonID)
	if err != nil {
		return out, err
	}
	defer bRows.Close()
	for bRows.Next() {
		var v BusinessView
		if err := bRows.Scan(&v.ID, &v.Name, &v.Visibility, &v.IsListed, &v.EmployeeCount, &v.RevenuePerTickMicros); err != nil {
			return out, err
		}
		out.Businesses = append(out.Businesses, v)
	}
	if err := bRows.Err(); err != nil {
		return out, err
	}

	out.NetWorthMicros = out.BalanceMicros + holdings
	return out, nil
}

func (s *Service) ListStocks(ctx context.Context, seasonID int64, includeUnlisted bool) ([]StockView, error) {
	query := `
		SELECT symbol, display_name, current_price_micros, listed_public
		FROM game.stocks
		WHERE season_id = $1
	`
	if !includeUnlisted {
		query += " AND listed_public = true"
	}
	query += " ORDER BY symbol"
	rows, err := s.db.Query(ctx, query, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StockView
	for rows.Next() {
		var s StockView
		if err := rows.Scan(&s.Symbol, &s.DisplayName, &s.CurrentPriceMicros, &s.ListedPublic); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (s *Service) StockDetail(ctx context.Context, seasonID int64, symbol string) (StockDetail, error) {
	var out StockDetail
	if err := s.db.QueryRow(ctx, `
		SELECT symbol, display_name, current_price_micros, listed_public
		FROM game.stocks
		WHERE season_id = $1 AND symbol = $2
	`, seasonID, strings.ToUpper(symbol)).Scan(&out.Symbol, &out.DisplayName, &out.CurrentPriceMicros, &out.ListedPublic); err != nil {
		return out, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT tick_at, price_micros
		FROM game.stock_prices sp
		JOIN game.stocks s ON s.id = sp.stock_id
		WHERE s.season_id = $1 AND s.symbol = $2
		ORDER BY tick_at DESC
		LIMIT 64
	`, seasonID, strings.ToUpper(symbol))
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var p PricePoint
		if err := rows.Scan(&p.TickAt, &p.PriceMicros); err != nil {
			return out, err
		}
		out.Series = append(out.Series, p)
	}
	return out, rows.Err()
}

func (s *Service) PlaceOrder(ctx context.Context, in OrderInput) (OrderResult, error) {
	var out OrderResult
	in.Symbol = strings.ToUpper(strings.TrimSpace(in.Symbol))
	in.Side = strings.ToLower(strings.TrimSpace(in.Side))
	if err := ValidateSymbol(in.Symbol); err != nil {
		return out, err
	}
	if in.QuantityUnits <= 0 {
		return out, fmt.Errorf("quantity must be > 0")
	}
	if in.Side != "buy" && in.Side != "sell" {
		return out, fmt.Errorf("side must be buy or sell")
	}

	const maxAttempts = 8
	retryDelay := 75 * time.Millisecond
	for attempt := 0; attempt < maxAttempts; attempt++ {
		tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			return out, err
		}
		err = func() error {
			defer tx.Rollback(ctx)

			if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "order"); err != nil {
				return err
			}

			var stockID int64
			var listed bool
			if err := tx.QueryRow(ctx, `
				SELECT id, current_price_micros, listed_public
				FROM game.stocks
				WHERE season_id = $1 AND symbol = $2
				FOR UPDATE
			`, in.SeasonID, in.Symbol).Scan(&stockID, &out.PriceMicros, &listed); err != nil {
				if err == pgx.ErrNoRows {
					return ErrStockNotFound
				}
				return err
			}
			if !listed {
				return fmt.Errorf("stock is not listed publicly")
			}
			notional, err := notionalMicros(out.PriceMicros, in.QuantityUnits)
			if err != nil {
				return err
			}
			fee := int64(math.Round(float64(notional) * 0.0015))
			out.NotionalMicros = notional
			out.FeeMicros = fee

			var balance, peak int64
			if err := tx.QueryRow(ctx, `
				SELECT balance_micros, peak_net_worth_micros
				FROM game.wallets
				WHERE user_id = $1 AND season_id = $2
				FOR UPDATE
			`, in.UserID, in.SeasonID).Scan(&balance, &peak); err != nil {
				return err
			}
			debtLimit := DebtLimitFromPeak(peak)

			switch in.Side {
			case "buy":
				nextBalance := balance - notional - fee
				if nextBalance < -debtLimit {
					maxUnits, maxNotional, maxFee := maxAffordableBuy(out.PriceMicros, balance, debtLimit)
					return fmt.Errorf("%w: max buy %.4f shares (notional %.2f + fee %.2f stonky)", ErrInsufficientFunds, UnitsToShares(maxUnits), MicrosToStonky(maxNotional), MicrosToStonky(maxFee))
				}
				if err := upsertBuyPosition(ctx, tx, in.UserID, in.SeasonID, stockID, in.QuantityUnits, out.PriceMicros); err != nil {
					return err
				}
				balance = nextBalance
			case "sell":
				if err := applySellPosition(ctx, tx, in.UserID, in.SeasonID, stockID, in.QuantityUnits); err != nil {
					return err
				}
				balance = balance + notional - fee
			}

			if _, err := tx.Exec(ctx, `
				UPDATE game.wallets
				SET balance_micros = $1, updated_at = now()
				WHERE user_id = $2 AND season_id = $3
			`, balance, in.UserID, in.SeasonID); err != nil {
				return err
			}

			if err := s.updatePeakNetWorthTx(ctx, tx, in.UserID, in.SeasonID); err != nil {
				return err
			}

			if err := appendLedgerEntries(ctx, tx, in.UserID, in.SeasonID, in.Side, notional, fee); err != nil {
				return err
			}

			err = tx.QueryRow(ctx, `
				INSERT INTO game.orders (user_id, season_id, stock_id, side, quantity_units, price_micros, fee_micros)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				RETURNING id
			`, in.UserID, in.SeasonID, stockID, in.Side, in.QuantityUnits, out.PriceMicros, fee).Scan(&out.OrderID)
			if err != nil {
				return err
			}

			out.BalanceMicros = balance
			return tx.Commit(ctx)
		}()
		if err == nil {
			return out, nil
		}
		if !isSerializationError(err) {
			return out, err
		}
		if attempt == maxAttempts-1 {
			return out, ErrTxConflict
		}
		if err := sleepWithContext(ctx, retryDelay); err != nil {
			return out, err
		}
		if retryDelay < 1200*time.Millisecond {
			retryDelay *= 2
		}
	}

	return out, ErrTxConflict
}

func (s *Service) CreateBusiness(ctx context.Context, in CreateBusinessInput) (int64, error) {
	var id int64
	in.Name = strings.TrimSpace(in.Name)
	in.Visibility = strings.ToLower(strings.TrimSpace(in.Visibility))
	if in.Name == "" {
		return 0, fmt.Errorf("business name is required")
	}
	if err := validateEntityName(in.Name); err != nil {
		return 0, err
	}
	if in.Visibility != "private" && in.Visibility != "public" {
		return 0, fmt.Errorf("visibility must be private or public")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "create_business"); err != nil {
		return 0, err
	}

	netWorth, err := netWorthTx(ctx, tx, in.UserID, in.SeasonID)
	if err != nil {
		return 0, err
	}
	if netWorth < BusinessUnlockMicros {
		return 0, ErrBusinessLocked
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO game.businesses (owner_user_id, season_id, name, visibility, is_listed, base_revenue_micros)
		VALUES ($1, $2, $3, $4, false, $5)
		RETURNING id
	`, in.UserID, in.SeasonID, in.Name, in.Visibility, 18*MicrosPerStonky).Scan(&id)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Service) BusinessState(ctx context.Context, userID string, seasonID, businessID int64) (BusinessView, error) {
	var out BusinessView
	err := s.db.QueryRow(ctx, `
		SELECT b.id, b.name, b.visibility, b.is_listed,
		       COUNT(be.id),
		       COALESCE(b.base_revenue_micros + SUM(be.revenue_per_tick_micros), b.base_revenue_micros)
		FROM game.businesses b
		LEFT JOIN game.business_employees be ON be.business_id = b.id
		WHERE b.id = $1 AND b.season_id = $2 AND b.owner_user_id = $3
		GROUP BY b.id
	`, businessID, seasonID, userID).Scan(&out.ID, &out.Name, &out.Visibility, &out.IsListed, &out.EmployeeCount, &out.RevenuePerTickMicros)
	return out, err
}

func (s *Service) SetBusinessVisibility(ctx context.Context, userID string, seasonID, businessID int64, visibility string) error {
	visibility = strings.ToLower(strings.TrimSpace(visibility))
	if visibility != "private" && visibility != "public" {
		return fmt.Errorf("visibility must be private or public")
	}
	cmd, err := s.db.Exec(ctx, `
		UPDATE game.businesses
		SET visibility = $1, updated_at = now()
		WHERE id = $2 AND season_id = $3 AND owner_user_id = $4
	`, visibility, businessID, seasonID, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrUnauthorized
	}
	return nil
}

func (s *Service) HireEmployee(ctx context.Context, in HireEmployeeInput) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "hire_employee"); err != nil {
		return err
	}

	var ownerID string
	if err := tx.QueryRow(ctx, `
		SELECT owner_user_id
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
		FOR UPDATE
	`, in.BusinessID, in.SeasonID).Scan(&ownerID); err != nil {
		return err
	}
	if ownerID != in.UserID {
		return ErrUnauthorized
	}

	var candidateName, role, trait string
	var cost, revenue int64
	var risk int32
	if err := tx.QueryRow(ctx, `
		SELECT full_name, role, trait, hire_cost_micros, revenue_per_tick_micros, risk_bps
		FROM game.employee_candidates
		WHERE id = $1 AND season_id = $2
	`, in.CandidateID, in.SeasonID).Scan(&candidateName, &role, &trait, &cost, &revenue, &risk); err != nil {
		return err
	}

	var balance, peak int64
	if err := tx.QueryRow(ctx, `
		SELECT balance_micros, peak_net_worth_micros
		FROM game.wallets
		WHERE user_id = $1 AND season_id = $2
		FOR UPDATE
	`, in.UserID, in.SeasonID).Scan(&balance, &peak); err != nil {
		return err
	}
	if balance-cost < -DebtLimitFromPeak(peak) {
		return ErrInsufficientFunds
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO game.business_employees
		    (business_id, season_id, source_candidate_id, full_name, role, trait, revenue_per_tick_micros, risk_bps)
		VALUES
		    ($1, $2, $3, $4, $5, $6, $7, $8)
	`, in.BusinessID, in.SeasonID, in.CandidateID, candidateName, role, trait, revenue, risk)
	if err != nil {
		return err
	}

	balance -= cost
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = $1, updated_at = now()
		WHERE user_id = $2 AND season_id = $3
	`, balance, in.UserID, in.SeasonID); err != nil {
		return err
	}
	if err := appendLedgerEntries(ctx, tx, in.UserID, in.SeasonID, "employee_hire", cost, 0); err != nil {
		return err
	}
	if err := s.updatePeakNetWorthTx(ctx, tx, in.UserID, in.SeasonID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) ListEmployeeCandidates(ctx context.Context, seasonID int64) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, full_name, role, trait, hire_cost_micros, revenue_per_tick_micros, risk_bps
		FROM game.employee_candidates
		WHERE season_id = $1
		ORDER BY id
	`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var name, role, trait string
		var cost, revenue int64
		var risk int32
		if err := rows.Scan(&id, &name, &role, &trait, &cost, &revenue, &risk); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id":                      id,
			"full_name":               name,
			"role":                    role,
			"trait":                   trait,
			"hire_cost_micros":        cost,
			"revenue_per_tick_micros": revenue,
			"risk_bps":                risk,
		})
	}
	return out, rows.Err()
}

func (s *Service) ListBusinessEmployees(ctx context.Context, userID string, seasonID, businessID int64) ([]map[string]any, error) {
	var ownerID string
	if err := s.db.QueryRow(ctx, `
		SELECT owner_user_id
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
	`, businessID, seasonID).Scan(&ownerID); err != nil {
		return nil, err
	}
	if ownerID != userID {
		return nil, ErrUnauthorized
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, full_name, role, trait, revenue_per_tick_micros, risk_bps, created_at
		FROM game.business_employees
		WHERE business_id = $1 AND season_id = $2
		ORDER BY id
	`, businessID, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var fullName, role, trait string
		var revenue int64
		var risk int32
		var createdAt time.Time
		if err := rows.Scan(&id, &fullName, &role, &trait, &revenue, &risk, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id":                      id,
			"full_name":               fullName,
			"role":                    role,
			"trait":                   trait,
			"revenue_per_tick_micros": revenue,
			"risk_bps":                risk,
			"created_at":              createdAt,
		})
	}
	return out, rows.Err()
}

func (s *Service) CreateCustomStock(ctx context.Context, in CreateStockInput) error {
	in.Symbol = strings.ToUpper(strings.TrimSpace(in.Symbol))
	if err := ValidateSymbol(in.Symbol); err != nil {
		return err
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if in.DisplayName == "" {
		return fmt.Errorf("display name required")
	}
	if err := validateEntityName(in.DisplayName); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "create_stock"); err != nil {
		return err
	}

	var ownerID string
	if err := tx.QueryRow(ctx, `
		SELECT owner_user_id
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
	`, in.BusinessID, in.SeasonID).Scan(&ownerID); err != nil {
		return err
	}
	if ownerID != in.UserID {
		return ErrUnauthorized
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO game.stocks
		    (season_id, symbol, display_name, listed_public, current_price_micros, anchor_price_micros, created_by_user_id, business_id)
		VALUES
		    ($1, $2, $3, false, $4, $4, $5, $6)
	`, in.SeasonID, in.Symbol, in.DisplayName, 100*MicrosPerStonky, in.UserID, in.BusinessID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) IPOStock(ctx context.Context, in IPOInput) error {
	in.Symbol = strings.ToUpper(strings.TrimSpace(in.Symbol))
	if err := ValidateSymbol(in.Symbol); err != nil {
		return err
	}
	if in.PriceMicros <= 0 {
		return fmt.Errorf("price must be > 0")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "ipo_stock"); err != nil {
		return err
	}

	var stockID int64
	var createdBy string
	var listed bool
	if err := tx.QueryRow(ctx, `
		SELECT id, COALESCE(created_by_user_id, ''), listed_public
		FROM game.stocks
		WHERE season_id = $1 AND symbol = $2
		FOR UPDATE
	`, in.SeasonID, in.Symbol).Scan(&stockID, &createdBy, &listed); err != nil {
		return err
	}
	if listed {
		return fmt.Errorf("stock already listed")
	}
	if createdBy != in.UserID {
		return ErrUnauthorized
	}

	if _, err := tx.Exec(ctx, `
		UPDATE game.stocks
		SET listed_public = true,
		    current_price_micros = $1,
		    anchor_price_micros = $1,
		    updated_at = now()
		WHERE id = $2
	`, in.PriceMicros, stockID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO game.stock_prices (stock_id, tick_at, price_micros)
		VALUES ($1, now(), $2)
	`, stockID, in.PriceMicros); err != nil {
		return err
	}
	_, _ = tx.Exec(ctx, `UPDATE game.businesses SET is_listed = true WHERE id = (SELECT business_id FROM game.stocks WHERE id = $1)`, stockID)

	return tx.Commit(ctx)
}

func (s *Service) BusinessIPO(ctx context.Context, userID string, seasonID, businessID int64, symbol string, priceMicros int64, idem string) error {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if err := ValidateSymbol(symbol); err != nil {
		return err
	}
	if priceMicros <= 0 {
		return fmt.Errorf("price must be > 0")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := claimIdempotency(ctx, tx, userID, idem, "business_ipo"); err != nil {
		return err
	}

	var name, visibility, ownerID string
	if err := tx.QueryRow(ctx, `
		SELECT name, visibility, owner_user_id
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
		FOR UPDATE
	`, businessID, seasonID).Scan(&name, &visibility, &ownerID); err != nil {
		return err
	}
	if ownerID != userID {
		return ErrUnauthorized
	}
	if visibility != "public" {
		return fmt.Errorf("business must be public before ipo")
	}
	display := businessDisplayName(name)

	_, err = tx.Exec(ctx, `
		INSERT INTO game.stocks
		    (season_id, symbol, display_name, listed_public, current_price_micros, anchor_price_micros, created_by_user_id, business_id)
		VALUES ($1, $2, $3, true, $4, $4, $5, $6)
		ON CONFLICT (season_id, symbol) DO NOTHING
	`, seasonID, symbol, display, priceMicros, userID, businessID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE game.businesses
		SET is_listed = true, stock_symbol = $1, updated_at = now()
		WHERE id = $2
	`, symbol, businessID)
	if err != nil {
		return err
	}
	var stockID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM game.stocks WHERE season_id = $1 AND symbol = $2
	`, seasonID, symbol).Scan(&stockID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO game.stock_prices (stock_id, tick_at, price_micros)
		VALUES ($1, now(), $2)
	`, stockID, priceMicros)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) AddFriend(ctx context.Context, userID, inviteCode string) error {
	inviteCode = strings.ToUpper(strings.TrimSpace(inviteCode))
	var followee string
	if err := s.db.QueryRow(ctx, `SELECT user_id FROM users.profiles WHERE invite_code = $1`, inviteCode).Scan(&followee); err != nil {
		return err
	}
	if followee == userID {
		return fmt.Errorf("cannot follow yourself")
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO game.friend_follows (follower_user_id, followee_user_id)
		VALUES ($1, $2)
		ON CONFLICT (follower_user_id, followee_user_id) DO NOTHING
	`, userID, followee)
	return err
}

func (s *Service) RemoveFriend(ctx context.Context, userID, inviteCode string) error {
	inviteCode = strings.ToUpper(strings.TrimSpace(inviteCode))
	var followee string
	if err := s.db.QueryRow(ctx, `SELECT user_id FROM users.profiles WHERE invite_code = $1`, inviteCode).Scan(&followee); err != nil {
		return err
	}
	_, err := s.db.Exec(ctx, `
		DELETE FROM game.friend_follows
		WHERE follower_user_id = $1 AND followee_user_id = $2
	`, userID, followee)
	return err
}

func (s *Service) GlobalLeaderboard(ctx context.Context, seasonID int64, limit int) ([]LeaderboardRow, error) {
	rows, err := s.db.Query(ctx, `
		WITH holdings AS (
			SELECT p.user_id,
			       COALESCE(SUM((p.quantity_units * st.current_price_micros) / $2), 0) AS holdings_micros
			FROM game.positions p
			JOIN game.stocks st ON st.id = p.stock_id
			WHERE p.season_id = $1
			GROUP BY p.user_id
		)
		SELECT pr.username, pr.invite_code,
		       (w.balance_micros + COALESCE(h.holdings_micros, 0)) AS net_worth_micros
		FROM game.wallets w
		JOIN users.profiles pr ON pr.user_id = w.user_id
		LEFT JOIN holdings h ON h.user_id = w.user_id
		WHERE w.season_id = $1
		ORDER BY net_worth_micros DESC
		LIMIT $3
	`, seasonID, ShareScale, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LeaderboardRow
	var rank int64 = 1
	for rows.Next() {
		var r LeaderboardRow
		if err := rows.Scan(&r.Username, &r.InviteCode, &r.NetWorthMicros); err != nil {
			return nil, err
		}
		r.Rank = rank
		rank++
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) FriendsLeaderboard(ctx context.Context, seasonID int64, userID string, limit int) ([]LeaderboardRow, error) {
	rows, err := s.db.Query(ctx, `
		WITH social AS (
			SELECT $3::text AS user_id
			UNION
			SELECT followee_user_id
			FROM game.friend_follows
			WHERE follower_user_id = $3
		),
		holdings AS (
			SELECT p.user_id,
			       COALESCE(SUM((p.quantity_units * st.current_price_micros) / $2), 0) AS holdings_micros
			FROM game.positions p
			JOIN game.stocks st ON st.id = p.stock_id
			WHERE p.season_id = $1
			GROUP BY p.user_id
		)
		SELECT pr.username, pr.invite_code,
		       (w.balance_micros + COALESCE(h.holdings_micros, 0)) AS net_worth_micros
		FROM social so
		JOIN game.wallets w ON w.user_id = so.user_id AND w.season_id = $1
		JOIN users.profiles pr ON pr.user_id = w.user_id
		LEFT JOIN holdings h ON h.user_id = w.user_id
		ORDER BY net_worth_micros DESC
		LIMIT $4
	`, seasonID, ShareScale, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LeaderboardRow
	var rank int64 = 1
	for rows.Next() {
		var r LeaderboardRow
		if err := rows.Scan(&r.Username, &r.InviteCode, &r.NetWorthMicros); err != nil {
			return nil, err
		}
		r.Rank = rank
		rank++
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) ReplaySync(ctx context.Context, userID string, seasonID int64, commands []map[string]any) ([]map[string]any, error) {
	results := make([]map[string]any, 0, len(commands))
	for _, cmd := range commands {
		method, _ := cmd["method"].(string)
		path, _ := cmd["path"].(string)
		idem, _ := cmd["idempotency_key"].(string)
		results = append(results, map[string]any{
			"method":          method,
			"path":            path,
			"idempotency_key": idem,
			"status":          "queued_for_cli_replay",
			"user_id":         userID,
			"season_id":       seasonID,
		})
	}
	return results, nil
}

func (s *Service) RunMarketTick(ctx context.Context, seasonID int64, tickEvery time.Duration, interestAPR float64, volatility string) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	params := volatilityParams(volatility)
	regime, err := currentRegimeTx(ctx, tx, seasonID)
	if err != nil {
		return err
	}
	if s.nextFloat() < params.RegimeSwitchProb {
		regime = randomRegime(s.nextFloat())
		if _, err := tx.Exec(ctx, `
			INSERT INTO game.market_state (season_id, regime, updated_at)
			VALUES ($1, $2, now())
			ON CONFLICT (season_id) DO UPDATE SET regime = $2, updated_at = now()
		`, seasonID, regime); err != nil {
			return err
		}
	}

	rows, err := tx.Query(ctx, `
		SELECT id, current_price_micros, anchor_price_micros
		FROM game.stocks
		WHERE season_id = $1
		FOR UPDATE
	`, seasonID)
	if err != nil {
		return err
	}
	type row struct {
		id     int64
		price  int64
		anchor int64
	}
	var stocks []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.price, &r.anchor); err != nil {
			rows.Close()
			return err
		}
		stocks = append(stocks, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	const minPriceMicros = int64(10_000)                // 0.01 stonky
	const maxPriceMicros = int64(2_000_000_000_000_000) // 2 trillion stonky
	for _, st := range stocks {
		anchorRet := (0.30 * regimeDrift(regime)) + params.AnchorNoiseScale*normalish(s.nextFloat())
		if s.nextFloat() < params.ShockProb*0.20 {
			anchorRet += signedShock(s.nextFloat(), s.nextFloat(), params.ShockScale*0.40)
		}
		nextAnchor := evolvePrice(st.anchor, anchorRet, params.MaxDropPerTick)
		if nextAnchor < minPriceMicros {
			nextAnchor = minPriceMicros
		}
		if nextAnchor > maxPriceMicros {
			nextAnchor = maxPriceMicros
		}

		ret := regimeDrift(regime) + params.NoiseScale*normalish(s.nextFloat()) + meanReversion(st.price, st.anchor, params.MeanReversion)
		if s.nextFloat() < params.ShockProb {
			ret += signedShock(s.nextFloat(), s.nextFloat(), params.ShockScale)
		}
		if s.nextFloat() < params.ExtremeShockProb {
			ret += signedShock(s.nextFloat(), s.nextFloat(), params.ExtremeShockScale)
		}

		next := evolvePrice(st.price, ret, params.MaxDropPerTick)
		if next < minPriceMicros {
			next = minPriceMicros
		}
		if next > maxPriceMicros {
			next = maxPriceMicros
		}
		if _, err := tx.Exec(ctx, `
			UPDATE game.stocks
			SET current_price_micros = $1,
			    anchor_price_micros = $2,
			    updated_at = now()
			WHERE id = $3
		`, next, nextAnchor, st.id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO game.stock_prices (stock_id, tick_at, price_micros)
			VALUES ($1, now(), $2)
		`, st.id, next); err != nil {
			return err
		}
	}

	if err := applyBusinessRevenueTx(ctx, tx, seasonID); err != nil {
		return err
	}
	if err := applyDebtInterestTx(ctx, tx, seasonID, tickEvery, interestAPR); err != nil {
		return err
	}
	if err := updateSeasonPeakNetWorthTx(ctx, tx, seasonID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func applyDebtInterestTx(ctx context.Context, tx pgx.Tx, seasonID int64, tickEvery time.Duration, apr float64) error {
	if apr <= 0 {
		return nil
	}
	ticksPerYear := (365 * 24 * time.Hour).Seconds() / tickEvery.Seconds()
	if ticksPerYear <= 0 {
		return nil
	}
	perTick := apr / ticksPerYear
	rows, err := tx.Query(ctx, `
		SELECT user_id, balance_micros
		FROM game.wallets
		WHERE season_id = $1 AND balance_micros < 0
		FOR UPDATE
	`, seasonID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type neg struct {
		userID  string
		balance int64
	}
	var items []neg
	for rows.Next() {
		var n neg
		if err := rows.Scan(&n.userID, &n.balance); err != nil {
			return err
		}
		items = append(items, n)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, n := range items {
		interest := int64(math.Ceil(math.Abs(float64(n.balance)) * perTick))
		if interest <= 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `
			UPDATE game.wallets
			SET balance_micros = balance_micros - $1,
			    updated_at = now()
			WHERE season_id = $2 AND user_id = $3
		`, interest, seasonID, n.userID); err != nil {
			return err
		}
		if err := appendLedgerEntries(ctx, tx, n.userID, seasonID, "debt_interest", interest, 0); err != nil {
			return err
		}
	}
	return nil
}

func applyBusinessRevenueTx(ctx context.Context, tx pgx.Tx, seasonID int64) error {
	rows, err := tx.Query(ctx, `
		SELECT b.owner_user_id,
		       COALESCE(SUM(b.base_revenue_micros + be_sum.employee_revenue), SUM(b.base_revenue_micros)) AS total_revenue
		FROM game.businesses b
		LEFT JOIN LATERAL (
			SELECT COALESCE(SUM(be.revenue_per_tick_micros), 0) AS employee_revenue
			FROM game.business_employees be
			WHERE be.business_id = b.id
		) be_sum ON TRUE
		WHERE b.season_id = $1
		GROUP BY b.owner_user_id
	`, seasonID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type gain struct {
		userID  string
		revenue int64
	}
	var gains []gain
	for rows.Next() {
		var g gain
		if err := rows.Scan(&g.userID, &g.revenue); err != nil {
			return err
		}
		if g.revenue > 0 {
			gains = append(gains, g)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, g := range gains {
		if _, err := tx.Exec(ctx, `
			UPDATE game.wallets
			SET balance_micros = balance_micros + $1,
			    updated_at = now()
			WHERE season_id = $2 AND user_id = $3
		`, g.revenue, seasonID, g.userID); err != nil {
			return err
		}
		if err := appendLedgerEntries(ctx, tx, g.userID, seasonID, "business_revenue", g.revenue, 0); err != nil {
			return err
		}
	}
	return nil
}

func updateSeasonPeakNetWorthTx(ctx context.Context, tx pgx.Tx, seasonID int64) error {
	_, err := tx.Exec(ctx, `
		UPDATE game.wallets w
		SET peak_net_worth_micros = GREATEST(
		        w.peak_net_worth_micros,
		        w.balance_micros + COALESCE((
		            SELECT SUM((p.quantity_units * s.current_price_micros) / $2)
		            FROM game.positions p
		            JOIN game.stocks s ON s.id = p.stock_id
		            WHERE p.user_id = w.user_id
		              AND p.season_id = w.season_id
		        ), 0)
		    ),
		    updated_at = now()
		WHERE w.season_id = $1
	`, seasonID, ShareScale)
	return err
}

func currentRegimeTx(ctx context.Context, tx pgx.Tx, seasonID int64) (string, error) {
	var regime string
	err := tx.QueryRow(ctx, `
		SELECT regime
		FROM game.market_state
		WHERE season_id = $1
	`, seasonID).Scan(&regime)
	if err == nil {
		return regime, nil
	}
	if err != pgx.ErrNoRows {
		return "", err
	}
	regime = "neutral"
	_, err = tx.Exec(ctx, `
		INSERT INTO game.market_state (season_id, regime, updated_at)
		VALUES ($1, $2, now())
	`, seasonID, regime)
	return regime, err
}

func randomRegime(seed float64) string {
	switch {
	case seed < 0.33:
		return "bear"
	case seed < 0.66:
		return "neutral"
	default:
		return "bull"
	}
}

func regimeDrift(regime string) float64 {
	switch regime {
	case "bull":
		return 0.0085
	case "bear":
		return -0.0085
	default:
		return 0.0000
	}
}

func meanReversion(price, anchor int64, strength float64) float64 {
	if anchor <= 0 {
		return 0
	}
	return strength * (float64(anchor-price) / float64(anchor))
}

func normalish(seed float64) float64 {
	return (seed + seed - 1)
}

func signedShock(magSeed, signSeed, base float64) float64 {
	mag := base * (0.35 + 2.8*magSeed*magSeed)
	if signSeed < 0.5 {
		return -mag
	}
	return mag
}

func evolvePrice(priceMicros int64, ret, maxDropPerTick float64) int64 {
	if priceMicros <= 0 {
		return 1
	}
	// Bound only the downside; upside can run.
	if ret < -maxDropPerTick {
		ret = -maxDropPerTick
	}
	next := int64(math.Round(float64(priceMicros) * math.Exp(ret)))
	if next < 1 {
		next = 1
	}
	return next
}

func isSerializationError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "40001"
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func maxAffordableBuy(priceMicros, balanceMicros, debtLimitMicros int64) (maxUnits, maxNotional, maxFee int64) {
	if priceMicros <= 0 {
		return 0, 0, 0
	}
	budget := balanceMicros + debtLimitMicros
	if budget <= 0 {
		return 0, 0, 0
	}
	hi := (budget * ShareScale) / priceMicros
	lo := int64(0)
	best := int64(0)
	for lo <= hi {
		mid := lo + (hi-lo)/2
		notional, err := notionalMicros(priceMicros, mid)
		if err != nil {
			hi = mid - 1
			continue
		}
		fee := int64(math.Round(float64(notional) * 0.0015))
		if notional+fee <= budget {
			best = mid
			lo = mid + 1
			maxNotional = notional
			maxFee = fee
			continue
		}
		hi = mid - 1
	}
	return best, maxNotional, maxFee
}

type marketDynamics struct {
	NoiseScale        float64
	ShockProb         float64
	ShockScale        float64
	ExtremeShockProb  float64
	ExtremeShockScale float64
	MeanReversion     float64
	AnchorNoiseScale  float64
	RegimeSwitchProb  float64
	MaxDropPerTick    float64
}

func volatilityParams(mode string) marketDynamics {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "calm":
		return marketDynamics{
			NoiseScale:        0.020,
			ShockProb:         0.05,
			ShockScale:        0.09,
			ExtremeShockProb:  0.008,
			ExtremeShockScale: 0.22,
			MeanReversion:     0.03,
			AnchorNoiseScale:  0.012,
			RegimeSwitchProb:  0.04,
			MaxDropPerTick:    1.20,
		}
	case "wild":
		return marketDynamics{
			NoiseScale:        0.060,
			ShockProb:         0.18,
			ShockScale:        0.20,
			ExtremeShockProb:  0.050,
			ExtremeShockScale: 0.60,
			MeanReversion:     0.010,
			AnchorNoiseScale:  0.038,
			RegimeSwitchProb:  0.11,
			MaxDropPerTick:    2.60,
		}
	default:
		return marketDynamics{
			NoiseScale:        0.038,
			ShockProb:         0.11,
			ShockScale:        0.14,
			ExtremeShockProb:  0.020,
			ExtremeShockScale: 0.35,
			MeanReversion:     0.018,
			AnchorNoiseScale:  0.022,
			RegimeSwitchProb:  0.07,
			MaxDropPerTick:    2.00,
		}
	}
}

func (s *Service) nextFloat() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rand.Float64()
}

func appendLedgerEntries(ctx context.Context, tx pgx.Tx, userID string, seasonID int64, action string, amountMicros, feeMicros int64) error {
	txID := uuid.NewString()
	debit := -amountMicros
	credit := amountMicros
	if action == "sell" || action == "business_revenue" {
		debit, credit = credit, debit
	}
	meta, _ := json.Marshal(map[string]any{"action": action})
	_, err := tx.Exec(ctx, `
		INSERT INTO game.ledger_entries (tx_group_id, user_id, season_id, account, delta_micros, metadata)
		VALUES
		($1, $2, $3, 'wallet', $4, $6::jsonb),
		($1, $2, $3, 'counterparty', $5, $6::jsonb)
	`, txID, userID, seasonID, debit, credit, string(meta))
	if err != nil {
		return err
	}
	if feeMicros > 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO game.ledger_entries (tx_group_id, user_id, season_id, account, delta_micros, metadata)
			VALUES ($1, $2, $3, 'fees', $4, $5::jsonb)
		`, txID, userID, seasonID, -feeMicros, `{"action":"fee"}`)
	}
	return err
}

func claimIdempotency(ctx context.Context, tx pgx.Tx, userID, key, action string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("idempotency key is required")
	}
	cmd, err := tx.Exec(ctx, `
		INSERT INTO game.idempotency_keys (user_id, key, action, created_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id, key) DO NOTHING
	`, userID, key, action)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrDuplicateIdempotency
	}
	return nil
}

func upsertBuyPosition(ctx context.Context, tx pgx.Tx, userID string, seasonID, stockID, qtyUnits, priceMicros int64) error {
	var oldQty, oldAvg int64
	err := tx.QueryRow(ctx, `
		SELECT quantity_units, avg_price_micros
		FROM game.positions
		WHERE user_id = $1 AND season_id = $2 AND stock_id = $3
		FOR UPDATE
	`, userID, seasonID, stockID).Scan(&oldQty, &oldAvg)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}

	if err == pgx.ErrNoRows {
		_, err = tx.Exec(ctx, `
			INSERT INTO game.positions (user_id, season_id, stock_id, quantity_units, avg_price_micros)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, seasonID, stockID, qtyUnits, priceMicros)
		return err
	}

	newQty := oldQty + qtyUnits
	if newQty <= 0 {
		return fmt.Errorf("invalid resulting quantity")
	}

	totalOld, err := notionalMicros(oldAvg, oldQty)
	if err != nil {
		return err
	}
	totalNew, err := notionalMicros(priceMicros, qtyUnits)
	if err != nil {
		return err
	}
	weightedCost := totalOld + totalNew
	newAvg, err := divideMicros(weightedCost, newQty)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE game.positions
		SET quantity_units = $1, avg_price_micros = $2, updated_at = now()
		WHERE user_id = $3 AND season_id = $4 AND stock_id = $5
	`, newQty, newAvg, userID, seasonID, stockID)
	return err
}

func applySellPosition(ctx context.Context, tx pgx.Tx, userID string, seasonID, stockID, qtyUnits int64) error {
	var oldQty int64
	if err := tx.QueryRow(ctx, `
		SELECT quantity_units
		FROM game.positions
		WHERE user_id = $1 AND season_id = $2 AND stock_id = $3
		FOR UPDATE
	`, userID, seasonID, stockID).Scan(&oldQty); err != nil {
		if err == pgx.ErrNoRows {
			return ErrInsufficientShares
		}
		return err
	}
	if oldQty < qtyUnits {
		return ErrInsufficientShares
	}
	next := oldQty - qtyUnits
	if next == 0 {
		_, err := tx.Exec(ctx, `
			DELETE FROM game.positions
			WHERE user_id = $1 AND season_id = $2 AND stock_id = $3
		`, userID, seasonID, stockID)
		return err
	}
	_, err := tx.Exec(ctx, `
		UPDATE game.positions
		SET quantity_units = $1, updated_at = now()
		WHERE user_id = $2 AND season_id = $3 AND stock_id = $4
	`, next, userID, seasonID, stockID)
	return err
}

func (s *Service) updatePeakNetWorthTx(ctx context.Context, tx pgx.Tx, userID string, seasonID int64) error {
	netWorth, err := netWorthTx(ctx, tx, userID, seasonID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE game.wallets
		SET peak_net_worth_micros = GREATEST(peak_net_worth_micros, $1),
		    updated_at = now()
		WHERE user_id = $2 AND season_id = $3
	`, netWorth, userID, seasonID)
	return err
}

func netWorthTx(ctx context.Context, tx pgx.Tx, userID string, seasonID int64) (int64, error) {
	var balance int64
	if err := tx.QueryRow(ctx, `
		SELECT balance_micros
		FROM game.wallets
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID).Scan(&balance); err != nil {
		return 0, err
	}
	var holdings int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM((p.quantity_units * s.current_price_micros) / $3), 0)
		FROM game.positions p
		JOIN game.stocks s ON s.id = p.stock_id
		WHERE p.user_id = $1 AND p.season_id = $2
	`, userID, seasonID, ShareScale).Scan(&holdings); err != nil {
		return 0, err
	}
	return balance + holdings, nil
}

func notionalMicros(priceMicros, qtyUnits int64) (int64, error) {
	p := big.NewInt(priceMicros)
	q := big.NewInt(qtyUnits)
	v := new(big.Int).Mul(p, q)
	v = v.Div(v, big.NewInt(ShareScale))
	if !v.IsInt64() {
		return 0, fmt.Errorf("notional overflow")
	}
	return v.Int64(), nil
}

func divideMicros(totalMicros, qtyUnits int64) (int64, error) {
	if qtyUnits <= 0 {
		return 0, fmt.Errorf("qty must be > 0")
	}
	v := new(big.Int).Mul(big.NewInt(totalMicros), big.NewInt(ShareScale))
	v = v.Div(v, big.NewInt(qtyUnits))
	if !v.IsInt64() {
		return 0, fmt.Errorf("avg overflow")
	}
	return v.Int64(), nil
}

func generateInviteCode() (string, error) {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = letters[int(buf[i])%len(letters)]
	}
	return string(buf), nil
}

func usernameFromEmail(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	parts := strings.Split(email, "@")
	if len(parts) == 0 || parts[0] == "" {
		return "player"
	}
	return sanitizeUsername(parts[0])
}

func sanitizeUsername(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "player"
	}
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			out = append(out, r)
		} else {
			out = append(out, '_')
		}
	}
	res := strings.Trim(string(out), "_")
	if len(res) < 3 {
		res = "player_" + res
	}
	if len(res) > 24 {
		res = res[:24]
	}
	return res
}

func businessDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Player Business"
	}
	if len(name) > 48 {
		return name[:48]
	}
	return name
}

func validateEntityName(name string) error {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return fmt.Errorf("name is required")
	}
	if len(clean) > 64 {
		return fmt.Errorf("name too long (max 64 chars)")
	}
	lower := strings.ToLower(clean)
	for _, fragment := range blockedNameFragments {
		if strings.Contains(lower, fragment) {
			return fmt.Errorf("name contains blocked content")
		}
	}
	return nil
}
