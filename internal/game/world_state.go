package game

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type marketWorldState struct {
	Regime                 string
	PoliticalClimate       string
	PolicyFocus            string
	CatalystName           string
	CatalystSummary        string
	CatalystTicksRemaining int32
	Headline               string
	AmericasBps            int32
	EuropeBps              int32
	AsiaBps                int32
	RiskRewardBiasBps      int32
}

func ensurePlayerProgressTx(ctx context.Context, tx pgx.Tx, userID string, seasonID int64) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO game.player_progress (user_id, season_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, season_id) DO NOTHING
	`, userID, seasonID)
	return err
}

func ensureMarketStateTx(ctx context.Context, tx pgx.Tx, seasonID int64) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO game.market_state (season_id, regime, updated_at)
		VALUES ($1, 'neutral', now())
		ON CONFLICT (season_id) DO NOTHING
	`, seasonID)
	return err
}

func loadMarketWorldStateTx(ctx context.Context, tx pgx.Tx, seasonID int64) (marketWorldState, error) {
	var out marketWorldState
	if err := ensureMarketStateTx(ctx, tx, seasonID); err != nil {
		return out, err
	}
	err := tx.QueryRow(ctx, `
		SELECT regime,
		       political_climate,
		       policy_focus,
		       catalyst_name,
		       catalyst_summary,
		       catalyst_ticks_remaining,
		       headline,
		       americas_bps,
		       europe_bps,
		       asia_bps,
		       risk_reward_bias_bps
		FROM game.market_state
		WHERE season_id = $1
	`, seasonID).Scan(
		&out.Regime,
		&out.PoliticalClimate,
		&out.PolicyFocus,
		&out.CatalystName,
		&out.CatalystSummary,
		&out.CatalystTicksRemaining,
		&out.Headline,
		&out.AmericasBps,
		&out.EuropeBps,
		&out.AsiaBps,
		&out.RiskRewardBiasBps,
	)
	return out, err
}

func (s *Service) playerProgress(ctx context.Context, userID string, seasonID int64) (PlayerProgress, error) {
	var out PlayerProgress
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err := ensurePlayerProgressTx(ctx, tx, userID, seasonID); err != nil {
		return out, err
	}
	if err := tx.QueryRow(ctx, `
		SELECT reputation_score,
		       current_profit_streak,
		       best_profit_streak,
		       risk_appetite_bps,
		       last_net_worth_delta_micros,
		       last_risk_payout_micros,
		       last_streak_reward_micros
		FROM game.player_progress
		WHERE user_id = $1 AND season_id = $2
	`, userID, seasonID).Scan(
		&out.ReputationScore,
		&out.CurrentProfitStreak,
		&out.BestProfitStreak,
		&out.RiskAppetiteBps,
		&out.LastNetWorthDeltaMicros,
		&out.LastRiskPayoutMicros,
		&out.LastStreakRewardMicros,
	); err != nil {
		return out, err
	}
	out.ReputationTitle = reputationTitle(out.ReputationScore)
	out.NextStreakTarget = nextStreakTarget(out.CurrentProfitStreak)
	return out, tx.Commit(ctx)
}

func (s *Service) WorldState(ctx context.Context, seasonID int64) (WorldView, error) {
	var out WorldView
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	state, err := loadMarketWorldStateTx(ctx, tx, seasonID)
	if err != nil {
		return out, err
	}
	out = marketWorldStateView(state)

	rows, err := tx.Query(ctx, `
		SELECT category, headline, impact_summary, created_at
		FROM game.world_events
		WHERE season_id = $1
		ORDER BY created_at DESC
		LIMIT 6
	`, seasonID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var ev WorldEventView
		if err := rows.Scan(&ev.Category, &ev.Headline, &ev.ImpactSummary, &ev.CreatedAt); err != nil {
			return out, err
		}
		out.RecentEvents = append(out.RecentEvents, ev)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

func marketWorldStateView(state marketWorldState) WorldView {
	return WorldView{
		Regime:                 state.Regime,
		PoliticalClimate:       state.PoliticalClimate,
		PolicyFocus:            state.PolicyFocus,
		CatalystName:           state.CatalystName,
		CatalystSummary:        state.CatalystSummary,
		CatalystTicksRemaining: state.CatalystTicksRemaining,
		Headline:               state.Headline,
		RiskRewardBiasBps:      state.RiskRewardBiasBps,
		Regions: []RegionView{
			{Name: "Americas", TrendBps: state.AmericasBps},
			{Name: "Europe", TrendBps: state.EuropeBps},
			{Name: "Asia", TrendBps: state.AsiaBps},
		},
	}
}

func reputationTitle(score int32) string {
	switch {
	case score >= 8500:
		return "Market Icon"
	case score >= 7000:
		return "Trusted Closer"
	case score >= 5500:
		return "Rising Operator"
	case score >= 4000:
		return "Speculative"
	default:
		return "Watched By Regulators"
	}
}

func nextStreakTarget(current int32) int32 {
	targets := []int32{3, 5, 8}
	for _, t := range targets {
		if current < t {
			return t
		}
	}
	return current + 3
}

func clampBps(v, minV, maxV int32) int32 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func pickBySeed(seed float64, values []string) string {
	if len(values) == 0 {
		return ""
	}
	idx := int(math.Floor(seed * float64(len(values))))
	if idx >= len(values) {
		idx = len(values) - 1
	}
	if idx < 0 {
		idx = 0
	}
	return values[idx]
}

func regionTrend(state marketWorldState, region string) float64 {
	switch strings.ToLower(strings.TrimSpace(region)) {
	case "americas":
		return float64(state.AmericasBps) / 10000.0
	case "europe":
		return float64(state.EuropeBps) / 10000.0
	case "asia":
		return float64(state.AsiaBps) / 10000.0
	default:
		return 0
	}
}

func stockRegion(symbol string) string {
	sum := 0
	for _, r := range strings.TrimSpace(symbol) {
		sum += int(r)
	}
	switch sum % 3 {
	case 0:
		return "americas"
	case 1:
		return "europe"
	default:
		return "asia"
	}
}

func stockSector(symbol string) string {
	sum := 0
	for _, r := range strings.TrimSpace(symbol) {
		sum += int(r)
	}
	sectors := []string{"technology", "energy", "finance", "consumer", "healthcare"}
	return sectors[sum%len(sectors)]
}

func policyDrift(policyFocus, subject string) float64 {
	if policyFocus == "" || subject == "" {
		return 0
	}
	if strings.EqualFold(strings.TrimSpace(policyFocus), strings.TrimSpace(subject)) {
		return 0.010
	}
	if strings.EqualFold(strings.TrimSpace(policyFocus), "broad_market") {
		return 0.003
	}
	return -0.002
}

func businessPolicySubject(focus string) string {
	switch strings.ToLower(strings.TrimSpace(focus)) {
	case "product":
		return "technology"
	case "brand":
		return "consumer"
	case "supply":
		return "energy"
	case "talent":
		return "healthcare"
	case "regulatory":
		return "finance"
	case "finance":
		return "finance"
	default:
		return "broad_market"
	}
}

func catalystSpec(seed float64) (name, summary string, ticks int32, bias int32) {
	switch {
	case seed < 0.20:
		return "Rate Decision", "Central banks set the tone for leverage and appetite.", 4, -350
	case seed < 0.40:
		return "Global Earnings Run", "Traders position for a full earnings cascade.", 5, 450
	case seed < 0.60:
		return "Supply Chain Reset", "Logistics bottlenecks are being repriced across regions.", 6, -200
	case seed < 0.80:
		return "Consumer Demand Sprint", "Retail and growth names are pricing in a demand burst.", 5, 320
	default:
		return "Capital Rotation", "Big money is rotating between defensive and speculative plays.", 6, 120
	}
}

func politicalClimateSpec(seed float64) (climate, focus, headline string) {
	switch {
	case seed < 0.20:
		return "stimulus_wave", "consumer", "Stimulus money is pushing demand-sensitive names higher."
	case seed < 0.40:
		return "tariff_cycle", "energy", "Tariff headlines are stressing cross-border businesses."
	case seed < 0.60:
		return "antitrust_wave", "technology", "Regulators are circling concentrated winners."
	case seed < 0.80:
		return "election_heat", "finance", "Election rhetoric is driving short-term repricing."
	default:
		return "steady_hand", "broad_market", "Policy is stable, so execution matters more than headlines."
	}
}

func businessNarrativeSeed(seed float64) (region, arc, focus string) {
	regions := []string{"americas", "europe", "asia"}
	arcs := []string{"steady", "breakout", "expansion", "fragile", "turnaround", "defensive"}
	focuses := []string{"product", "brand", "supply", "talent", "regulatory", "finance"}
	return pickBySeed(seed, regions), pickBySeed(math.Mod(seed*1.7, 1), arcs), pickBySeed(math.Mod(seed*2.3, 1), focuses)
}

func storyLabel(focus, arc string) string {
	return fmt.Sprintf("%s_%s", strings.TrimSpace(focus), strings.TrimSpace(arc))
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (s *Service) evolveWorldStateTx(ctx context.Context, tx pgx.Tx, seasonID int64) (marketWorldState, error) {
	state, err := loadMarketWorldStateTx(ctx, tx, seasonID)
	if err != nil {
		return state, err
	}

	if state.CatalystTicksRemaining <= 1 {
		state.CatalystName, state.CatalystSummary, state.CatalystTicksRemaining, state.RiskRewardBiasBps = catalystSpec(s.nextFloat())
	}
	state.CatalystTicksRemaining--
	if state.CatalystTicksRemaining < 0 {
		state.CatalystTicksRemaining = 0
	}

	if s.nextFloat() < 0.18 {
		state.PoliticalClimate, state.PolicyFocus, state.Headline = politicalClimateSpec(s.nextFloat())
	}

	state.AmericasBps = clampBps(state.AmericasBps+int32(math.Round(normalish(s.nextFloat())*240)), -1200, 1200)
	state.EuropeBps = clampBps(state.EuropeBps+int32(math.Round(normalish(s.nextFloat())*220)), -1200, 1200)
	state.AsiaBps = clampBps(state.AsiaBps+int32(math.Round(normalish(s.nextFloat())*260)), -1200, 1200)

	switch state.PoliticalClimate {
	case "stimulus_wave":
		state.AmericasBps = clampBps(state.AmericasBps+180, -1200, 1200)
		state.RiskRewardBiasBps = clampBps(state.RiskRewardBiasBps+160, -1500, 1500)
	case "tariff_cycle":
		state.AsiaBps = clampBps(state.AsiaBps-220, -1200, 1200)
		state.EuropeBps = clampBps(state.EuropeBps-120, -1200, 1200)
		state.RiskRewardBiasBps = clampBps(state.RiskRewardBiasBps-180, -1500, 1500)
	case "antitrust_wave":
		state.AmericasBps = clampBps(state.AmericasBps-90, -1200, 1200)
		state.RiskRewardBiasBps = clampBps(state.RiskRewardBiasBps-120, -1500, 1500)
	case "election_heat":
		state.AmericasBps = clampBps(state.AmericasBps+60, -1200, 1200)
		state.EuropeBps = clampBps(state.EuropeBps-40, -1200, 1200)
	case "steady_hand":
		state.RiskRewardBiasBps = clampBps(int32(math.Round(float64(state.RiskRewardBiasBps)*0.75)), -1500, 1500)
	}

	state.Headline = strings.TrimSpace(state.Headline + " Catalyst: " + state.CatalystName)
	if _, err := tx.Exec(ctx, `
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
	`, seasonID, state.Regime, state.PoliticalClimate, state.PolicyFocus, state.CatalystName, state.CatalystSummary, state.CatalystTicksRemaining, state.Headline, state.AmericasBps, state.EuropeBps, state.AsiaBps, state.RiskRewardBiasBps); err != nil {
		return state, err
	}
	return state, nil
}

func recordWorldEventTx(ctx context.Context, tx pgx.Tx, seasonID int64, category, headline, impact string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO game.world_events (season_id, category, headline, impact_summary)
		VALUES ($1, $2, $3, $4)
	`, seasonID, category, headline, impact)
	return err
}

func (s *Service) applyPlayerProgressionTx(ctx context.Context, tx pgx.Tx, seasonID int64, world marketWorldState) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO game.player_progress (user_id, season_id)
		SELECT user_id, season_id
		FROM game.wallets
		WHERE season_id = $1
		ON CONFLICT (user_id, season_id) DO NOTHING
	`, seasonID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `
		WITH holdings AS (
			SELECT p.user_id,
			       COALESCE(
			           LEAST(
			               $3::numeric,
			               GREATEST(
			                   $4::numeric,
			                   SUM((p.quantity_units::numeric * s.current_price_micros::numeric) / $2::numeric)
			               )
			           )::bigint,
			           0
			       ) AS holdings_micros,
			       COALESCE(
			           LEAST(
			               $3::numeric,
			               GREATEST(
			                   $4::numeric,
			                   MAX((p.quantity_units::numeric * s.current_price_micros::numeric) / $2::numeric)
			               )
			           )::bigint,
			           0
			       ) AS max_position_micros
			FROM game.positions p
			JOIN game.stocks s ON s.id = p.stock_id
			WHERE p.season_id = $1
			GROUP BY p.user_id
		),
		biz AS (
			SELECT b.owner_user_id AS user_id,
			       COALESCE(SUM(CASE WHEN b.strategy = 'aggressive' THEN 1 ELSE 0 END), 0) AS aggressive_count,
			       COALESCE(SUM(CASE WHEN b.is_listed THEN 1 ELSE 0 END), 0) AS listed_count,
			       COALESCE(
			           LEAST(
			               $3::numeric,
			               GREATEST(0::numeric, SUM(bl.outstanding_micros::numeric))
			           )::bigint,
			           0
			       ) AS loan_micros
			FROM game.businesses b
			LEFT JOIN game.business_loans bl
			  ON bl.business_id = b.id AND bl.season_id = b.season_id AND bl.status = 'open'
			WHERE b.season_id = $1
			GROUP BY b.owner_user_id
		)
		SELECT w.user_id,
		       w.balance_micros,
		       COALESCE(h.holdings_micros, 0),
		       COALESCE(h.max_position_micros, 0),
		       COALESCE(b.aggressive_count, 0),
		       COALESCE(b.listed_count, 0),
		       COALESCE(b.loan_micros, 0),
		       pp.reputation_score,
		       pp.current_profit_streak,
		       pp.best_profit_streak,
		       pp.streak_reward_tier,
		       pp.last_net_worth_micros
		FROM game.wallets w
		JOIN game.player_progress pp ON pp.user_id = w.user_id AND pp.season_id = w.season_id
		LEFT JOIN holdings h ON h.user_id = w.user_id
		LEFT JOIN biz b ON b.user_id = w.user_id
		WHERE w.season_id = $1
		FOR UPDATE OF w, pp
	`, seasonID, ShareScale, maxBigintMicros, minBigintMicros)
	if err != nil {
		return err
	}
	defer rows.Close()

	type row struct {
		userID             string
		balanceMicros      int64
		holdingsMicros     int64
		maxPositionMicros  int64
		aggressiveCount    int64
		listedCount        int64
		loanMicros         int64
		reputationScore    int32
		currentStreak      int32
		bestStreak         int32
		rewardTier         int32
		lastNetWorthMicros int64
	}
	var items []row
	for rows.Next() {
		var it row
		if err := rows.Scan(&it.userID, &it.balanceMicros, &it.holdingsMicros, &it.maxPositionMicros, &it.aggressiveCount, &it.listedCount, &it.loanMicros, &it.reputationScore, &it.currentStreak, &it.bestStreak, &it.rewardTier, &it.lastNetWorthMicros); err != nil {
			return err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, it := range items {
		netWorth := saturatingAddInt64(it.balanceMicros, it.holdingsMicros)
		if netWorth <= 0 {
			netWorth = 1
		}
		riskScore := int32(0)
		if it.holdingsMicros > 0 {
			shareRisk := float64(it.maxPositionMicros) * 5000.0 / float64(max64(it.holdingsMicros, 1))
			riskScore += clampBps(int32(math.Round(shareRisk)), 0, 5000)
		}
		if it.loanMicros > 0 {
			loanRisk := float64(it.loanMicros) * 3500.0 / float64(max64(netWorth, 1))
			riskScore += clampBps(int32(math.Round(loanRisk)), 0, 3500)
		}
		riskScore += clampBps(int32(it.aggressiveCount*700), 0, 2200)
		riskScore += clampBps(int32(it.listedCount*180), 0, 800)
		if it.balanceMicros < 0 {
			riskScore += 1200
		}
		riskScore = clampBps(riskScore, 0, 10000)

		delta := int64(0)
		if it.lastNetWorthMicros > 0 {
			delta = saturatingSubInt64(netWorth, it.lastNetWorthMicros)
		}

		currentStreak := it.currentStreak
		bestStreak := it.bestStreak
		rewardTier := it.rewardTier
		reputation := it.reputationScore
		streakReward := int64(0)
		riskPayout := int64(0)

		if delta > 0 {
			currentStreak++
			if currentStreak > bestStreak {
				bestStreak = currentStreak
			}
			reputation = clampBps(reputation+40+int32(riskScore/300), 0, 10000)
		} else if delta < 0 {
			currentStreak = 0
			rewardTier = 0
			reputation = clampBps(reputation-55-int32(riskScore/260), 0, 10000)
		}

		thresholds := []struct {
			streak int32
			reward int64
		}{
			{3, 350 * MicrosPerStonky},
			{5, 700 * MicrosPerStonky},
			{8, 1_500 * MicrosPerStonky},
		}
		for idx, threshold := range thresholds {
			if currentStreak >= threshold.streak && rewardTier < int32(idx+1) {
				streakReward += threshold.reward
				rewardTier = int32(idx + 1)
				reputation = clampBps(reputation+120, 0, 10000)
			}
		}

		if delta > 0 && world.RiskRewardBiasBps > 0 {
			riskPayout = int64(math.Round(float64(delta) * float64(riskScore) * float64(world.RiskRewardBiasBps) / 100000000.0))
		}
		if delta < 0 && world.RiskRewardBiasBps < 0 {
			riskPayout = -int64(math.Round(float64(-delta) * float64(riskScore) * float64(-world.RiskRewardBiasBps) / 100000000.0))
		}
		if riskPayout != 0 || streakReward != 0 {
			total := saturatingAddInt64(riskPayout, streakReward)
			if err := addWalletDeltaTx(ctx, tx, seasonID, it.userID, total); err != nil {
				return err
			}
			if err := appendWalletDeltaEntry(ctx, tx, it.userID, seasonID, total, "progression_payout", map[string]any{
				"risk_payout_micros":   riskPayout,
				"streak_reward_micros": streakReward,
			}); err != nil {
				return err
			}
		}

		nextNetWorth := saturatingAddInt64(saturatingAddInt64(netWorth, riskPayout), streakReward)
		if _, err := tx.Exec(ctx, `
			UPDATE game.player_progress
			SET reputation_score = $1,
			    current_profit_streak = $2,
			    best_profit_streak = $3,
			    streak_reward_tier = $4,
			    risk_appetite_bps = $5,
			    last_net_worth_micros = $6,
			    last_net_worth_delta_micros = $7,
			    last_risk_payout_micros = $8,
			    last_streak_reward_micros = $9,
			    updated_at = now()
			WHERE user_id = $10 AND season_id = $11
		`, reputation, currentStreak, bestStreak, rewardTier, riskScore, nextNetWorth, delta, riskPayout, streakReward, it.userID, seasonID); err != nil {
			return err
		}
	}
	return nil
}

func logProgressionEvent(ctx context.Context, tx pgx.Tx, seasonID int64, headline string, impact map[string]any) error {
	raw, _ := json.Marshal(impact)
	_, err := tx.Exec(ctx, `
		INSERT INTO game.ledger_entries (tx_group_id, user_id, season_id, account, delta_micros, metadata)
		VALUES ($1, '', $2, 'world', 0, $3::jsonb)
	`, uuid.NewString(), seasonID, string(raw))
	if err != nil {
		return err
	}
	return recordWorldEventTx(ctx, tx, seasonID, "progression", headline, string(raw))
}

func trimWorldEvents(ctx context.Context, tx pgx.Tx, seasonID int64) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM game.world_events
		WHERE season_id = $1
		  AND id NOT IN (
		      SELECT id
		      FROM game.world_events
		      WHERE season_id = $1
		      ORDER BY created_at DESC
		      LIMIT 24
		  )
	`, seasonID)
	return err
}

func staleEventCutoff() time.Time {
	return time.Now().Add(-14 * 24 * time.Hour)
}
