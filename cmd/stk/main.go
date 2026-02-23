package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	cl "stanks/internal/cli"
	"stanks/internal/config"
	"stanks/internal/game"
	"stanks/internal/syncq"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func main() {
	cfg := config.LoadCLIFromEnv()
	var apiBase string

	root := &cobra.Command{
		Use:   "stk",
		Short: "Stanks CLI game client",
	}
	root.PersistentFlags().StringVar(&apiBase, "api", cfg.APIBaseURL, "stanks api base url")

	root.AddCommand(
		newSignupCmd(&apiBase),
		newLoginCmd(&apiBase),
		newLogoutCmd(),
		newDashCmd(&apiBase),
		newSyncCmd(&apiBase),
		newStocksCmd(&apiBase),
		newBusinessCmd(&apiBase),
		newLeaderboardCmd(&apiBase),
		newFriendsCmd(&apiBase),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newClient(apiBase *string) *cl.Client {
	return cl.NewClient(strings.TrimRight(strings.TrimSpace(*apiBase), "/"))
}

func newSignupCmd(apiBase *string) *cobra.Command {
	var email, password, username string
	cmd := &cobra.Command{
		Use:   "signup",
		Short: "Create a Stanks account (required before playing)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" || password == "" {
				return fmt.Errorf("--email and --password are required")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			client := newClient(apiBase)
			session, err := client.Signup(ctx, email, password, username)
			if err != nil {
				return err
			}
			if strings.TrimSpace(session.AccessToken) == "" {
				fmt.Println("signup created. verify your email, then run: stk login")
				return nil
			}
			if err := cl.SaveSession(cl.Session{
				AccessToken:  session.AccessToken,
				RefreshToken: session.RefreshToken,
				Email:        session.User.Email,
				UserID:       session.User.ID,
			}); err != nil {
				return err
			}
			fmt.Println("signup complete and session saved")
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "email")
	cmd.Flags().StringVar(&password, "password", "", "password")
	cmd.Flags().StringVar(&username, "username", "", "username")
	return cmd
}

func newLoginCmd(apiBase *string) *cobra.Command {
	var email, password string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to Stanks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" || password == "" {
				return fmt.Errorf("--email and --password are required")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			client := newClient(apiBase)
			session, err := client.Login(ctx, email, password)
			if err != nil {
				return err
			}
			if err := cl.SaveSession(cl.Session{
				AccessToken:  session.AccessToken,
				RefreshToken: session.RefreshToken,
				Email:        session.User.Email,
				UserID:       session.User.ID,
			}); err != nil {
				return err
			}
			fmt.Println("login successful")
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "email")
	cmd.Flags().StringVar(&password, "password", "", "password")
	return cmd
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear local session token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cl.ClearSession(); err != nil {
				return err
			}
			fmt.Println("logged out")
			return nil
		},
	}
}

func newDashCmd(apiBase *string) *cobra.Command {
	return &cobra.Command{
		Use:   "dash",
		Short: "Show your dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			client := newClient(apiBase)
			out, err := client.Dashboard(ctx, sess.AccessToken)
			if err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}
}

func newSyncCmd(apiBase *string) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Replay locally queued offline writes to cloud",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			queue, err := syncq.Load()
			if err != nil {
				return err
			}
			if len(queue) == 0 {
				fmt.Println("sync queue is empty")
				return nil
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			remaining := make([]syncq.Command, 0, len(queue))
			success := 0
			for _, q := range queue {
				_, err := client.Do(ctx, q.Method, q.Path, sess.AccessToken, q.Body, q.IdempotencyKey)
				if err != nil {
					remaining = append(remaining, q)
					fmt.Fprintf(os.Stderr, "sync failed for %s %s: %v\n", q.Method, q.Path, err)
					continue
				}
				success++
			}
			if err := syncq.Save(remaining); err != nil {
				return err
			}
			fmt.Printf("sync complete: replayed=%d remaining=%d\n", success, len(remaining))
			return nil
		},
	}
}

func newStocksCmd(apiBase *string) *cobra.Command {
	stocks := &cobra.Command{
		Use:     "stocks",
		Short:   "Stock market commands",
		Aliases: []string{"stock"},
	}

	stocks.AddCommand(newStocksListCmd(apiBase))
	stocks.AddCommand(newStocksBuyCmd(apiBase))
	stocks.AddCommand(newStocksSellCmd(apiBase))
	stocks.AddCommand(newStocksCreateCmd(apiBase))
	stocks.AddCommand(newStocksIPOCmd(apiBase))

	return stocks
}

func newStocksListCmd(apiBase *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list [all|SYMBOL]",
		Short: "List stocks or inspect one stock",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			client := newClient(apiBase)

			if len(args) == 0 {
				out, err := client.ListStocks(ctx, sess.AccessToken, false)
				if err != nil {
					return err
				}
				printJSON(out)
				return nil
			}
			arg := strings.ToUpper(strings.TrimSpace(args[0]))
			if arg == "ALL" {
				out, err := client.ListStocks(ctx, sess.AccessToken, true)
				if err != nil {
					return err
				}
				printJSON(out)
				return nil
			}
			out, err := client.StockDetail(ctx, sess.AccessToken, arg)
			if err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}
}

func newStocksBuyCmd(apiBase *string) *cobra.Command {
	var qty float64
	cmd := &cobra.Command{
		Use:   "buy <symbol>",
		Short: "Buy shares",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return placeOrderCommand(cmd, apiBase, "buy", args[0], qty)
		},
	}
	cmd.Flags().Float64Var(&qty, "qty", 1.0, "share quantity")
	return cmd
}

func newStocksSellCmd(apiBase *string) *cobra.Command {
	var qty float64
	cmd := &cobra.Command{
		Use:   "sell <symbol>",
		Short: "Sell shares",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return placeOrderCommand(cmd, apiBase, "sell", args[0], qty)
		},
	}
	cmd.Flags().Float64Var(&qty, "qty", 1.0, "share quantity")
	return cmd
}

func placeOrderCommand(cmd *cobra.Command, apiBase *string, side, symbol string, qty float64) error {
	sess, err := cl.LoadSession()
	if err != nil {
		return fmt.Errorf("login required: %w", err)
	}
	units, err := game.SharesToUnits(qty)
	if err != nil {
		return err
	}
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	idem := uuid.NewString()
	body := map[string]any{
		"symbol":         symbol,
		"side":           side,
		"quantity_units": units,
	}

	client := newClient(apiBase)
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()
	out, err := client.PlaceOrder(ctx, sess.AccessToken, symbol, side, idem, units)
	if err != nil {
		return queueOnNetworkError(err, syncq.Command{
			Method:         "POST",
			Path:           "/v1/orders",
			Body:           body,
			IdempotencyKey: idem,
		})
	}
	printJSON(out)
	return nil
}

func newStocksCreateCmd(apiBase *string) *cobra.Command {
	var name string
	var businessID int64
	cmd := &cobra.Command{
		Use:   "create <symbol>",
		Short: "Create your own stock for one of your businesses",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			if name == "" || businessID <= 0 {
				return fmt.Errorf("--name and --business are required")
			}
			idem := uuid.NewString()
			body := map[string]any{
				"symbol":       strings.ToUpper(strings.TrimSpace(args[0])),
				"display_name": name,
				"business_id":  businessID,
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.CreateStock(ctx, sess.AccessToken, body["symbol"].(string), name, businessID, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           "/v1/stocks/custom",
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			printJSON(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "display name")
	cmd.Flags().Int64Var(&businessID, "business", 0, "business id")
	return cmd
}

func newStocksIPOCmd(apiBase *string) *cobra.Command {
	var price float64
	cmd := &cobra.Command{
		Use:   "ipo <symbol>",
		Short: "List a created stock publicly",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			if price <= 0 {
				return fmt.Errorf("--price must be > 0")
			}
			symbol := strings.ToUpper(strings.TrimSpace(args[0]))
			priceMicros := game.StonkyToMicros(price)
			idem := uuid.NewString()
			body := map[string]any{"price_micros": priceMicros}
			path := "/v1/stocks/" + symbol + "/ipo"
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.IPOStock(ctx, sess.AccessToken, symbol, priceMicros, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			printJSON(out)
			return nil
		},
	}
	cmd.Flags().Float64Var(&price, "price", 0, "ipo price in stonky")
	return cmd
}

func newBusinessCmd(apiBase *string) *cobra.Command {
	business := &cobra.Command{
		Use:     "business",
		Short:   "Business management commands",
		Aliases: []string{"bussin"},
	}
	business.AddCommand(newBusinessCreateCmd(apiBase))
	business.AddCommand(newBusinessStateCmd(apiBase))
	business.AddCommand(newBusinessVisibilityCmd(apiBase))
	business.AddCommand(newBusinessIPOCmd(apiBase))
	business.AddCommand(newBusinessEmployeesCmd(apiBase))
	return business
}

func newBusinessCreateCmd(apiBase *string) *cobra.Command {
	var visibility string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a business (requires progression)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			name := strings.TrimSpace(args[0])
			idem := uuid.NewString()
			body := map[string]any{"name": name, "visibility": visibility}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.CreateBusiness(ctx, sess.AccessToken, name, visibility, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           "/v1/businesses",
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			printJSON(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&visibility, "visibility", "private", "private|public")
	return cmd
}

func newBusinessStateCmd(apiBase *string) *cobra.Command {
	return &cobra.Command{
		Use:   "state <business_id>",
		Short: "Show business state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid business id")
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.BusinessState(ctx, sess.AccessToken, id)
			if err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}
}

func newBusinessVisibilityCmd(apiBase *string) *cobra.Command {
	return &cobra.Command{
		Use:   "visibility <business_id> <private|public>",
		Short: "Set business visibility",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid business id")
			}
			visibility := strings.ToLower(strings.TrimSpace(args[1]))
			idem := uuid.NewString()
			body := map[string]any{"visibility": visibility}
			path := fmt.Sprintf("/v1/businesses/%d/visibility", id)
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.SetBusinessVisibility(ctx, sess.AccessToken, id, visibility, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			printJSON(out)
			return nil
		},
	}
}

func newBusinessIPOCmd(apiBase *string) *cobra.Command {
	var symbol string
	var price float64
	cmd := &cobra.Command{
		Use:   "ipo <business_id>",
		Short: "List a public business on market",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid business id")
			}
			if symbol == "" || price <= 0 {
				return fmt.Errorf("--symbol and --price are required")
			}
			idem := uuid.NewString()
			priceMicros := game.StonkyToMicros(price)
			body := map[string]any{
				"symbol":       strings.ToUpper(strings.TrimSpace(symbol)),
				"price_micros": priceMicros,
			}
			path := fmt.Sprintf("/v1/businesses/%d/ipo", id)
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.BusinessIPO(ctx, sess.AccessToken, id, body["symbol"].(string), priceMicros, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			printJSON(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&symbol, "symbol", "", "6-char stock symbol")
	cmd.Flags().Float64Var(&price, "price", 0, "ipo price in stonky")
	return cmd
}

func newBusinessEmployeesCmd(apiBase *string) *cobra.Command {
	employees := &cobra.Command{
		Use:   "employees",
		Short: "Employee operations",
	}
	employees.AddCommand(&cobra.Command{
		Use:   "list <business_id>",
		Short: "List employees hired by your business",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			businessID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid business id")
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.ListBusinessEmployees(ctx, sess.AccessToken, businessID)
			if err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	})
	employees.AddCommand(&cobra.Command{
		Use:   "candidates",
		Short: "List candidates available for hire",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.ListEmployeeCandidates(ctx, sess.AccessToken)
			if err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	})
	employees.AddCommand(&cobra.Command{
		Use:   "hire <business_id> <candidate_id>",
		Short: "Hire a candidate for your business",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			businessID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid business id")
			}
			candidateID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid candidate id")
			}
			idem := uuid.NewString()
			body := map[string]any{"candidate_id": candidateID}
			path := fmt.Sprintf("/v1/businesses/%d/employees/hire", businessID)
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.HireEmployee(ctx, sess.AccessToken, businessID, candidateID, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			printJSON(out)
			return nil
		},
	})
	return employees
}

func newLeaderboardCmd(apiBase *string) *cobra.Command {
	lb := &cobra.Command{
		Use:   "leaderboard",
		Short: "Leaderboard commands",
	}
	lb.AddCommand(&cobra.Command{
		Use:   "global",
		Short: "Global leaderboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.LeaderboardGlobal(ctx, sess.AccessToken)
			if err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	})
	lb.AddCommand(&cobra.Command{
		Use:   "friends",
		Short: "Friends leaderboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.LeaderboardFriends(ctx, sess.AccessToken)
			if err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	})
	return lb
}

func newFriendsCmd(apiBase *string) *cobra.Command {
	friends := &cobra.Command{
		Use:   "friends",
		Short: "Manage friends by invite code",
	}
	friends.AddCommand(&cobra.Command{
		Use:   "add <invite_code>",
		Short: "Follow a user using invite code",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			idem := uuid.NewString()
			code := strings.ToUpper(strings.TrimSpace(args[0]))
			body := map[string]any{"invite_code": code}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.AddFriend(ctx, sess.AccessToken, code, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           "/v1/friends",
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			printJSON(out)
			return nil
		},
	})
	friends.AddCommand(&cobra.Command{
		Use:   "remove <invite_code>",
		Short: "Unfollow a user using invite code",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			code := strings.ToUpper(strings.TrimSpace(args[0]))
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.RemoveFriend(ctx, sess.AccessToken, code)
			if err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	})
	return friends
}

func queueOnNetworkError(err error, entry syncq.Command) error {
	if err == nil {
		return nil
	}
	if isAPIStructuredError(err) {
		return err
	}
	if pushErr := syncq.Push(entry); pushErr != nil {
		return fmt.Errorf("request failed (%v) and queue write failed (%v)", err, pushErr)
	}
	fmt.Fprintf(os.Stderr, "request failed (%v), queued locally for `stk sync`\n", err)
	return nil
}

func isAPIStructuredError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "api status")
}

func printJSON(v any) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println(v)
		return
	}
	fmt.Println(string(raw))
}
