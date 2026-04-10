package game

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"
)

type rushModeSpec struct {
	Name             string
	BaseWinChance    float64
	BasePayoutBps    int32
	BasePoints       int32
	StreakBonusBps   int32
	WorldBonusFactor float64
}

var rushModes = map[string]rushModeSpec{
	"steady": {Name: "steady", BaseWinChance: 0.62, BasePayoutBps: 15400, BasePoints: 14, StreakBonusBps: 160, WorldBonusFactor: 0.10},
	"surge":  {Name: "surge", BaseWinChance: 0.46, BasePayoutBps: 20800, BasePoints: 26, StreakBonusBps: 280, WorldBonusFactor: 0.18},
	"apex":   {Name: "apex", BaseWinChance: 0.29, BasePayoutBps: 32200, BasePoints: 44, StreakBonusBps: 420, WorldBonusFactor: 0.26},
}

func ensureRushProgressTx(ctx context.Context, tx pgx.Tx, userID string, seasonID int64) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO game.rush_progress (user_id, season_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, season_id) DO NOTHING
	`, userID, seasonID)
	return err
}

func nextRushVaultTarget(level int32) int32 {
	base := int32(90)
	if level < 0 {
		level = 0
	}
	return base + level*45
}

func rushMilestoneReward(streak int32, wagerMicros int64) int64 {
	switch streak {
	case 3:
		return max64(120*MicrosPerStonky, wagerMicros/4)
	case 5:
		return max64(260*MicrosPerStonky, wagerMicros/2)
	case 8:
		return max64(700*MicrosPerStonky, wagerMicros)
	default:
		if streak > 8 && streak%4 == 0 {
			return max64(350*MicrosPerStonky, wagerMicros/3)
		}
		return 0
	}
}

func rushVaultReward(level int32, wagerMicros int64, win bool) int64 {
	base := int64(level+1) * 85 * MicrosPerStonky
	if win {
		base += wagerMicros / 3
	} else {
		base += wagerMicros / 6
	}
	return base
}

func clampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func rushStatusFromRow(current, best, rounds, wins, vaultPoints, vaultLevel int32, lastMode string, lastMult int32, lastPayout, lastVault int64, worldMomentum int32) RushStatus {
	winRate := int32(0)
	if rounds > 0 {
		winRate = int32(math.Round(float64(wins) * 10000.0 / float64(rounds)))
	}
	return RushStatus{
		CurrentStreak:         current,
		BestStreak:            best,
		RoundCount:            rounds,
		WinCount:              wins,
		WinRateBps:            winRate,
		VaultPoints:           vaultPoints,
		PointsToNextVault:     max32(0, nextRushVaultTarget(vaultLevel)-vaultPoints),
		VaultLevel:            vaultLevel,
		LastMode:              lastMode,
		LastMultiplierBps:     lastMult,
		LastPayoutMicros:      lastPayout,
		LastVaultRewardMicros: lastVault,
		WorldMomentumBps:      worldMomentum,
	}
}

func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func (s *Service) RushStatus(ctx context.Context, userID string, seasonID int64) (RushStatus, error) {
	var out RushStatus
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err := ensureRushProgressTx(ctx, tx, userID, seasonID); err != nil {
		return out, err
	}
	world, err := loadMarketWorldStateTx(ctx, tx, seasonID)
	if err != nil {
		return out, err
	}
	var current, best, rounds, wins, vaultPoints, vaultLevel int32
	var lastMode string
	var lastMult int32
	var lastPayout, lastVault int64
	if err := tx.QueryRow(ctx, `
		SELECT current_streak,
		       best_streak,
		       round_count,
		       win_count,
		       vault_points,
		       vault_level,
		       last_mode,
		       last_multiplier_bps,
		       last_payout_micros,
		       last_vault_reward_micros
		FROM game.rush_progress
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID).Scan(&current, &best, &rounds, &wins, &vaultPoints, &vaultLevel, &lastMode, &lastMult, &lastPayout, &lastVault); err != nil {
		return out, err
	}
	out = rushStatusFromRow(current, best, rounds, wins, vaultPoints, vaultLevel, lastMode, lastMult, lastPayout, lastVault, world.RiskRewardBiasBps)
	return out, tx.Commit(ctx)
}

func (s *Service) PlayRush(ctx context.Context, in RushPlayInput) (map[string]any, error) {
	out := map[string]any{}
	in.Mode = strings.ToLower(strings.TrimSpace(in.Mode))
	if in.Mode == "" {
		in.Mode = "steady"
	}
	spec, ok := rushModes[in.Mode]
	if !ok {
		return out, fmt.Errorf("mode must be one of: steady, surge, apex")
	}
	if in.AmountMicros <= 0 {
		return out, fmt.Errorf("amount must be > 0")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err := claimIdempotency(ctx, tx, in.UserID, in.IdempotencyKey, "play_rush"); err != nil {
		return out, err
	}
	if err := ensureRushProgressTx(ctx, tx, in.UserID, in.SeasonID); err != nil {
		return out, err
	}

	world, err := loadMarketWorldStateTx(ctx, tx, in.SeasonID)
	if err != nil {
		return out, err
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

	var current, best, rounds, wins, vaultPoints, vaultLevel int32
	if err := tx.QueryRow(ctx, `
		SELECT current_streak, best_streak, round_count, win_count, vault_points, vault_level
		FROM game.rush_progress
		WHERE user_id = $1 AND season_id = $2
		FOR UPDATE
	`, in.UserID, in.SeasonID).Scan(&current, &best, &rounds, &wins, &vaultPoints, &vaultLevel); err != nil {
		return out, err
	}

	worldBoost := float64(world.RiskRewardBiasBps) / 10000.0
	streakBoost := float64(minInt32(current, 8)) * 0.012
	winChance := clampFloat(spec.BaseWinChance+worldBoost*spec.WorldBonusFactor+streakBoost, 0.12, 0.82)
	multiplierBps := spec.BasePayoutBps + int32(math.Round(float64(spec.StreakBonusBps*minInt32(current, 10))))
	if world.RiskRewardBiasBps > 0 {
		multiplierBps += int32(math.Round(float64(world.RiskRewardBiasBps) * spec.WorldBonusFactor))
	}

	rounds++
	win := s.nextFloat() < winChance
	delta := -in.AmountMicros
	payout := int64(0)
	milestoneReward := int64(0)
	vaultReward := int64(0)
	vaultOpened := false

	if win {
		payout = int64(math.Round(float64(in.AmountMicros) * float64(multiplierBps) / 10000.0))
		delta = payout - in.AmountMicros
		wins++
		current++
		if current > best {
			best = current
		}
		milestoneReward = rushMilestoneReward(current, in.AmountMicros)
		delta += milestoneReward
		vaultPoints += spec.BasePoints + int32(in.AmountMicros/(220*MicrosPerStonky)) + current*2
	} else {
		current = 0
		vaultPoints += spec.BasePoints / 2
	}

	target := nextRushVaultTarget(vaultLevel)
	if vaultPoints >= target {
		vaultPoints -= target
		vaultReward = rushVaultReward(vaultLevel, in.AmountMicros, win)
		delta += vaultReward
		vaultLevel++
		vaultOpened = true
	}

	if _, err := tx.Exec(ctx, `
		UPDATE game.wallets
		SET balance_micros = balance_micros + $1,
		    updated_at = now()
		WHERE user_id = $2 AND season_id = $3
	`, delta, in.UserID, in.SeasonID); err != nil {
		return out, err
	}
	action := "rush_loss"
	if delta >= 0 {
		action = "rush_win"
	}
	if err := appendWalletDeltaEntry(ctx, tx, in.UserID, in.SeasonID, delta, action, map[string]any{
		"mode":                in.Mode,
		"wager_micros":        in.AmountMicros,
		"payout_micros":       payout,
		"milestone_reward":    milestoneReward,
		"vault_reward_micros": vaultReward,
		"world_momentum_bps":  world.RiskRewardBiasBps,
		"multiplier_bps":      multiplierBps,
		"win_chance_estimate": winChance,
		"vault_opened":        vaultOpened,
	}); err != nil {
		return out, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE game.rush_progress
		SET current_streak = $1,
		    best_streak = $2,
		    round_count = $3,
		    win_count = $4,
		    vault_points = $5,
		    vault_level = $6,
		    last_mode = $7,
		    last_multiplier_bps = $8,
		    last_payout_micros = $9,
		    last_vault_reward_micros = $10,
		    updated_at = now()
		WHERE user_id = $11 AND season_id = $12
	`, current, best, rounds, wins, vaultPoints, vaultLevel, in.Mode, multiplierBps, delta, vaultReward, in.UserID, in.SeasonID); err != nil {
		return out, err
	}

	var newBalance int64
	if err := tx.QueryRow(ctx, `
		SELECT balance_micros
		FROM game.wallets
		WHERE user_id = $1 AND season_id = $2
	`, in.UserID, in.SeasonID).Scan(&newBalance); err != nil {
		return out, err
	}

	if err := tx.Commit(ctx); err != nil {
		return out, err
	}

	status := rushStatusFromRow(current, best, rounds, wins, vaultPoints, vaultLevel, in.Mode, multiplierBps, delta, vaultReward, world.RiskRewardBiasBps)
	out["ok"] = true
	out["mode"] = in.Mode
	out["won"] = win
	out["wager_micros"] = in.AmountMicros
	out["multiplier_bps"] = multiplierBps
	out["net_delta_micros"] = delta
	out["milestone_reward_micros"] = milestoneReward
	out["vault_reward_micros"] = vaultReward
	out["vault_opened"] = vaultOpened
	out["balance_micros"] = newBalance
	out["status"] = status
	return out, nil
}

func minInt32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
