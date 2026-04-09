package game

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type machineSpec struct {
	Type         string
	DisplayName  string
	CostMicros   int64
	OutputMicros int64
	UpkeepMicros int64
	Reliability  int32
}

var machineCatalog = []machineSpec{
	{Type: "assembly_line", DisplayName: "Assembly Line", CostMicros: 6_500 * MicrosPerStonky, OutputMicros: 70 * MicrosPerStonky, UpkeepMicros: 12 * MicrosPerStonky, Reliability: 9450},
	{Type: "robotics_cell", DisplayName: "Robotics Cell", CostMicros: 12_500 * MicrosPerStonky, OutputMicros: 155 * MicrosPerStonky, UpkeepMicros: 28 * MicrosPerStonky, Reliability: 9300},
	{Type: "cloud_cluster", DisplayName: "Cloud Cluster", CostMicros: 18_000 * MicrosPerStonky, OutputMicros: 220 * MicrosPerStonky, UpkeepMicros: 42 * MicrosPerStonky, Reliability: 9250},
	{Type: "bio_reactor", DisplayName: "Bio Reactor", CostMicros: 25_000 * MicrosPerStonky, OutputMicros: 330 * MicrosPerStonky, UpkeepMicros: 66 * MicrosPerStonky, Reliability: 9100},
	{Type: "quantum_rig", DisplayName: "Quantum Rig", CostMicros: 40_000 * MicrosPerStonky, OutputMicros: 530 * MicrosPerStonky, UpkeepMicros: 105 * MicrosPerStonky, Reliability: 8900},
}

var fundUniverse = map[string][]string{
	"TECH6X": {"COBOLT", "NIMBUS", "SWIFTR", "KOTLIN", "NODEON", "QUARKX"},
	"CORE20": {"COBOLT", "NIMBUS", "RUSTIC", "PYLONS", "JAVOLT", "SWIFTR", "KOTLIN", "NODEON", "RUBYIX", "ELIXIR", "QUARKX", "VECTRA", "DATUMX", "CYBRON", "FUSION", "NEBULA", "ORBITZ", "ZENITH", "ARCANE", "LUMINA"},
	"VOLT10": {"SWIFTR", "QUARKX", "VECTRA", "CYBRON", "ORBITZ", "ARCANE", "COBOLT", "NODEON", "ELIXIR", "FUSION"},
	"DIVMAX": {"RUSTIC", "PYLONS", "RUBYIX", "DATUMX", "ZENITH", "LUMINA", "NIMBUS", "COBOLT"},
	"AIEDGE": {"VECTRA", "QUARKX", "ORBITZ", "CYBRON", "ARCANE", "SWIFTR"},
	"STABLE": {"NIMBUS", "RUSTIC", "PYLONS", "JAVOLT", "KOTLIN", "DATUMX", "LUMINA"},
}

const seededCandidatePoolSize = int(MaxBusinessEmployees)

type generatedCandidate struct {
	Name    string
	Role    string
	Trait   string
	Cost    int64
	Revenue int64
	RiskBps int32
}

func candidatePool(start, count int) []generatedCandidate {
	first := []string{"Maya", "Arun", "Iris", "Noah", "Tara", "Kian", "Lea", "Ravi", "Nora", "Evan", "Zara", "Omar", "Lina", "Kade", "Ava", "Dion", "Sana", "Milo", "Rhea", "Theo"}
	last := []string{"Lee", "Vale", "Knox", "Pike", "Sol", "Moss", "Rowe", "Jain", "Park", "Reid", "Cross", "Quill", "Stone", "Wren", "Bose", "Cho", "Kent", "Ford", "Hart", "Yoon"}
	roles := []string{"operator", "engineer", "sales", "finance", "product", "ops", "growth", "legal", "design", "analyst"}
	traits := []string{"disciplined", "innovative", "charismatic", "conservative", "visionary", "resilient", "strategic", "meticulous", "adaptive", "ambitious"}

	out := make([]generatedCandidate, 0, count)
	for i := 0; i < count; i++ {
		idx := start + i
		role := roles[idx%len(roles)]
		trait := traits[(idx*3)%len(traits)]
		revenue := int64(28+(idx%12)*7) * MicrosPerStonky
		cost := int64(420+(idx%15)*95) * MicrosPerStonky
		risk := int32(12 + (idx*9)%88)
		if role == "growth" || role == "sales" {
			risk += 20
			revenue += 10 * MicrosPerStonky
		}
		if role == "finance" || role == "legal" {
			risk -= 8
		}
		out = append(out, generatedCandidate{
			Name:    fmt.Sprintf("%s %s", first[idx%len(first)], last[(idx*7)%len(last)]),
			Role:    role,
			Trait:   trait,
			Cost:    cost,
			Revenue: revenue,
			RiskBps: risk,
		})
	}
	return out
}

func machineByType(machineType string) (machineSpec, error) {
	machineType = strings.ToLower(strings.TrimSpace(machineType))
	for _, spec := range machineCatalog {
		if spec.Type == machineType {
			return spec, nil
		}
	}
	return machineSpec{}, fmt.Errorf("unknown machine_type: %s", machineType)
}

func (s *Service) ListBusinessMachinery(ctx context.Context, userID string, seasonID, businessID int64) ([]map[string]any, error) {
	var owner string
	if err := s.db.QueryRow(ctx, `SELECT owner_user_id FROM game.businesses WHERE id = $1 AND season_id = $2`, businessID, seasonID).Scan(&owner); err != nil {
		return nil, err
	}
	if owner != userID {
		return nil, ErrUnauthorized
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, machine_type, level, output_bonus_micros, upkeep_micros, reliability_bps, updated_at
		FROM game.business_machinery
		WHERE business_id = $1 AND season_id = $2
		ORDER BY machine_type
	`, businessID, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var machineType string
		var level int32
		var output, upkeep int64
		var reliability int32
		var updatedAt any
		if err := rows.Scan(&id, &machineType, &level, &output, &upkeep, &reliability, &updatedAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id":                  id,
			"machine_type":        machineType,
			"level":               level,
			"output_bonus_micros": output,
			"upkeep_micros":       upkeep,
			"reliability_bps":     reliability,
			"updated_at":          updatedAt,
		})
	}
	return out, rows.Err()
}

func (s *Service) BuyBusinessMachinery(ctx context.Context, in BuyMachineryInput) (map[string]any, error) {
	out := map[string]any{}
	spec, err := machineByType(in.MachineType)
	if err != nil {
		return out, err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)

	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "buy_machinery"); err != nil {
		return out, err
	}
	var owner string
	if err := tx.QueryRow(ctx, `
		SELECT owner_user_id
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
		FOR UPDATE
	`, in.BusinessID, in.SeasonID).Scan(&owner); err != nil {
		return out, err
	}
	if owner != in.UserID {
		return out, ErrUnauthorized
	}

	var balance int64
	if err := tx.QueryRow(ctx, `
		SELECT balance_micros
		FROM game.wallets
		WHERE user_id = $1 AND season_id = $2
		FOR UPDATE
	`, in.UserID, in.SeasonID).Scan(&balance); err != nil {
		return out, err
	}

	var level int32
	err = tx.QueryRow(ctx, `
		SELECT level
		FROM game.business_machinery
		WHERE business_id = $1 AND season_id = $2 AND machine_type = $3
		FOR UPDATE
	`, in.BusinessID, in.SeasonID, spec.Type).Scan(&level)
	if err != nil && err != pgx.ErrNoRows {
		return out, err
	}
	nextLevel := int32(1)
	if err == nil {
		nextLevel = level + 1
	}
	cost := int64(float64(spec.CostMicros) * (1 + 0.25*float64(nextLevel-1)))
	if !hasPositiveBalanceAfterSpend(balance, cost) {
		return out, ErrInsufficientFunds
	}

	if err == pgx.ErrNoRows {
		_, err = tx.Exec(ctx, `
			INSERT INTO game.business_machinery
			    (business_id, season_id, machine_type, level, output_bonus_micros, upkeep_micros, reliability_bps)
			VALUES
			    ($1, $2, $3, 1, $4, $5, $6)
		`, in.BusinessID, in.SeasonID, spec.Type, spec.OutputMicros, spec.UpkeepMicros, spec.Reliability)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE game.business_machinery
			SET level = $1,
			    output_bonus_micros = ROUND(output_bonus_micros::numeric * 1.22),
			    upkeep_micros = ROUND(upkeep_micros::numeric * 1.18),
			    reliability_bps = GREATEST(7000, reliability_bps - 40),
			    updated_at = now()
			WHERE business_id = $2 AND season_id = $3 AND machine_type = $4
		`, nextLevel, in.BusinessID, in.SeasonID, spec.Type)
	}
	if err != nil {
		return out, err
	}
	balance -= cost
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = $1, updated_at = now()
		WHERE user_id = $2 AND season_id = $3
	`, balance, in.UserID, in.SeasonID); err != nil {
		return out, err
	}
	if err := appendLedgerEntries(ctx, tx, in.UserID, in.SeasonID, "machinery_buy", cost, 0); err != nil {
		return out, err
	}
	if err := s.updatePeakNetWorthTx(ctx, tx, in.UserID, in.SeasonID); err != nil {
		return out, err
	}
	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	out["ok"] = true
	out["machine_type"] = spec.Type
	out["cost_micros"] = cost
	out["new_balance_micros"] = balance
	out["new_level"] = nextLevel
	return out, nil
}

func (s *Service) TrainProfessional(ctx context.Context, in TrainProfessionalInput) (map[string]any, error) {
	out := map[string]any{}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)

	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "train_professional"); err != nil {
		return out, err
	}
	var owner string
	if err := tx.QueryRow(ctx, `
		SELECT owner_user_id
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
		FOR UPDATE
	`, in.BusinessID, in.SeasonID).Scan(&owner); err != nil {
		return out, err
	}
	if owner != in.UserID {
		return out, ErrUnauthorized
	}

	var curRevenue int64
	var curRisk int32
	if err := tx.QueryRow(ctx, `
		SELECT revenue_per_tick_micros, risk_bps
		FROM game.business_employees
		WHERE id = $1 AND business_id = $2 AND season_id = $3
		FOR UPDATE
	`, in.EmployeeID, in.BusinessID, in.SeasonID).Scan(&curRevenue, &curRisk); err != nil {
		return out, err
	}
	cost := int64(math.Round(float64(curRevenue) * 1.8))

	var balance int64
	if err := tx.QueryRow(ctx, `
		SELECT balance_micros
		FROM game.wallets
		WHERE user_id = $1 AND season_id = $2
		FOR UPDATE
	`, in.UserID, in.SeasonID).Scan(&balance); err != nil {
		return out, err
	}
	if !hasPositiveBalanceAfterSpend(balance, cost) {
		return out, ErrInsufficientFunds
	}

	nextRevenue := int64(math.Round(float64(curRevenue) * 1.15))
	nextRisk := int32(math.Min(10000, float64(curRisk+120)))
	if _, err := tx.Exec(ctx, `
		UPDATE game.business_employees
		SET revenue_per_tick_micros = $1, risk_bps = $2
		WHERE id = $3 AND business_id = $4 AND season_id = $5
	`, nextRevenue, nextRisk, in.EmployeeID, in.BusinessID, in.SeasonID); err != nil {
		return out, err
	}
	balance -= cost
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = $1, updated_at = now()
		WHERE user_id = $2 AND season_id = $3
	`, balance, in.UserID, in.SeasonID); err != nil {
		return out, err
	}
	if err := appendLedgerEntries(ctx, tx, in.UserID, in.SeasonID, "professional_training", cost, 0); err != nil {
		return out, err
	}
	if err := s.updatePeakNetWorthTx(ctx, tx, in.UserID, in.SeasonID); err != nil {
		return out, err
	}
	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	out["ok"] = true
	out["employee_id"] = in.EmployeeID
	out["training_cost_micros"] = cost
	out["balance_micros"] = balance
	out["revenue_per_tick_micros"] = nextRevenue
	out["risk_bps"] = nextRisk
	return out, nil
}

func (s *Service) TakeBusinessLoan(ctx context.Context, in BusinessLoanInput) (map[string]any, error) {
	out := map[string]any{}
	if in.AmountMicros <= 0 {
		return out, fmt.Errorf("amount must be > 0")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "take_business_loan"); err != nil {
		return out, err
	}

	var owner string
	if err := tx.QueryRow(ctx, `
		SELECT owner_user_id
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
		FOR UPDATE
	`, in.BusinessID, in.SeasonID).Scan(&owner); err != nil {
		return out, err
	}
	if owner != in.UserID {
		return out, ErrUnauthorized
	}

	netWorth, err := netWorthTx(ctx, tx, in.UserID, in.SeasonID)
	if err != nil {
		return out, err
	}
	maxLoan := int64(math.Round(float64(netWorth) * 0.45))
	var outstanding int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(outstanding_micros), 0)
		FROM game.business_loans
		WHERE owner_user_id = $1 AND season_id = $2 AND status = 'open'
	`, in.UserID, in.SeasonID).Scan(&outstanding); err != nil {
		return out, err
	}
	if outstanding+in.AmountMicros > maxLoan {
		return out, fmt.Errorf("loan request exceeds borrowing capacity")
	}

	interestBps := int32(65 + int32(math.Round(s.nextFloat()*95)))
	var balance int64
	if err := tx.QueryRow(ctx, `
		SELECT balance_micros
		FROM game.wallets
		WHERE user_id = $1 AND season_id = $2
		FOR UPDATE
	`, in.UserID, in.SeasonID).Scan(&balance); err != nil {
		return out, err
	}
	balance += in.AmountMicros
	if _, err := tx.Exec(ctx, `
		INSERT INTO game.business_loans
		    (business_id, season_id, owner_user_id, principal_micros, outstanding_micros, interest_bps, status)
		VALUES
		    ($1, $2, $3, $4, $4, $5, 'open')
	`, in.BusinessID, in.SeasonID, in.UserID, in.AmountMicros, interestBps); err != nil {
		return out, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = $1, updated_at = now()
		WHERE user_id = $2 AND season_id = $3
	`, balance, in.UserID, in.SeasonID); err != nil {
		return out, err
	}
	if err := appendLedgerEntries(ctx, tx, in.UserID, in.SeasonID, "business_loan_draw", in.AmountMicros, 0); err != nil {
		return out, err
	}
	if err := s.updatePeakNetWorthTx(ctx, tx, in.UserID, in.SeasonID); err != nil {
		return out, err
	}
	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	out["ok"] = true
	out["interest_bps"] = interestBps
	out["amount_micros"] = in.AmountMicros
	out["balance_micros"] = balance
	return out, nil
}

func (s *Service) RepayBusinessLoan(ctx context.Context, in BusinessLoanInput) (map[string]any, error) {
	out := map[string]any{}
	if in.AmountMicros <= 0 {
		return out, fmt.Errorf("amount must be > 0")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "repay_business_loan"); err != nil {
		return out, err
	}
	var owner string
	if err := tx.QueryRow(ctx, `
		SELECT owner_user_id
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
		FOR UPDATE
	`, in.BusinessID, in.SeasonID).Scan(&owner); err != nil {
		return out, err
	}
	if owner != in.UserID {
		return out, ErrUnauthorized
	}
	var balance int64
	if err := tx.QueryRow(ctx, `
		SELECT balance_micros
		FROM game.wallets
		WHERE user_id = $1 AND season_id = $2
		FOR UPDATE
	`, in.UserID, in.SeasonID).Scan(&balance); err != nil {
		return out, err
	}
	if !hasPositiveBalanceAfterSpend(balance, in.AmountMicros) {
		return out, ErrInsufficientFunds
	}

	rows, err := tx.Query(ctx, `
		SELECT id, outstanding_micros
		FROM game.business_loans
		WHERE business_id = $1 AND season_id = $2 AND status = 'open'
		ORDER BY id
		FOR UPDATE
	`, in.BusinessID, in.SeasonID)
	if err != nil {
		return out, err
	}
	type loan struct {
		id          int64
		outstanding int64
	}
	loans := make([]loan, 0)
	for rows.Next() {
		var l loan
		if err := rows.Scan(&l.id, &l.outstanding); err != nil {
			rows.Close()
			return out, err
		}
		loans = append(loans, l)
	}
	rows.Close()
	if len(loans) == 0 {
		return out, fmt.Errorf("no open business loans")
	}

	remaining := in.AmountMicros
	repaid := int64(0)
	for _, l := range loans {
		if remaining <= 0 {
			break
		}
		pay := l.outstanding
		if pay > remaining {
			pay = remaining
		}
		next := l.outstanding - pay
		status := "open"
		if next == 0 {
			status = "repaid"
		}
		if _, err := tx.Exec(ctx, `
			UPDATE game.business_loans
			SET outstanding_micros = $1, status = $2, missed_ticks = CASE WHEN $2 = 'repaid' THEN 0 ELSE missed_ticks END, updated_at = now()
			WHERE id = $3
		`, next, status, l.id); err != nil {
			return out, err
		}
		remaining -= pay
		repaid += pay
	}
	if repaid == 0 {
		return out, fmt.Errorf("nothing was repaid")
	}
	balance -= repaid
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = $1, updated_at = now()
		WHERE user_id = $2 AND season_id = $3
	`, balance, in.UserID, in.SeasonID); err != nil {
		return out, err
	}
	if err := appendLedgerEntries(ctx, tx, in.UserID, in.SeasonID, "business_loan_repay", repaid, 0); err != nil {
		return out, err
	}
	if err := s.updatePeakNetWorthTx(ctx, tx, in.UserID, in.SeasonID); err != nil {
		return out, err
	}
	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	out["ok"] = true
	out["repaid_micros"] = repaid
	out["balance_micros"] = balance
	return out, nil
}

func (s *Service) ListBusinessLoans(ctx context.Context, userID string, seasonID, businessID int64) ([]map[string]any, error) {
	var owner string
	if err := s.db.QueryRow(ctx, `
		SELECT owner_user_id
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
	`, businessID, seasonID).Scan(&owner); err != nil {
		return nil, err
	}
	if owner != userID {
		return nil, ErrUnauthorized
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, principal_micros, outstanding_micros, interest_bps, missed_ticks, status, created_at, updated_at
		FROM game.business_loans
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
		var principal, outstanding int64
		var interestBps int32
		var missed int32
		var status string
		var createdAt, updatedAt any
		if err := rows.Scan(&id, &principal, &outstanding, &interestBps, &missed, &status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id":                 id,
			"principal_micros":   principal,
			"outstanding_micros": outstanding,
			"interest_bps":       interestBps,
			"missed_ticks":       missed,
			"status":             status,
			"created_at":         createdAt,
			"updated_at":         updatedAt,
		})
	}
	return out, rows.Err()
}

func (s *Service) SetBusinessStrategy(ctx context.Context, in BusinessStrategyInput) error {
	strategy := strings.ToLower(strings.TrimSpace(in.Strategy))
	switch strategy {
	case "aggressive", "balanced", "defensive":
	default:
		return fmt.Errorf("strategy must be aggressive, balanced, or defensive")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "set_business_strategy"); err != nil {
		return err
	}
	cmd, err := tx.Exec(ctx, `
		UPDATE game.businesses
		SET strategy = $1, updated_at = now()
		WHERE id = $2 AND season_id = $3 AND owner_user_id = $4
	`, strategy, in.BusinessID, in.SeasonID, in.UserID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrUnauthorized
	}
	return tx.Commit(ctx)
}

func (s *Service) BuyBusinessUpgrade(ctx context.Context, in BusinessUpgradeInput) (map[string]any, error) {
	out := map[string]any{}
	upgrade := strings.ToLower(strings.TrimSpace(in.Upgrade))
	var col string
	switch upgrade {
	case "marketing":
		col = "marketing_level"
	case "rd":
		col = "rd_level"
	case "automation":
		col = "automation_level"
	case "compliance":
		col = "compliance_level"
	case "seats":
		col = "seat_capacity"
	default:
		return out, fmt.Errorf("upgrade must be one of: marketing, rd, automation, compliance, seats")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "buy_business_upgrade"); err != nil {
		return out, err
	}
	var owner string
	if upgrade == "seats" {
		var seatCapacity int64
		if err := tx.QueryRow(ctx, `
			SELECT owner_user_id, seat_capacity
			FROM game.businesses
			WHERE id = $1 AND season_id = $2
			FOR UPDATE
		`, in.BusinessID, in.SeasonID).Scan(&owner, &seatCapacity); err != nil {
			return out, err
		}
		if owner != in.UserID {
			return out, ErrUnauthorized
		}
		seatCapacity = effectiveEmployeeLimit(seatCapacity)
		if seatCapacity >= MaxBusinessEmployees {
			return out, fmt.Errorf("seat capacity already at max")
		}
		step := SeatUpgradeIncrement
		if seatCapacity+step > MaxBusinessEmployees {
			step = MaxBusinessEmployees - seatCapacity
		}
		level := int((seatCapacity - BaseBusinessEmployeeLimit) / SeatUpgradeIncrement)
		cost := int64(math.Round(float64((1_800+level*700)*int(MicrosPerStonky)) * (1 + float64(level)*0.18)))

		var balance int64
		if err := tx.QueryRow(ctx, `
			SELECT balance_micros
			FROM game.wallets
			WHERE user_id = $1 AND season_id = $2
			FOR UPDATE
		`, in.UserID, in.SeasonID).Scan(&balance); err != nil {
			return out, err
		}
		if !hasPositiveBalanceAfterSpend(balance, cost) {
			return out, ErrInsufficientFunds
		}
		if _, err := tx.Exec(ctx, `
			UPDATE game.businesses
			SET seat_capacity = LEAST($1, seat_capacity + $2), updated_at = now()
			WHERE id = $3 AND season_id = $4
		`, MaxBusinessEmployees, step, in.BusinessID, in.SeasonID); err != nil {
			return out, err
		}
		balance -= cost
		if _, err := tx.Exec(ctx, `
			UPDATE game.wallets
			SET balance_micros = $1, updated_at = now()
			WHERE user_id = $2 AND season_id = $3
		`, balance, in.UserID, in.SeasonID); err != nil {
			return out, err
		}
		if err := appendLedgerEntries(ctx, tx, in.UserID, in.SeasonID, "business_upgrade_"+upgrade, cost, 0); err != nil {
			return out, err
		}
		if err := s.updatePeakNetWorthTx(ctx, tx, in.UserID, in.SeasonID); err != nil {
			return out, err
		}
		if err := tx.Commit(ctx); err != nil {
			return out, err
		}
		out["ok"] = true
		out["upgrade"] = upgrade
		out["new_level"] = level + 1
		out["cost_micros"] = cost
		out["balance_micros"] = balance
		out["employee_limit"] = seatCapacity + step
		return out, nil
	}

	var level int32
	query := fmt.Sprintf(`
		SELECT owner_user_id, %s
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
		FOR UPDATE
	`, col)
	if err := tx.QueryRow(ctx, query, in.BusinessID, in.SeasonID).Scan(&owner, &level); err != nil {
		return out, err
	}
	if owner != in.UserID {
		return out, ErrUnauthorized
	}
	cost := int64(math.Round(float64((900+int(level)*350)*int(MicrosPerStonky)) * (1 + float64(level)*0.12)))

	var balance int64
	if err := tx.QueryRow(ctx, `
		SELECT balance_micros
		FROM game.wallets
		WHERE user_id = $1 AND season_id = $2
		FOR UPDATE
	`, in.UserID, in.SeasonID).Scan(&balance); err != nil {
		return out, err
	}
	if !hasPositiveBalanceAfterSpend(balance, cost) {
		return out, ErrInsufficientFunds
	}
	update := fmt.Sprintf(`
		UPDATE game.businesses
		SET %s = %s + 1, updated_at = now()
		WHERE id = $1 AND season_id = $2
	`, col, col)
	if _, err := tx.Exec(ctx, update, in.BusinessID, in.SeasonID); err != nil {
		return out, err
	}
	balance -= cost
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = $1, updated_at = now()
		WHERE user_id = $2 AND season_id = $3
	`, balance, in.UserID, in.SeasonID); err != nil {
		return out, err
	}
	if err := appendLedgerEntries(ctx, tx, in.UserID, in.SeasonID, "business_upgrade_"+upgrade, cost, 0); err != nil {
		return out, err
	}
	if err := s.updatePeakNetWorthTx(ctx, tx, in.UserID, in.SeasonID); err != nil {
		return out, err
	}
	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	out["ok"] = true
	out["upgrade"] = upgrade
	out["new_level"] = level + 1
	out["cost_micros"] = cost
	out["balance_micros"] = balance
	return out, nil
}

func (s *Service) BusinessReserveDeposit(ctx context.Context, in BusinessReserveInput) error {
	if in.AmountMicros <= 0 {
		return fmt.Errorf("amount must be > 0")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "business_reserve_deposit"); err != nil {
		return err
	}
	var owner string
	if err := tx.QueryRow(ctx, `
		SELECT owner_user_id
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
		FOR UPDATE
	`, in.BusinessID, in.SeasonID).Scan(&owner); err != nil {
		return err
	}
	if owner != in.UserID {
		return ErrUnauthorized
	}
	var balance int64
	if err := tx.QueryRow(ctx, `
		SELECT balance_micros
		FROM game.wallets
		WHERE user_id = $1 AND season_id = $2
		FOR UPDATE
	`, in.UserID, in.SeasonID).Scan(&balance); err != nil {
		return err
	}
	if !hasPositiveBalanceAfterSpend(balance, in.AmountMicros) {
		return ErrInsufficientFunds
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = balance_micros - $1, updated_at = now()
		WHERE user_id = $2 AND season_id = $3
	`, in.AmountMicros, in.UserID, in.SeasonID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.businesses
		SET cash_reserve_micros = cash_reserve_micros + $1, updated_at = now()
		WHERE id = $2 AND season_id = $3
	`, in.AmountMicros, in.BusinessID, in.SeasonID); err != nil {
		return err
	}
	if err := appendLedgerEntries(ctx, tx, in.UserID, in.SeasonID, "business_reserve_deposit", in.AmountMicros, 0); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) BusinessReserveWithdraw(ctx context.Context, in BusinessReserveInput) error {
	if in.AmountMicros <= 0 {
		return fmt.Errorf("amount must be > 0")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "business_reserve_withdraw"); err != nil {
		return err
	}
	var owner string
	var reserve int64
	if err := tx.QueryRow(ctx, `
		SELECT owner_user_id, cash_reserve_micros
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
		FOR UPDATE
	`, in.BusinessID, in.SeasonID).Scan(&owner, &reserve); err != nil {
		return err
	}
	if owner != in.UserID {
		return ErrUnauthorized
	}
	if reserve < in.AmountMicros {
		return ErrInsufficientFunds
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.businesses
		SET cash_reserve_micros = cash_reserve_micros - $1, updated_at = now()
		WHERE id = $2 AND season_id = $3
	`, in.AmountMicros, in.BusinessID, in.SeasonID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = balance_micros + $1, updated_at = now()
		WHERE user_id = $2 AND season_id = $3
	`, in.AmountMicros, in.UserID, in.SeasonID); err != nil {
		return err
	}
	if err := appendLedgerEntries(ctx, tx, in.UserID, in.SeasonID, "business_reserve_withdraw", in.AmountMicros, 0); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) SellBusinessToBank(ctx context.Context, userID string, seasonID, businessID int64, idem string) (map[string]any, error) {
	out := map[string]any{}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err := claimIdempotency(ctx, tx, userID, idem, "sell_business_to_bank"); err != nil {
		return out, err
	}

	var owner string
	var baseRevenue int64
	if err := tx.QueryRow(ctx, `
		SELECT owner_user_id, base_revenue_micros
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
		FOR UPDATE
	`, businessID, seasonID).Scan(&owner, &baseRevenue); err != nil {
		return out, err
	}
	if owner != userID {
		return out, ErrUnauthorized
	}

	var employeeRevenue int64
	var employeeCount int64
	var machineryOutput, machineryUpkeep int64
	var loanOutstanding int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(be.revenue_per_tick_micros), 0), b.employee_count
		FROM game.businesses b
		LEFT JOIN game.business_employees be
			ON be.business_id = b.id
		   AND be.season_id = b.season_id
		WHERE b.id = $1 AND b.season_id = $2
		GROUP BY b.employee_count
	`, businessID, seasonID).Scan(&employeeRevenue, &employeeCount); err != nil {
		return out, err
	}
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(output_bonus_micros), 0), COALESCE(SUM(upkeep_micros), 0)
		FROM game.business_machinery
		WHERE business_id = $1 AND season_id = $2
	`, businessID, seasonID).Scan(&machineryOutput, &machineryUpkeep); err != nil {
		return out, err
	}
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(outstanding_micros), 0)
		FROM game.business_loans
		WHERE business_id = $1 AND season_id = $2 AND status = 'open'
	`, businessID, seasonID).Scan(&loanOutstanding); err != nil {
		return out, err
	}

	operating := baseRevenue + employeeRevenue + machineryOutput - machineryUpkeep
	if operating < 0 {
		operating = 0
	}
	scale := float64(14 + employeeCount/3)
	factor := 0.82 + (s.nextFloat() * 0.40)
	gross := int64(math.Round(float64(operating) * scale * factor))
	payout := gross - loanOutstanding
	if payout < 0 {
		payout = 0
	}

	stakes, err := loadBusinessStakesTx(ctx, tx, businessID, seasonID)
	if err != nil {
		return out, err
	}
	ownerPayout := int64(0)
	ownerStakeBps := int32(0)
	for _, stake := range stakes {
		stakePayout := int64(math.Round(float64(payout) * float64(stake.StakeBps) / 10000.0))
		if stakePayout == 0 && payout > 0 && stake.StakeBps > 0 {
			stakePayout = 1
		}
		if _, err := tx.Exec(ctx, `
			UPDATE game.wallets
			SET balance_micros = balance_micros + $1, updated_at = now()
			WHERE user_id = $2 AND season_id = $3
		`, stakePayout, stake.UserID, seasonID); err != nil {
			return out, err
		}
		if err := appendLedgerEntries(ctx, tx, stake.UserID, seasonID, "business_sale", stakePayout, 0); err != nil {
			return out, err
		}
		if stake.UserID == userID {
			ownerPayout = stakePayout
			ownerStakeBps = stake.StakeBps
		}
	}
	var ownerBalance int64
	if err := tx.QueryRow(ctx, `
		SELECT balance_micros
		FROM game.wallets
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID).Scan(&ownerBalance); err != nil {
		return out, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE game.business_loans
		SET outstanding_micros = 0, status = 'sold_off', updated_at = now()
		WHERE business_id = $1 AND season_id = $2 AND status = 'open'
	`, businessID, seasonID); err != nil {
		return out, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO game.business_sale_history
		    (business_id, season_id, owner_user_id, gross_valuation_micros, adjustment_factor, loan_payoff_micros, payout_micros)
		VALUES
		    ($1, $2, $3, $4, $5, $6, $7)
	`, businessID, seasonID, userID, gross, factor, loanOutstanding, payout); err != nil {
		return out, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM game.businesses WHERE id = $1 AND season_id = $2`, businessID, seasonID); err != nil {
		return out, err
	}
	if err := s.updatePeakNetWorthTx(ctx, tx, userID, seasonID); err != nil {
		return out, err
	}
	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	out["ok"] = true
	out["gross_valuation_micros"] = gross
	out["adjustment_factor"] = factor
	out["loan_payoff_micros"] = loanOutstanding
	out["payout_micros"] = payout
	out["owner_payout_micros"] = ownerPayout
	out["owner_stake_bps"] = ownerStakeBps
	out["balance_micros"] = ownerBalance
	return out, nil
}

type businessStakeRow struct {
	UserID          string
	Username        string
	StakeBps        int32
	CostBasisMicros int64
}

func loadBusinessStakesTx(ctx context.Context, tx pgx.Tx, businessID, seasonID int64) ([]businessStakeRow, error) {
	rows, err := tx.Query(ctx, `
		SELECT s.user_id, p.username, s.stake_bps, s.cost_basis_micros
		FROM game.business_stakes s
		JOIN users.profiles p ON p.user_id = s.user_id
		WHERE s.business_id = $1 AND s.season_id = $2
		ORDER BY s.stake_bps DESC, p.username
	`, businessID, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []businessStakeRow
	for rows.Next() {
		var row businessStakeRow
		if err := rows.Scan(&row.UserID, &row.Username, &row.StakeBps, &row.CostBasisMicros); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func loadBusinessStakeBpsTx(ctx context.Context, tx pgx.Tx, businessID, seasonID int64, userID string) (int32, error) {
	var stakeBps int32
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(stake_bps, 0)
		FROM game.business_stakes
		WHERE business_id = $1 AND season_id = $2 AND user_id = $3
	`, businessID, seasonID, userID).Scan(&stakeBps)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	return stakeBps, err
}

func (s *Service) ListStakes(ctx context.Context, userID string, seasonID int64) ([]StakeView, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT business_id, stake_bps, cost_basis_micros
		FROM game.business_stakes
		WHERE user_id = $1 AND season_id = $2
		ORDER BY business_id
	`, userID, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type heldStake struct {
		businessID int64
		stakeBps   int32
		costBasis  int64
	}
	var held []heldStake
	for rows.Next() {
		var item heldStake
		if err := rows.Scan(&item.businessID, &item.stakeBps, &item.costBasis); err != nil {
			return nil, err
		}
		held = append(held, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]StakeView, 0, len(held))
	for _, item := range held {
		cycles, err := loadBusinessCyclesTx(ctx, tx, seasonID, "", &item.businessID)
		if err != nil {
			return nil, err
		}
		if len(cycles) == 0 {
			continue
		}
		cycle := cycles[0]
		proj := projectBusinessCycle(cycle)
		totalValue := estimateBusinessValuationMicros(cycle, proj)
		shareValue := int64(math.Round(float64(totalValue) * float64(item.stakeBps) / 10000.0))
		revenueShare := int64(math.Round(float64(proj.RevenuePerTickMicros) * float64(item.stakeBps) / 10000.0))
		out = append(out, StakeView{
			BusinessID:           cycle.businessID,
			BusinessName:         cycle.name,
			ControllerUsername:   cycle.controllerUsername,
			PrimaryRegion:        cycle.primaryRegion,
			NarrativeArc:         cycle.narrativeArc,
			StakeBps:             item.stakeBps,
			RevenueShareMicros:   revenueShare,
			EstimatedValueMicros: shareValue,
			CostBasisMicros:      item.costBasis,
			UnrealizedPLMicros:   shareValue - item.costBasis,
			LastEvent:            cycle.lastEvent,
		})
	}
	return out, tx.Commit(ctx)
}

func (s *Service) TransferBusinessStake(ctx context.Context, in TransferBusinessStakeInput) (map[string]any, error) {
	out := map[string]any{}
	if in.StakeBps <= 0 {
		return out, fmt.Errorf("stake percent must be > 0")
	}
	if in.StakeBps >= 10000 {
		return out, fmt.Errorf("cannot transfer 100%% or more")
	}
	in.RecipientUsername = strings.ToLower(strings.TrimSpace(in.RecipientUsername))
	if in.RecipientUsername == "" {
		return out, fmt.Errorf("recipient username is required")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "transfer_business_stake"); err != nil {
		return out, err
	}

	var ownerUserID string
	if err := tx.QueryRow(ctx, `
		SELECT owner_user_id
		FROM game.businesses
		WHERE id = $1 AND season_id = $2
		FOR UPDATE
	`, in.BusinessID, in.SeasonID).Scan(&ownerUserID); err != nil {
		return out, err
	}
	if ownerUserID != in.UserID {
		return out, ErrUnauthorized
	}

	var recipientUserID string
	if err := tx.QueryRow(ctx, `
		SELECT user_id
		FROM users.profiles
		WHERE LOWER(username) = $1
	`, in.RecipientUsername).Scan(&recipientUserID); err != nil {
		return out, fmt.Errorf("recipient username not found")
	}
	if recipientUserID == in.UserID {
		return out, fmt.Errorf("cannot transfer stake to yourself")
	}

	var ownerStakeBps int32
	var ownerBasis int64
	if err := tx.QueryRow(ctx, `
		SELECT stake_bps, cost_basis_micros
		FROM game.business_stakes
		WHERE business_id = $1 AND season_id = $2 AND user_id = $3
		FOR UPDATE
	`, in.BusinessID, in.SeasonID, in.UserID).Scan(&ownerStakeBps, &ownerBasis); err != nil {
		return out, err
	}
	if ownerStakeBps <= in.StakeBps {
		return out, fmt.Errorf("you must keep some controlling stake")
	}

	cycles, err := loadBusinessCyclesTx(ctx, tx, in.SeasonID, "", &in.BusinessID)
	if err != nil {
		return out, err
	}
	if len(cycles) == 0 {
		return out, fmt.Errorf("business not found")
	}
	valueNow := estimateBusinessValuationMicros(cycles[0], projectBusinessCycle(cycles[0]))
	transferredBasis := int64(math.Round(float64(valueNow) * float64(in.StakeBps) / 10000.0))
	ownerBasisReduction := int64(math.Round(float64(ownerBasis) * float64(in.StakeBps) / float64(ownerStakeBps)))

	if _, err := tx.Exec(ctx, `
		UPDATE game.business_stakes
		SET stake_bps = stake_bps - $1,
		    cost_basis_micros = GREATEST(0, cost_basis_micros - $2),
		    updated_at = now()
		WHERE business_id = $3 AND season_id = $4 AND user_id = $5
	`, in.StakeBps, ownerBasisReduction, in.BusinessID, in.SeasonID, in.UserID); err != nil {
		return out, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO game.business_stakes (business_id, season_id, user_id, stake_bps, cost_basis_micros)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (business_id, user_id)
		DO UPDATE SET stake_bps = game.business_stakes.stake_bps + EXCLUDED.stake_bps,
		              cost_basis_micros = game.business_stakes.cost_basis_micros + EXCLUDED.cost_basis_micros,
		              updated_at = now()
	`, in.BusinessID, in.SeasonID, recipientUserID, in.StakeBps, transferredBasis); err != nil {
		return out, err
	}

	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	out["ok"] = true
	out["business_id"] = in.BusinessID
	out["recipient_username"] = in.RecipientUsername
	out["stake_bps"] = in.StakeBps
	out["estimated_value_micros"] = transferredBasis
	return out, nil
}

func (s *Service) ListFunds(ctx context.Context, seasonID int64) ([]map[string]any, error) {
	navs, err := s.fundNAVs(ctx, seasonID)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(fundUniverse))
	for code := range fundUniverse {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	out := make([]map[string]any, 0, len(codes))
	for _, code := range codes {
		out = append(out, map[string]any{
			"code":       code,
			"components": fundUniverse[code],
			"nav_micros": navs[code],
		})
	}
	return out, nil
}

func (s *Service) TradeFund(ctx context.Context, in FundOrderInput) (map[string]any, error) {
	out := map[string]any{}
	in.FundCode = strings.ToUpper(strings.TrimSpace(in.FundCode))
	in.Side = strings.ToLower(strings.TrimSpace(in.Side))
	if in.Units <= 0 {
		return out, fmt.Errorf("units must be > 0")
	}
	if in.Side != "buy" && in.Side != "sell" {
		return out, fmt.Errorf("side must be buy or sell")
	}
	if _, ok := fundUniverse[in.FundCode]; !ok {
		return out, fmt.Errorf("unknown fund code: %s", in.FundCode)
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "fund_trade"); err != nil {
		return out, err
	}
	navs, err := s.fundNAVsTx(ctx, tx, in.SeasonID)
	if err != nil {
		return out, err
	}
	nav := navs[in.FundCode]
	notional, err := notionalMicros(nav, in.Units)
	if err != nil {
		return out, err
	}
	fee := int64(math.Round(float64(notional) * 0.0010))

	var balance int64
	if err := tx.QueryRow(ctx, `
		SELECT balance_micros
		FROM game.wallets
		WHERE user_id = $1 AND season_id = $2
		FOR UPDATE
	`, in.UserID, in.SeasonID).Scan(&balance); err != nil {
		return out, err
	}

	var posUnits, avgNav int64
	err = tx.QueryRow(ctx, `
		SELECT units, avg_nav_micros
		FROM game.fund_positions
		WHERE user_id = $1 AND season_id = $2 AND fund_code = $3
		FOR UPDATE
	`, in.UserID, in.SeasonID, in.FundCode).Scan(&posUnits, &avgNav)
	if err != nil && err != pgx.ErrNoRows {
		return out, err
	}
	if err == pgx.ErrNoRows {
		posUnits = 0
		avgNav = nav
	}

	switch in.Side {
	case "buy":
		next := balance - notional - fee
		if next <= 0 {
			return out, ErrInsufficientFunds
		}
		newUnits := posUnits + in.Units
		weightedOld, _ := notionalMicros(avgNav, posUnits)
		weightedNew, _ := notionalMicros(nav, in.Units)
		nextAvg, err := divideMicros(weightedOld+weightedNew, newUnits)
		if err != nil {
			return out, err
		}
		if posUnits == 0 {
			_, err = tx.Exec(ctx, `
				INSERT INTO game.fund_positions (user_id, season_id, fund_code, units, avg_nav_micros)
				VALUES ($1, $2, $3, $4, $5)
			`, in.UserID, in.SeasonID, in.FundCode, newUnits, nextAvg)
		} else {
			_, err = tx.Exec(ctx, `
				UPDATE game.fund_positions
				SET units = $1, avg_nav_micros = $2, updated_at = now()
				WHERE user_id = $3 AND season_id = $4 AND fund_code = $5
			`, newUnits, nextAvg, in.UserID, in.SeasonID, in.FundCode)
		}
		if err != nil {
			return out, err
		}
		balance = next
	case "sell":
		if posUnits < in.Units {
			return out, ErrInsufficientShares
		}
		newUnits := posUnits - in.Units
		if newUnits == 0 {
			if _, err := tx.Exec(ctx, `
				DELETE FROM game.fund_positions
				WHERE user_id = $1 AND season_id = $2 AND fund_code = $3
			`, in.UserID, in.SeasonID, in.FundCode); err != nil {
				return out, err
			}
		} else {
			if _, err := tx.Exec(ctx, `
				UPDATE game.fund_positions
				SET units = $1, updated_at = now()
				WHERE user_id = $2 AND season_id = $3 AND fund_code = $4
			`, newUnits, in.UserID, in.SeasonID, in.FundCode); err != nil {
				return out, err
			}
		}
		balance = balance + notional - fee
	}

	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = $1, updated_at = now()
		WHERE user_id = $2 AND season_id = $3
	`, balance, in.UserID, in.SeasonID); err != nil {
		return out, err
	}
	if err := appendLedgerEntries(ctx, tx, in.UserID, in.SeasonID, "fund_"+in.Side, notional, fee); err != nil {
		return out, err
	}
	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	out["ok"] = true
	out["side"] = in.Side
	out["fund_code"] = in.FundCode
	out["nav_micros"] = nav
	out["notional_micros"] = notional
	out["fee_micros"] = fee
	out["balance_micros"] = balance
	return out, nil
}

func (s *Service) estimateFundHoldingsMicros(ctx context.Context, userID string, seasonID int64) (int64, error) {
	rows, err := s.db.Query(ctx, `
		SELECT fund_code, units
		FROM game.fund_positions
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	navs, err := s.fundNAVs(ctx, seasonID)
	if err != nil {
		return 0, err
	}
	total := int64(0)
	for rows.Next() {
		var code string
		var units int64
		if err := rows.Scan(&code, &units); err != nil {
			return 0, err
		}
		nav := navs[strings.ToUpper(strings.TrimSpace(code))]
		value, err := notionalMicros(nav, units)
		if err != nil {
			return 0, err
		}
		total += value
	}
	return total, rows.Err()
}

func (s *Service) fundNAVs(ctx context.Context, seasonID int64) (map[string]int64, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	navs, err := s.fundNAVsTx(ctx, tx, seasonID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return navs, nil
}

func (s *Service) fundNAVsTx(ctx context.Context, tx pgx.Tx, seasonID int64) (map[string]int64, error) {
	rows, err := tx.Query(ctx, `
		SELECT symbol, current_price_micros
		FROM game.stocks
		WHERE season_id = $1
	`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	prices := map[string]int64{}
	for rows.Next() {
		var symbol string
		var price int64
		if err := rows.Scan(&symbol, &price); err != nil {
			return nil, err
		}
		prices[symbol] = price
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	navs := make(map[string]int64, len(fundUniverse))
	for code, symbols := range fundUniverse {
		if len(symbols) == 0 {
			navs[code] = 100 * MicrosPerStonky
			continue
		}
		total := int64(0)
		count := int64(0)
		for _, sym := range symbols {
			if p, ok := prices[sym]; ok && p > 0 {
				total += p
				count++
			}
		}
		if count == 0 {
			navs[code] = 100 * MicrosPerStonky
			continue
		}
		navs[code] = total / count
	}
	return navs, nil
}

func appendWalletDeltaEntry(ctx context.Context, tx pgx.Tx, userID string, seasonID, delta int64, action string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["action"] = action
	raw, _ := json.Marshal(metadata)
	_, err := tx.Exec(ctx, `
		INSERT INTO game.ledger_entries (tx_group_id, user_id, season_id, account, delta_micros, metadata)
		VALUES ($1, $2, $3, 'wallet', $4, $5::jsonb)
	`, uuid.NewString(), userID, seasonID, delta, string(raw))
	return err
}
