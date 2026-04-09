package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"stanks/internal/game"

	"golang.org/x/term"
)

func runSelectLoop(ctx context.Context, store *adminStore, userID string) error {
	currentUserID := strings.TrimSpace(userID)
	for {
		fmt.Println()
		if err := printPlayerSummary(ctx, store, currentUserID); err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("1. Change balance")
		fmt.Println("2. Set balance")
		fmt.Println("3. Change peak")
		fmt.Println("4. Set peak")
		fmt.Println("5. Set active business")
		fmt.Println("6. List businesses")
		fmt.Println("7. Set player progress")
		fmt.Println("8. Rename business")
		fmt.Println("9. Set business visibility")
		fmt.Println("10. Set business listed")
		fmt.Println("11. Set business revenue")
		fmt.Println("12. Set business narrative")
		fmt.Println("13. List business stakes")
		fmt.Println("14. Set business stake")
		fmt.Println("15. Delete business")
		fmt.Println("16. List positions")
		fmt.Println("17. Set position")
		fmt.Println("18. Delete position")
		fmt.Println("19. List stocks")
		fmt.Println("20. Set stock price")
		fmt.Println("21. Show world")
		fmt.Println("22. Switch player")
		fmt.Println("23. Quit")

		choice, err := promptChoice("Select action", 23)
		if err != nil {
			return err
		}

		switch choice {
		case 1:
			delta, err := promptStonky("Balance delta")
			if err != nil {
				return err
			}
			row, err := store.ChangeBalance(ctx, currentUserID, delta)
			if err != nil {
				return err
			}
			fmt.Printf("Balance -> %s stonky\n", formatMicros(row.BalanceMicros))
		case 2:
			amount, err := promptStonky("Set balance to")
			if err != nil {
				return err
			}
			row, err := store.SetBalance(ctx, currentUserID, amount)
			if err != nil {
				return err
			}
			fmt.Printf("Balance -> %s stonky\n", formatMicros(row.BalanceMicros))
		case 3:
			delta, err := promptStonky("Peak delta")
			if err != nil {
				return err
			}
			row, err := store.ChangePeak(ctx, currentUserID, delta)
			if err != nil {
				return err
			}
			fmt.Printf("Peak -> %s stonky\n", formatMicros(row.PeakNetWorthMicros))
		case 4:
			amount, err := promptStonky("Set peak to")
			if err != nil {
				return err
			}
			row, err := store.SetPeak(ctx, currentUserID, amount)
			if err != nil {
				return err
			}
			fmt.Printf("Peak -> %s stonky\n", formatMicros(row.PeakNetWorthMicros))
		case 5:
			businessID, err := promptInt64("Business ID (0 to clear)")
			if err != nil {
				return err
			}
			row, err := store.SetActiveBusiness(ctx, currentUserID, businessID)
			if err != nil {
				return err
			}
			if row.ActiveBusinessID == nil {
				fmt.Println("Active business cleared.")
			} else {
				fmt.Printf("Active business -> %d\n", *row.ActiveBusinessID)
			}
		case 6:
			rows, err := store.ListBusinessesByUser(ctx, currentUserID)
			if err != nil {
				return err
			}
			printBusinesses(rows)
		case 7:
			reputation, err := promptInt64("Reputation score")
			if err != nil {
				return err
			}
			currentStreak, err := promptInt64("Current streak")
			if err != nil {
				return err
			}
			bestStreak, err := promptInt64("Best streak")
			if err != nil {
				return err
			}
			riskBps, err := promptInt64("Risk appetite bps")
			if err != nil {
				return err
			}
			row, err := store.SetPlayerProgress(ctx, currentUserID, int32(reputation), int32(currentStreak), int32(bestStreak), int32(riskBps))
			if err != nil {
				return err
			}
			fmt.Printf("Progress -> rep=%d streak=%d/%d risk=%d\n", row.ReputationScore, row.CurrentProfitStreak, row.BestProfitStreak, row.RiskAppetiteBps)
		case 8:
			businessID, err := promptInt64("Business ID")
			if err != nil {
				return err
			}
			name, err := promptRequired("New business name")
			if err != nil {
				return err
			}
			row, err := store.SetBusinessName(ctx, businessID, name)
			if err != nil {
				return err
			}
			fmt.Printf("Business %d renamed to %q\n", row.ID, row.Name)
		case 9:
			businessID, err := promptInt64("Business ID")
			if err != nil {
				return err
			}
			visibility, err := promptRequired("Visibility (private/public)")
			if err != nil {
				return err
			}
			row, err := store.SetBusinessVisibility(ctx, businessID, strings.ToLower(visibility))
			if err != nil {
				return err
			}
			fmt.Printf("Business %d visibility -> %s\n", row.ID, row.Visibility)
		case 10:
			businessID, err := promptInt64("Business ID")
			if err != nil {
				return err
			}
			listed, err := promptBool("Listed")
			if err != nil {
				return err
			}
			row, err := store.SetBusinessListed(ctx, businessID, listed)
			if err != nil {
				return err
			}
			fmt.Printf("Business %d listed -> %t\n", row.ID, row.IsListed)
		case 11:
			businessID, err := promptInt64("Business ID")
			if err != nil {
				return err
			}
			revenue, err := promptStonky("Base revenue")
			if err != nil {
				return err
			}
			row, err := store.SetBusinessRevenue(ctx, businessID, revenue)
			if err != nil {
				return err
			}
			fmt.Printf("Business %d revenue -> %s stonky\n", row.ID, formatMicros(row.BaseRevenueMicros))
		case 12:
			businessID, err := promptInt64("Business ID")
			if err != nil {
				return err
			}
			region, err := promptRequired("Primary region")
			if err != nil {
				return err
			}
			arc, err := promptRequired("Narrative arc")
			if err != nil {
				return err
			}
			focus, err := promptRequired("Narrative focus")
			if err != nil {
				return err
			}
			pressure, err := promptInt64("Narrative pressure bps")
			if err != nil {
				return err
			}
			row, err := store.SetBusinessNarrative(ctx, businessID, region, arc, focus, int32(pressure))
			if err != nil {
				return err
			}
			fmt.Printf("Business %d narrative -> %s %s %s %d\n", row.ID, row.PrimaryRegion, row.NarrativeArc, row.NarrativeFocus, row.NarrativePressureBps)
		case 13:
			businessID, err := promptInt64("Business ID")
			if err != nil {
				return err
			}
			rows, err := store.ListBusinessStakes(ctx, businessID)
			if err != nil {
				return err
			}
			printStakes(rows)
		case 14:
			businessID, err := promptInt64("Business ID")
			if err != nil {
				return err
			}
			username, err := promptRequired("Username")
			if err != nil {
				return err
			}
			percent, err := promptFloat("Stake percent")
			if err != nil {
				return err
			}
			rows, err := store.SetBusinessStake(ctx, businessID, username, int32(math.Round(percent*100)))
			if err != nil {
				return err
			}
			printStakes(rows)
		case 15:
			businessID, err := promptInt64("Business ID")
			if err != nil {
				return err
			}
			confirm, err := promptBool("Delete this business")
			if err != nil {
				return err
			}
			if !confirm {
				fmt.Println("Delete cancelled.")
				continue
			}
			if err := store.DeleteBusiness(ctx, businessID); err != nil {
				return err
			}
		case 16:
			rows, err := store.ListPositionsByUser(ctx, currentUserID)
			if err != nil {
				return err
			}
			printPositions(rows)
		case 17:
			symbol, err := promptRequired("Stock symbol")
			if err != nil {
				return err
			}
			shares, err := promptFloat("Shares")
			if err != nil {
				return err
			}
			price, err := promptStonky("Average price")
			if err != nil {
				return err
			}
			row, err := store.SetPosition(ctx, currentUserID, strings.ToUpper(symbol), shares, price)
			if err != nil {
				return err
			}
			fmt.Printf("Position -> %s %0.4f @ %s stonky\n", row.Symbol, game.UnitsToShares(row.QuantityUnits), formatMicros(row.AvgPriceMicros))
		case 18:
			symbol, err := promptRequired("Stock symbol")
			if err != nil {
				return err
			}
			if err := store.DeletePosition(ctx, currentUserID, strings.ToUpper(symbol)); err != nil {
				return err
			}
		case 19:
			rows, err := store.ListStocks(ctx)
			if err != nil {
				return err
			}
			printStocks(rows)
		case 20:
			symbol, err := promptRequired("Stock symbol")
			if err != nil {
				return err
			}
			price, err := promptStonky("Set stock price to")
			if err != nil {
				return err
			}
			row, err := store.SetStockPrice(ctx, strings.ToUpper(symbol), price)
			if err != nil {
				return err
			}
			fmt.Printf("Stock %s -> %s stonky\n", row.Symbol, formatMicros(row.CurrentPriceMicros))
		case 21:
			row, err := store.WorldState(ctx)
			if err != nil {
				return err
			}
			printWorld(row)
		case 22:
			nextUserID, err := promptRequired("Player user ID")
			if err != nil {
				return err
			}
			currentUserID = nextUserID
		case 23:
			return nil
		}
	}
}

func printPlayerSummary(ctx context.Context, store *adminStore, userID string) error {
	row, err := store.PlayerByID(ctx, userID)
	if err != nil {
		return err
	}
	fmt.Printf("Player: %s (%s)\n", row.Username, row.UserID)
	fmt.Printf("Email: %s\n", row.Email)
	fmt.Printf("Invite: %s\n", row.InviteCode)
	fmt.Printf("Balance: %s stonky\n", formatMicros(row.BalanceMicros))
	fmt.Printf("Peak: %s stonky\n", formatMicros(row.PeakNetWorthMicros))
	fmt.Printf("Reputation: %d\n", row.ReputationScore)
	fmt.Printf("Profit Streak: %d (best %d)\n", row.CurrentProfitStreak, row.BestProfitStreak)
	fmt.Printf("Risk Appetite: %d bps\n", row.RiskAppetiteBps)
	if row.ActiveBusinessID == nil {
		fmt.Println("Active Business: none")
	} else {
		fmt.Printf("Active Business: %d\n", *row.ActiveBusinessID)
	}
	return nil
}

func printPlayers(rows []playerRow) {
	if len(rows) == 0 {
		fmt.Println("No players found.")
		return
	}
	fmt.Printf("%-26s  %-18s  %-16s  %-14s  %-10s  %-14s\n", "USER ID", "USERNAME", "BALANCE", "PEAK", "REPUTATION", "ACTIVE BIZ")
	for _, row := range rows {
		active := "-"
		if row.ActiveBusinessID != nil {
			active = strconv.FormatInt(*row.ActiveBusinessID, 10)
		}
		fmt.Printf("%-26s  %-18s  %16s  %14s  %-10d  %-14s\n",
			truncate(row.UserID, 26),
			truncate(row.Username, 18),
			formatMicros(row.BalanceMicros),
			formatMicros(row.PeakNetWorthMicros),
			row.ReputationScore,
			active,
		)
	}
}

func printBusinesses(rows []businessRow) {
	if len(rows) == 0 {
		fmt.Println("No businesses found.")
		return
	}
	fmt.Printf("%-6s  %-20s  %-8s  %-6s  %-10s  %-10s  %-9s\n", "ID", "NAME", "VIS", "LIST", "REGION", "ARC", "PRESSURE")
	for _, row := range rows {
		fmt.Printf("%-6d  %-20s  %-8s  %-6t  %-10s  %-10s  %9d\n",
			row.ID,
			truncate(row.Name, 20),
			row.Visibility,
			row.IsListed,
			truncate(row.PrimaryRegion, 10),
			truncate(row.NarrativeArc, 10),
			row.NarrativePressureBps,
		)
	}
}

func printStakes(rows []stakeRow) {
	if len(rows) == 0 {
		fmt.Println("No stakes found.")
		return
	}
	fmt.Printf("%-6s  %-18s  %-18s  %-10s  %-12s\n", "BIZ", "USER ID", "USERNAME", "STAKE", "COST BASIS")
	for _, row := range rows {
		fmt.Printf("%-6d  %-18s  %-18s  %9.2f%%  %12s\n",
			row.BusinessID,
			truncate(row.UserID, 18),
			truncate(row.Username, 18),
			float64(row.StakeBps)/100.0,
			formatMicros(row.CostBasisMicros),
		)
	}
}

func printWorld(row worldRow) {
	fmt.Printf("Regime: %s\n", row.Regime)
	fmt.Printf("Politics: %s\n", row.PoliticalClimate)
	fmt.Printf("Policy Focus: %s\n", row.PolicyFocus)
	fmt.Printf("Catalyst: %s (%d ticks)\n", row.CatalystName, row.CatalystTicksRemaining)
	fmt.Printf("Headline: %s\n", row.Headline)
	fmt.Printf("Summary: %s\n", row.CatalystSummary)
	fmt.Printf("Regions: americas=%d europe=%d asia=%d\n", row.AmericasBps, row.EuropeBps, row.AsiaBps)
	fmt.Printf("Risk Bias: %d\n", row.RiskRewardBiasBps)
}

func printPositions(rows []positionRow) {
	if len(rows) == 0 {
		fmt.Println("No positions found.")
		return
	}
	fmt.Printf("%-8s  %-22s  %-12s  %-12s\n", "SYMBOL", "NAME", "SHARES", "AVG PRICE")
	for _, row := range rows {
		fmt.Printf("%-8s  %-22s  %12.4f  %12s\n",
			row.Symbol,
			truncate(row.DisplayName, 22),
			game.UnitsToShares(row.QuantityUnits),
			formatMicros(row.AvgPriceMicros),
		)
	}
}

func printStocks(rows []stockRow) {
	if len(rows) == 0 {
		fmt.Println("No stocks found.")
		return
	}
	fmt.Printf("%-8s  %-22s  %-12s  %-12s  %-6s\n", "SYMBOL", "NAME", "PRICE", "ANCHOR", "PUBLIC")
	for _, row := range rows {
		fmt.Printf("%-8s  %-22s  %12s  %12s  %-6t\n",
			row.Symbol,
			truncate(row.DisplayName, 22),
			formatMicros(row.CurrentPriceMicros),
			formatMicros(row.AnchorPriceMicros),
			row.ListedPublic,
		)
	}
}

func parseStonkyArg(raw string) (int64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid stonky amount: %w", err)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("invalid stonky amount")
	}
	return game.StonkyToMicros(value), nil
}

func promptRequired(label string) (string, error) {
	fmt.Printf("%s: ", label)
	text, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	return text, nil
}

func promptInt64(label string) (int64, error) {
	raw, err := promptRequired(label)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer: %w", err)
	}
	return value, nil
}

func promptPassword(label string) (string, error) {
	fmt.Printf("%s: ", label)
	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	return text, nil
}

func promptFloat(label string) (float64, error) {
	raw, err := promptRequired(label)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %w", err)
	}
	return value, nil
}

func promptStonky(label string) (int64, error) {
	raw, err := promptRequired(label + " (stonky)")
	if err != nil {
		return 0, err
	}
	return parseStonkyArg(raw)
}

func promptChoice(label string, max int) (int, error) {
	raw, err := promptRequired(label)
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > max {
		return 0, fmt.Errorf("choose a number between 1 and %d", max)
	}
	return value, nil
}

func promptBool(label string) (bool, error) {
	raw, err := promptRequired(label + " (y/n)")
	if err != nil {
		return false, err
	}
	switch strings.ToLower(raw) {
	case "y", "yes", "true", "1":
		return true, nil
	case "n", "no", "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("enter y or n")
	}
}

func formatMicros(v int64) string {
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	whole := v / game.MicrosPerStonky
	frac := v % game.MicrosPerStonky
	return fmt.Sprintf("%s%s.%02d", sign, comma(whole), frac/10_000)
}

func comma(v int64) string {
	raw := strconv.FormatInt(v, 10)
	if len(raw) <= 3 {
		return raw
	}
	var parts []string
	for len(raw) > 3 {
		parts = append([]string{raw[len(raw)-3:]}, parts...)
		raw = raw[:len(raw)-3]
	}
	parts = append([]string{raw}, parts...)
	return strings.Join(parts, ",")
}

func truncate(raw string, max int) string {
	if len(raw) <= max {
		return raw
	}
	if max <= 3 {
		return raw[:max]
	}
	return raw[:max-3] + "..."
}
