package main

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"stanks/internal/game"
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
		fmt.Println("7. Rename business")
		fmt.Println("8. Set business visibility")
		fmt.Println("9. Set business listed")
		fmt.Println("10. Set business revenue")
		fmt.Println("11. Delete business")
		fmt.Println("12. List positions")
		fmt.Println("13. Set position")
		fmt.Println("14. Delete position")
		fmt.Println("15. List stocks")
		fmt.Println("16. Set stock price")
		fmt.Println("17. Switch player")
		fmt.Println("18. Quit")

		choice, err := promptChoice("Select action", 18)
		if err != nil {
			return err
		}

		switch choice {
		case 1:
			delta, err := promptStonky("Balance delta")
			if err != nil {
				return err
			}
			row, err := store.changeBalance(ctx, currentUserID, delta)
			if err != nil {
				return err
			}
			fmt.Printf("Balance -> %s stonky\n", formatMicros(row.BalanceMicros))
		case 2:
			amount, err := promptStonky("Set balance to")
			if err != nil {
				return err
			}
			row, err := store.setBalance(ctx, currentUserID, amount)
			if err != nil {
				return err
			}
			fmt.Printf("Balance -> %s stonky\n", formatMicros(row.BalanceMicros))
		case 3:
			delta, err := promptStonky("Peak delta")
			if err != nil {
				return err
			}
			row, err := store.changePeak(ctx, currentUserID, delta)
			if err != nil {
				return err
			}
			fmt.Printf("Peak -> %s stonky\n", formatMicros(row.PeakNetWorthMicros))
		case 4:
			amount, err := promptStonky("Set peak to")
			if err != nil {
				return err
			}
			row, err := store.setPeak(ctx, currentUserID, amount)
			if err != nil {
				return err
			}
			fmt.Printf("Peak -> %s stonky\n", formatMicros(row.PeakNetWorthMicros))
		case 5:
			businessID, err := promptInt64("Business ID (0 to clear)")
			if err != nil {
				return err
			}
			row, err := store.setActiveBusiness(ctx, currentUserID, businessID)
			if err != nil {
				return err
			}
			if row.ActiveBusinessID == nil {
				fmt.Println("Active business cleared.")
			} else {
				fmt.Printf("Active business -> %d\n", *row.ActiveBusinessID)
			}
		case 6:
			rows, err := store.listBusinessesByUser(ctx, currentUserID)
			if err != nil {
				return err
			}
			printBusinesses(rows)
		case 7:
			businessID, err := promptInt64("Business ID")
			if err != nil {
				return err
			}
			name, err := promptRequired("New business name")
			if err != nil {
				return err
			}
			row, err := store.setBusinessName(ctx, businessID, name)
			if err != nil {
				return err
			}
			fmt.Printf("Business %d renamed to %q\n", row.ID, row.Name)
		case 8:
			businessID, err := promptInt64("Business ID")
			if err != nil {
				return err
			}
			visibility, err := promptRequired("Visibility (private/public)")
			if err != nil {
				return err
			}
			row, err := store.setBusinessVisibility(ctx, businessID, strings.ToLower(visibility))
			if err != nil {
				return err
			}
			fmt.Printf("Business %d visibility -> %s\n", row.ID, row.Visibility)
		case 9:
			businessID, err := promptInt64("Business ID")
			if err != nil {
				return err
			}
			listed, err := promptBool("Listed")
			if err != nil {
				return err
			}
			row, err := store.setBusinessListed(ctx, businessID, listed)
			if err != nil {
				return err
			}
			fmt.Printf("Business %d listed -> %t\n", row.ID, row.IsListed)
		case 10:
			businessID, err := promptInt64("Business ID")
			if err != nil {
				return err
			}
			revenue, err := promptStonky("Base revenue")
			if err != nil {
				return err
			}
			row, err := store.setBusinessRevenue(ctx, businessID, revenue)
			if err != nil {
				return err
			}
			fmt.Printf("Business %d revenue -> %s stonky\n", row.ID, formatMicros(row.BaseRevenueMicros))
		case 11:
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
			if err := store.deleteBusiness(ctx, businessID); err != nil {
				return err
			}
		case 12:
			rows, err := store.listPositionsByUser(ctx, currentUserID)
			if err != nil {
				return err
			}
			printPositions(rows)
		case 13:
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
			row, err := store.setPosition(ctx, currentUserID, strings.ToUpper(symbol), shares, price)
			if err != nil {
				return err
			}
			fmt.Printf("Position -> %s %0.4f @ %s stonky\n", row.Symbol, game.UnitsToShares(row.QuantityUnits), formatMicros(row.AvgPriceMicros))
		case 14:
			symbol, err := promptRequired("Stock symbol")
			if err != nil {
				return err
			}
			if err := store.deletePosition(ctx, currentUserID, strings.ToUpper(symbol)); err != nil {
				return err
			}
		case 15:
			rows, err := store.listStocks(ctx)
			if err != nil {
				return err
			}
			printStocks(rows)
		case 16:
			symbol, err := promptRequired("Stock symbol")
			if err != nil {
				return err
			}
			price, err := promptStonky("Set stock price to")
			if err != nil {
				return err
			}
			row, err := store.setStockPrice(ctx, strings.ToUpper(symbol), price)
			if err != nil {
				return err
			}
			fmt.Printf("Stock %s -> %s stonky\n", row.Symbol, formatMicros(row.CurrentPriceMicros))
		case 17:
			nextUserID, err := promptRequired("Player user ID")
			if err != nil {
				return err
			}
			currentUserID = nextUserID
		case 18:
			return nil
		}
	}
}

func printPlayerSummary(ctx context.Context, store *adminStore, userID string) error {
	row, err := store.playerByID(ctx, userID)
	if err != nil {
		return err
	}
	fmt.Printf("Player: %s (%s)\n", row.Username, row.UserID)
	fmt.Printf("Email: %s\n", row.Email)
	fmt.Printf("Invite: %s\n", row.InviteCode)
	fmt.Printf("Balance: %s stonky\n", formatMicros(row.BalanceMicros))
	fmt.Printf("Peak: %s stonky\n", formatMicros(row.PeakNetWorthMicros))
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
	fmt.Printf("%-26s  %-18s  %-16s  %-14s  %-14s\n", "USER ID", "USERNAME", "BALANCE", "PEAK", "ACTIVE BIZ")
	for _, row := range rows {
		active := "-"
		if row.ActiveBusinessID != nil {
			active = strconv.FormatInt(*row.ActiveBusinessID, 10)
		}
		fmt.Printf("%-26s  %-18s  %16s  %14s  %-14s\n",
			truncate(row.UserID, 26),
			truncate(row.Username, 18),
			formatMicros(row.BalanceMicros),
			formatMicros(row.PeakNetWorthMicros),
			active,
		)
	}
}

func printBusinesses(rows []businessRow) {
	if len(rows) == 0 {
		fmt.Println("No businesses found.")
		return
	}
	fmt.Printf("%-6s  %-24s  %-8s  %-6s  %-14s\n", "ID", "NAME", "VIS", "LIST", "BASE REV")
	for _, row := range rows {
		fmt.Printf("%-6d  %-24s  %-8s  %-6t  %14s\n",
			row.ID,
			truncate(row.Name, 24),
			row.Visibility,
			row.IsListed,
			formatMicros(row.BaseRevenueMicros),
		)
	}
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
