package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	adminapi "stanks/internal/admin"
	"stanks/internal/config"
	"stanks/internal/game"
	"github.com/spf13/cobra"
)

type adminStore struct {
	baseURL  string
	client   *http.Client
	username string
	password string
}

type playerRow = adminapi.Player
type businessRow = adminapi.Business
type positionRow = adminapi.Position
type stockRow = adminapi.Stock

var stdinReader = bufio.NewReader(os.Stdin)

func main() {
	loadAdminEnv()

	cfg := config.LoadCLIFromEnv()

	store := &adminStore{
		baseURL: cfg.APIBaseURL,
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
	root := &cobra.Command{
		Use:           "admin",
		Short:         "Administrative control CLI for Stanks",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			switch cmd.Name() {
			case "help", "completion":
				return nil
			}
			return requireAdminAccess(store)
		},
	}

	root.AddCommand(
		newPlayersCmd(store),
		newShowCmd(store),
		newChangeBalanceCmd(store),
		newSetBalanceCmd(store),
		newChangePeakCmd(store),
		newSetPeakCmd(store),
		newSetActiveBusinessCmd(store),
		newListBusinessesCmd(store),
		newSetBusinessNameCmd(store),
		newSetBusinessVisibilityCmd(store),
		newSetBusinessListedCmd(store),
		newSetBusinessRevenueCmd(store),
		newDeleteBusinessCmd(store),
		newListPositionsCmd(store),
		newSetPositionCmd(store),
		newDeletePositionCmd(store),
		newListStocksCmd(store),
		newSetStockPriceCmd(store),
		newSelectCmd(store),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newPlayersCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:     "players [query]",
		Aliases: []string{"player-list"},
		Short:   "List players in the active season",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			query := ""
			if len(args) > 0 {
				query = strings.TrimSpace(args[0])
			}
			rows, err := store.ListPlayers(ctx, query)
			if err != nil {
				return err
			}
			printPlayers(rows)
			return nil
		},
	}
}

func newShowCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "show <user-id>",
		Short: "Show player summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			return printPlayerSummary(ctx, store, strings.TrimSpace(args[0]))
		},
	}
}

func newChangeBalanceCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:     "change-bal <user-id> <stonky-delta>",
		Aliases: []string{"change_bal"},
		Short:   "Add or subtract from wallet balance using normal stonky values",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			deltaMicros, err := parseStonkyArg(args[1])
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			row, err := store.ChangeBalance(ctx, strings.TrimSpace(args[0]), deltaMicros)
			if err != nil {
				return err
			}
			fmt.Printf("Balance updated for %s -> %s stonky\n", row.UserID, formatMicros(row.BalanceMicros))
			return nil
		},
	}
}

func newSetBalanceCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "set-bal <user-id> <stonky-amount>",
		Short: "Set wallet balance using a normal stonky value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			amountMicros, err := parseStonkyArg(args[1])
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			row, err := store.SetBalance(ctx, strings.TrimSpace(args[0]), amountMicros)
			if err != nil {
				return err
			}
			fmt.Printf("Balance set for %s -> %s stonky\n", row.UserID, formatMicros(row.BalanceMicros))
			return nil
		},
	}
}

func newChangePeakCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:     "change-peak <user-id> <stonky-delta>",
		Aliases: []string{"change_peak"},
		Short:   "Add or subtract from peak net worth using normal stonky values",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			deltaMicros, err := parseStonkyArg(args[1])
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			row, err := store.ChangePeak(ctx, strings.TrimSpace(args[0]), deltaMicros)
			if err != nil {
				return err
			}
			fmt.Printf("Peak updated for %s -> %s stonky\n", row.UserID, formatMicros(row.PeakNetWorthMicros))
			return nil
		},
	}
}

func newSetPeakCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "set-peak <user-id> <stonky-amount>",
		Short: "Set peak net worth using a normal stonky value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			amountMicros, err := parseStonkyArg(args[1])
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			row, err := store.SetPeak(ctx, strings.TrimSpace(args[0]), amountMicros)
			if err != nil {
				return err
			}
			fmt.Printf("Peak set for %s -> %s stonky\n", row.UserID, formatMicros(row.PeakNetWorthMicros))
			return nil
		},
	}
}

func newSetActiveBusinessCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "set-active-business <user-id> <business-id|0>",
		Short: "Set the player's active business, or 0 to clear it",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			businessID, err := strconv.ParseInt(strings.TrimSpace(args[1]), 10, 64)
			if err != nil || businessID < 0 {
				return fmt.Errorf("business id must be >= 0")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			row, err := store.SetActiveBusiness(ctx, strings.TrimSpace(args[0]), businessID)
			if err != nil {
				return err
			}
			if row.ActiveBusinessID == nil {
				fmt.Printf("Active business cleared for %s\n", row.UserID)
				return nil
			}
			fmt.Printf("Active business set for %s -> %d\n", row.UserID, *row.ActiveBusinessID)
			return nil
		},
	}
}

func newListBusinessesCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "businesses <user-id>",
		Short: "List a player's businesses",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			rows, err := store.ListBusinessesByUser(ctx, strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}
			printBusinesses(rows)
			return nil
		},
	}
}

func newSetBusinessNameCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "set-business-name <business-id> <name>",
		Short: "Rename a business",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			businessID, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
			if err != nil || businessID <= 0 {
				return fmt.Errorf("business id must be > 0")
			}
			name := strings.TrimSpace(args[1])
			if name == "" {
				return fmt.Errorf("name is required")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			row, err := store.SetBusinessName(ctx, businessID, name)
			if err != nil {
				return err
			}
			fmt.Printf("Business %d renamed to %q\n", row.ID, row.Name)
			return nil
		},
	}
}

func newSetBusinessVisibilityCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "set-business-visibility <business-id> <private|public>",
		Short: "Set business visibility",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			businessID, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
			if err != nil || businessID <= 0 {
				return fmt.Errorf("business id must be > 0")
			}
			visibility := strings.ToLower(strings.TrimSpace(args[1]))
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			row, err := store.SetBusinessVisibility(ctx, businessID, visibility)
			if err != nil {
				return err
			}
			fmt.Printf("Business %d visibility -> %s\n", row.ID, row.Visibility)
			return nil
		},
	}
}

func newSetBusinessListedCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "set-business-listed <business-id> <true|false>",
		Short: "Set business listed state",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			businessID, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
			if err != nil || businessID <= 0 {
				return fmt.Errorf("business id must be > 0")
			}
			listed, err := strconv.ParseBool(strings.TrimSpace(args[1]))
			if err != nil {
				return fmt.Errorf("listed must be true or false")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			row, err := store.SetBusinessListed(ctx, businessID, listed)
			if err != nil {
				return err
			}
			fmt.Printf("Business %d listed -> %t\n", row.ID, row.IsListed)
			return nil
		},
	}
}

func newSetBusinessRevenueCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "set-business-revenue <business-id> <stonky-amount>",
		Short: "Set base business revenue using normal stonky value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			businessID, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
			if err != nil || businessID <= 0 {
				return fmt.Errorf("business id must be > 0")
			}
			amountMicros, err := parseStonkyArg(args[1])
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			row, err := store.SetBusinessRevenue(ctx, businessID, amountMicros)
			if err != nil {
				return err
			}
			fmt.Printf("Business %d revenue -> %s stonky\n", row.ID, formatMicros(row.BaseRevenueMicros))
			return nil
		},
	}
}

func newDeleteBusinessCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "delete-business <business-id>",
		Short: "Delete a business",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			businessID, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
			if err != nil || businessID <= 0 {
				return fmt.Errorf("business id must be > 0")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			return store.DeleteBusiness(ctx, businessID)
		},
	}
}

func newListPositionsCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "positions <user-id>",
		Short: "List a player's stock positions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			rows, err := store.ListPositionsByUser(ctx, strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}
			printPositions(rows)
			return nil
		},
	}
}

func newSetPositionCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "set-position <user-id> <symbol> <shares> <avg-price>",
		Short: "Create or replace a player's stock position",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			shares, err := strconv.ParseFloat(strings.TrimSpace(args[2]), 64)
			if err != nil {
				return fmt.Errorf("invalid shares: %w", err)
			}
			priceMicros, err := parseStonkyArg(args[3])
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			row, err := store.SetPosition(ctx, strings.TrimSpace(args[0]), strings.ToUpper(strings.TrimSpace(args[1])), shares, priceMicros)
			if err != nil {
				return err
			}
			fmt.Printf("Position set for %s -> %s %0.4f @ %s stonky\n", args[0], row.Symbol, game.UnitsToShares(row.QuantityUnits), formatMicros(row.AvgPriceMicros))
			return nil
		},
	}
}

func newDeletePositionCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "delete-position <user-id> <symbol>",
		Short: "Delete a player's stock position",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			return store.DeletePosition(ctx, strings.TrimSpace(args[0]), strings.ToUpper(strings.TrimSpace(args[1])))
		},
	}
}

func newListStocksCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "stocks",
		Short: "List active season stocks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			rows, err := store.ListStocks(ctx)
			if err != nil {
				return err
			}
			printStocks(rows)
			return nil
		},
	}
}

func newSetStockPriceCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "set-stock-price <symbol> <stonky-price>",
		Short: "Force-set a stock's current and anchor price",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			priceMicros, err := parseStonkyArg(args[1])
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			row, err := store.SetStockPrice(ctx, strings.ToUpper(strings.TrimSpace(args[0])), priceMicros)
			if err != nil {
				return err
			}
			fmt.Printf("Stock %s -> %s stonky\n", row.Symbol, formatMicros(row.CurrentPriceMicros))
			return nil
		},
	}
}

func newSelectCmd(store *adminStore) *cobra.Command {
	return &cobra.Command{
		Use:   "select <user-id>",
		Short: "Interactively manage a player",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return runSelectLoop(ctx, store, strings.TrimSpace(args[0]))
		},
	}
}
