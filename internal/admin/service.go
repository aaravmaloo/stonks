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
	UserID              string `json:"user_id"`
	Email               string `json:"email"`
	Username            string `json:"username"`
	InviteCode          string `json:"invite_code"`
	BalanceMicros       int64  `json:"balance_micros"`
	PeakNetWorthMicros  int64  `json:"peak_net_worth_micros"`
	ActiveBusinessID    *int64 `json:"active_business_id,omitempty"`
	ReputationScore     int32  `json:"reputation_score"`
	CurrentProfitStreak int32  `json:"current_profit_streak"`
	BestProfitStreak    int32  `json:"best_profit_streak"`
	RiskAppetiteBps     int32  `json:"risk_appetite_bps"`
}

type Business struct {
	ID                   int64  `json:"id"`
	Name                 string `json:"name"`
	Visibility           string `json:"visibility"`
	IsListed             bool   `json:"is_listed"`
	BaseRevenueMicros    int64  `json:"base_revenue_micros"`
	PrimaryRegion        string `json:"primary_region"`
	NarrativeArc         string `json:"narrative_arc"`
	NarrativeFocus       string `json:"narrative_focus"`
	NarrativePressureBps int32  `json:"narrative_pressure_bps"`
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

type Stake struct {
	BusinessID      int64  `json:"business_id"`
	UserID          string `json:"user_id"`
	Username        string `json:"username"`
	StakeBps        int32  `json:"stake_bps"`
	CostBasisMicros int64  `json:"cost_basis_micros"`
}

type WorldState struct {
	Regime                 string `json:"regime"`
	PoliticalClimate       string `json:"political_climate"`
	PolicyFocus            string `json:"policy_focus"`
	CatalystName           string `json:"catalyst_name"`
	CatalystSummary        string `json:"catalyst_summary"`
	CatalystTicksRemaining int32  `json:"catalyst_ticks_remaining"`
	Headline               string `json:"headline"`
	AmericasBps            int32  `json:"americas_bps"`
	EuropeBps              int32  `json:"europe_bps"`
	AsiaBps                int32  `json:"asia_bps"`
	RiskRewardBiasBps      int32  `json:"risk_reward_bias_bps"`
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
		       w.active_business_id,
		       COALESCE(pp.reputation_score, 5000),
		       COALESCE(pp.current_profit_streak, 0),
		       COALESCE(pp.best_profit_streak, 0),
		       COALESCE(pp.risk_appetite_bps, 0)
		FROM users.profiles p
		LEFT JOIN game.wallets w
		  ON w.user_id = p.user_id AND w.season_id = $1
		LEFT JOIN game.player_progress pp
		  ON pp.user_id = p.user_id AND pp.season_id = $1
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
			&row.ReputationScore,
			&row.CurrentProfitStreak,
			&row.BestProfitStreak,
			&row.RiskAppetiteBps,
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
		       w.active_business_id,
		       COALESCE(pp.reputation_score, 5000),
		       COALESCE(pp.current_profit_streak, 0),
		       COALESCE(pp.best_profit_streak, 0),
		       COALESCE(pp.risk_appetite_bps, 0)
		FROM users.profiles p
		LEFT JOIN game.wallets w
		  ON w.user_id = p.user_id AND w.season_id = $2
		LEFT JOIN game.player_progress pp
		  ON pp.user_id = p.user_id AND pp.season_id = $2
		WHERE p.user_id = $1
	`, strings.TrimSpace(userID), seasonID).Scan(
		&row.UserID,
		&row.Email,
		&row.Username,
		&row.InviteCode,
		&row.BalanceMicros,
		&row.PeakNetWorthMicros,
		&row.ActiveBusinessID,
		&row.ReputationScore,
		&row.CurrentProfitStreak,
		&row.BestProfitStreak,
		&row.RiskAppetiteBps,
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

func (s *Service) ensurePlayerProgress(ctx context.Context, tx pgx.Tx, userID string, seasonID int64) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO game.player_progress (user_id, season_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, season_id) DO NOTHING
	`, userID, seasonID)
	return err
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
		SELECT id, name, visibility, is_listed, base_revenue_micros, primary_region, narrative_arc, narrative_focus, narrative_pressure_bps
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
		if err := rows.Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros, &row.PrimaryRegion, &row.NarrativeArc, &row.NarrativeFocus, &row.NarrativePressureBps); err != nil {
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
		RETURNING id, name, visibility, is_listed, base_revenue_micros, primary_region, narrative_arc, narrative_focus, narrative_pressure_bps
	`, businessID, strings.TrimSpace(name)).Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros, &row.PrimaryRegion, &row.NarrativeArc, &row.NarrativeFocus, &row.NarrativePressureBps)
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
		RETURNING id, name, visibility, is_listed, base_revenue_micros, primary_region, narrative_arc, narrative_focus, narrative_pressure_bps
	`, businessID, visibility).Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros, &row.PrimaryRegion, &row.NarrativeArc, &row.NarrativeFocus, &row.NarrativePressureBps)
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
		RETURNING id, name, visibility, is_listed, base_revenue_micros, primary_region, narrative_arc, narrative_focus, narrative_pressure_bps
	`, businessID, listed).Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros, &row.PrimaryRegion, &row.NarrativeArc, &row.NarrativeFocus, &row.NarrativePressureBps)
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
		RETURNING id, name, visibility, is_listed, base_revenue_micros, primary_region, narrative_arc, narrative_focus, narrative_pressure_bps
	`, businessID, amountMicros).Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros, &row.PrimaryRegion, &row.NarrativeArc, &row.NarrativeFocus, &row.NarrativePressureBps)
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

func (s *Service) SetPlayerProgress(ctx context.Context, userID string, reputationScore, currentStreak, bestStreak, riskAppetiteBps int32) (Player, error) {
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
	if err := s.ensurePlayerProgress(ctx, tx, userID, seasonID); err != nil {
		return Player{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.player_progress
		SET reputation_score = $3,
		    current_profit_streak = $4,
		    best_profit_streak = $5,
		    risk_appetite_bps = $6,
		    updated_at = now()
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID, reputationScore, currentStreak, bestStreak, riskAppetiteBps); err != nil {
		return Player{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Player{}, err
	}
	return s.PlayerByID(ctx, userID)
}

func (s *Service) ListBusinessStakes(ctx context.Context, businessID int64) ([]Stake, error) {
	rows, err := s.db.Query(ctx, `
		SELECT s.business_id, s.user_id, p.username, s.stake_bps, s.cost_basis_micros
		FROM game.business_stakes s
		JOIN users.profiles p ON p.user_id = s.user_id
		WHERE s.business_id = $1
		ORDER BY s.stake_bps DESC, p.username
	`, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Stake
	for rows.Next() {
		var row Stake
		if err := rows.Scan(&row.BusinessID, &row.UserID, &row.Username, &row.StakeBps, &row.CostBasisMicros); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Service) SetBusinessStake(ctx context.Context, businessID int64, username string, stakeBps int32) ([]Stake, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var ownerUserID string
	if err := tx.QueryRow(ctx, `
		SELECT owner_user_id
		FROM game.businesses
		WHERE id = $1
		FOR UPDATE
	`, businessID).Scan(&ownerUserID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("business %d not found", businessID)
		}
		return nil, err
	}

	username = strings.TrimSpace(strings.ToLower(username))
	var targetUserID string
	if err := tx.QueryRow(ctx, `SELECT user_id FROM users.profiles WHERE LOWER(username) = $1`, username).Scan(&targetUserID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("username %q not found", username)
		}
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT user_id, stake_bps
		FROM game.business_stakes
		WHERE business_id = $1
		FOR UPDATE
	`, businessID)
	if err != nil {
		return nil, err
	}
	current := map[string]int32{}
	for rows.Next() {
		var userID string
		var currentStake int32
		if err := rows.Scan(&userID, &currentStake); err != nil {
			rows.Close()
			return nil, err
		}
		current[userID] = currentStake
	}
	rows.Close()

	oldTarget := current[targetUserID]
	diff := stakeBps - oldTarget
	ownerStake := current[ownerUserID]
	if targetUserID != ownerUserID && ownerStake-diff < 0 {
		return nil, fmt.Errorf("owner stake would go negative")
	}
	if targetUserID == ownerUserID {
		var others int32
		for userID, currentStake := range current {
			if userID == ownerUserID {
				continue
			}
			others += currentStake
		}
		if stakeBps+others != 10000 {
			return nil, fmt.Errorf("owner stake must leave total stakes at exactly 100%%")
		}
	} else if ownerStake-diff+stakeBps+(10000-ownerStake-oldTarget) != 10000 {
		return nil, fmt.Errorf("stake math invalid")
	}

	if targetUserID != ownerUserID {
		if _, err := tx.Exec(ctx, `
			INSERT INTO game.business_stakes (business_id, season_id, user_id, stake_bps, cost_basis_micros)
			SELECT id, season_id, $2, 0, 0
			FROM game.businesses
			WHERE id = $1
			ON CONFLICT (business_id, user_id) DO NOTHING
		`, businessID, targetUserID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE game.business_stakes
			SET stake_bps = $2, updated_at = now()
			WHERE business_id = $1 AND user_id = $3
		`, businessID, stakeBps, targetUserID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE game.business_stakes
			SET stake_bps = stake_bps - $2, updated_at = now()
			WHERE business_id = $1 AND user_id = $3
		`, businessID, diff, ownerUserID); err != nil {
			return nil, err
		}
		if stakeBps == 0 {
			if _, err := tx.Exec(ctx, `DELETE FROM game.business_stakes WHERE business_id = $1 AND user_id = $2`, businessID, targetUserID); err != nil {
				return nil, err
			}
		}
	} else {
		if _, err := tx.Exec(ctx, `
			UPDATE game.business_stakes
			SET stake_bps = $2, updated_at = now()
			WHERE business_id = $1 AND user_id = $3
		`, businessID, stakeBps, ownerUserID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.ListBusinessStakes(ctx, businessID)
}

func (s *Service) SetBusinessNarrative(ctx context.Context, businessID int64, region, arc, focus string, pressureBps int32) (Business, error) {
	var row Business
	err := s.db.QueryRow(ctx, `
		UPDATE game.businesses
		SET primary_region = $2,
		    narrative_arc = $3,
		    narrative_focus = $4,
		    narrative_pressure_bps = $5,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, name, visibility, is_listed, base_revenue_micros, primary_region, narrative_arc, narrative_focus, narrative_pressure_bps
	`, businessID, strings.TrimSpace(region), strings.TrimSpace(arc), strings.TrimSpace(focus), pressureBps).Scan(&row.ID, &row.Name, &row.Visibility, &row.IsListed, &row.BaseRevenueMicros, &row.PrimaryRegion, &row.NarrativeArc, &row.NarrativeFocus, &row.NarrativePressureBps)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Business{}, fmt.Errorf("business %d not found", businessID)
		}
		return Business{}, err
	}
	return row, nil
}

func (s *Service) WorldState(ctx context.Context) (WorldState, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
	if err != nil {
		return WorldState{}, err
	}
	var row WorldState
	err = s.db.QueryRow(ctx, `
		SELECT regime, political_climate, policy_focus, catalyst_name, catalyst_summary, catalyst_ticks_remaining,
		       headline, americas_bps, europe_bps, asia_bps, risk_reward_bias_bps
		FROM game.market_state
		WHERE season_id = $1
	`, seasonID).Scan(&row.Regime, &row.PoliticalClimate, &row.PolicyFocus, &row.CatalystName, &row.CatalystSummary, &row.CatalystTicksRemaining, &row.Headline, &row.AmericasBps, &row.EuropeBps, &row.AsiaBps, &row.RiskRewardBiasBps)
	return row, err
}

func (s *Service) SetWorldState(ctx context.Context, in WorldState) (WorldState, error) {
	seasonID, err := s.ActiveSeasonID(ctx)
	if err != nil {
		return WorldState{}, err
	}
	err = s.db.QueryRow(ctx, `
		UPDATE game.market_state
		SET regime = $2,
		    political_climate = $3,
		    policy_focus = $4,
		    catalyst_name = $5,
		    catalyst_summary = $6,
		    catalyst_ticks_remaining = $7,
		    headline = $8,
		    americas_bps = $9,
		    europe_bps = $10,
		    asia_bps = $11,
		    risk_reward_bias_bps = $12,
		    updated_at = now()
		WHERE season_id = $1
		RETURNING regime, political_climate, policy_focus, catalyst_name, catalyst_summary, catalyst_ticks_remaining,
		          headline, americas_bps, europe_bps, asia_bps, risk_reward_bias_bps
	`, seasonID, in.Regime, in.PoliticalClimate, in.PolicyFocus, in.CatalystName, in.CatalystSummary, in.CatalystTicksRemaining, in.Headline, in.AmericasBps, in.EuropeBps, in.AsiaBps, in.RiskRewardBiasBps).
		Scan(&in.Regime, &in.PoliticalClimate, &in.PolicyFocus, &in.CatalystName, &in.CatalystSummary, &in.CatalystTicksRemaining, &in.Headline, &in.AmericasBps, &in.EuropeBps, &in.AsiaBps, &in.RiskRewardBiasBps)
	return in, err
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
