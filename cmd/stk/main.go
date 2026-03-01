package main

import (
	"context"
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
	apiBase := cfg.APIBaseURL

	root := &cobra.Command{
		Use:           "stk",
		Short:         "Stanks CLI game client",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		newSignupCmd(&apiBase),
		newLoginCmd(&apiBase),
		newLogoutCmd(),
		newDashCmd(&apiBase),
		newSyncCmd(&apiBase),
		newStocksCmd(&apiBase),
		newFundsCmd(&apiBase),
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
	return &cobra.Command{
		Use:   "signup",
		Short: "Create a Stanks account",
		RunE: func(cmd *cobra.Command, args []string) error {
			email, err := promptRequired("Email")
			if err != nil {
				return err
			}
			password, err := promptRequired("Password")
			if err != nil {
				return err
			}
			username, err := promptOptional("Username (optional)")
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			client := newClient(apiBase)
			session, err := client.Signup(ctx, email, password, username)
			if err != nil {
				return err
			}
			if strings.TrimSpace(session.AccessToken) == "" {
				printWarn("Signup created. Verify email, then run `stk login`.")
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
			printSuccess("Signup complete. Session saved.")
			return nil
		},
	}
}

func newLoginCmd(apiBase *string) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Login to Stanks",
		RunE: func(cmd *cobra.Command, args []string) error {
			email, err := promptRequired("Email")
			if err != nil {
				return err
			}
			password, err := promptRequired("Password")
			if err != nil {
				return err
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
			printSuccess("Login successful.")
			return nil
		},
	}
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear local session token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cl.ClearSession(); err != nil {
				return err
			}
			printSuccess("Logged out.")
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
			return renderDashboard(out)
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
				printInfo("Sync queue is empty.")
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
					printError(fmt.Sprintf("Sync failed for %s %s: %v", q.Method, q.Path, err))
					continue
				}
				success++
			}
			if err := syncq.Save(remaining); err != nil {
				return err
			}
			printSuccess(fmt.Sprintf("Sync complete: replayed=%d remaining=%d", success, len(remaining)))
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
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			client := newClient(apiBase)

			if len(args) == 0 {
				choice, err := promptChoice("View", []string{"market", "all", "symbol"}, "market")
				if err != nil {
					return err
				}
				switch choice {
				case "market":
					out, err := client.ListStocks(ctx, sess.AccessToken, false)
					if err != nil {
						return err
					}
					return renderStocksList(out)
				case "all":
					out, err := client.ListStocks(ctx, sess.AccessToken, true)
					if err != nil {
						return err
					}
					return renderStocksList(out)
				default:
					symbol, err := promptSymbol("Symbol")
					if err != nil {
						return err
					}
					out, err := client.StockDetail(ctx, sess.AccessToken, symbol)
					if err != nil {
						return err
					}
					return renderStockDetail(out)
				}
			}

			arg := strings.ToUpper(strings.TrimSpace(args[0]))
			if arg == "ALL" {
				out, err := client.ListStocks(ctx, sess.AccessToken, true)
				if err != nil {
					return err
				}
				return renderStocksList(out)
			}
			if arg == "MARKET" {
				out, err := client.ListStocks(ctx, sess.AccessToken, false)
				if err != nil {
					return err
				}
				return renderStocksList(out)
			}
			out, err := client.StockDetail(ctx, sess.AccessToken, arg)
			if err != nil {
				return err
			}
			return renderStockDetail(out)
		},
	}
}

func newStocksBuyCmd(apiBase *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy [symbol]",
		Short: "Buy shares",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol, err := symbolFromArgsOrPrompt(args)
			if err != nil {
				return err
			}
			qty, err := promptFloat("Shares to buy", 0)
			if err != nil {
				return err
			}
			return placeOrderCommand(cmd, apiBase, "buy", symbol, qty)
		},
	}
	return cmd
}

func newStocksSellCmd(apiBase *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sell [symbol]",
		Short: "Sell shares",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol, err := symbolFromArgsOrPrompt(args)
			if err != nil {
				return err
			}
			qty, err := promptFloat("Shares to sell", 0)
			if err != nil {
				return err
			}
			return placeOrderCommand(cmd, apiBase, "sell", symbol, qty)
		},
	}
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
	return renderOrderResult(out, side, symbol, qty)
}

func newStocksCreateCmd(apiBase *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [symbol]",
		Short: "Create your own stock for one of your businesses",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			symbol, err := symbolFromArgsOrPrompt(args)
			if err != nil {
				return err
			}
			name, err := promptRequired("Display name")
			if err != nil {
				return err
			}
			businessID, err := promptInt64("Business ID", 1)
			if err != nil {
				return err
			}
			idem := uuid.NewString()
			body := map[string]any{
				"symbol":       symbol,
				"display_name": name,
				"business_id":  businessID,
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.CreateStock(ctx, sess.AccessToken, symbol, name, businessID, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           "/v1/stocks/custom",
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			return renderSimpleOK(out, fmt.Sprintf("Created custom stock %s.", symbol))
		},
	}
	return cmd
}

func newStocksIPOCmd(apiBase *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ipo [symbol]",
		Short: "List a created stock publicly",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			symbol, err := symbolFromArgsOrPrompt(args)
			if err != nil {
				return err
			}
			price, err := promptFloat("IPO price (stonky)", 0)
			if err != nil {
				return err
			}
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
			return renderSimpleOK(out, fmt.Sprintf("IPO opened for %s at %s stonky.", symbol, formatMicros(priceMicros)))
		},
	}
	return cmd
}

func newBusinessCmd(apiBase *string) *cobra.Command {
	business := &cobra.Command{
		Use:     "business",
		Short:   "Business management commands",
		Aliases: []string{"bussin"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBusinessGuidedFlow(cmd, apiBase)
		},
	}
	business.AddCommand(newBusinessCreateCmd(apiBase))
	business.AddCommand(newBusinessStateCmd(apiBase))
	business.AddCommand(newBusinessVisibilityCmd(apiBase))
	business.AddCommand(newBusinessIPOCmd(apiBase))
	business.AddCommand(newBusinessEmployeesCmd(apiBase))
	business.AddCommand(newBusinessMachineryCmd(apiBase))
	business.AddCommand(newBusinessLoansCmd(apiBase))
	business.AddCommand(newBusinessStrategyCmd(apiBase))
	business.AddCommand(newBusinessUpgradesCmd(apiBase))
	business.AddCommand(newBusinessReserveCmd(apiBase))
	business.AddCommand(newBusinessSellCmd(apiBase))
	return business
}

func runBusinessGuidedFlow(cmd *cobra.Command, apiBase *string) error {
	sess, err := cl.LoadSession()
	if err != nil {
		return fmt.Errorf("login required: %w", err)
	}
	client := newClient(apiBase)
	ctx, cancel := context.WithTimeout(cmd.Context(), 45*time.Second)
	defer cancel()

	action, err := promptChoice("Business action", []string{
		"create", "state", "visibility", "ipo",
		"employees_candidates", "employees_list", "employees_hire", "employees_train",
		"machinery_list", "machinery_buy",
		"loans_list", "loans_take", "loans_repay",
		"strategy", "upgrade", "reserve_deposit", "reserve_withdraw",
		"sell",
	}, "state")
	if err != nil {
		return err
	}

	switch action {
	case "create":
		name, err := promptRequired("Business name")
		if err != nil {
			return err
		}
		visibility, err := promptChoice("Visibility", []string{"private", "public"}, "private")
		if err != nil {
			return err
		}
		idem := uuid.NewString()
		out, err := client.CreateBusiness(ctx, sess.AccessToken, name, visibility, idem)
		if err != nil {
			return err
		}
		return renderBusinessCreated(out, name, visibility)
	case "state":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		out, err := client.BusinessState(ctx, sess.AccessToken, id)
		if err != nil {
			return err
		}
		return renderBusinessState(out)
	case "visibility":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		visibility, err := promptChoice("Visibility", []string{"private", "public"}, "private")
		if err != nil {
			return err
		}
		idem := uuid.NewString()
		out, err := client.SetBusinessVisibility(ctx, sess.AccessToken, id, visibility, idem)
		if err != nil {
			return err
		}
		return renderSimpleOK(out, fmt.Sprintf("Business %d visibility set to %s.", id, visibility))
	case "ipo":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		symbol, err := promptSymbol("Stock symbol")
		if err != nil {
			return err
		}
		price, err := promptFloat("IPO price (stonky)", 0)
		if err != nil {
			return err
		}
		priceMicros := game.StonkyToMicros(price)
		idem := uuid.NewString()
		out, err := client.BusinessIPO(ctx, sess.AccessToken, id, symbol, priceMicros, idem)
		if err != nil {
			return err
		}
		return renderSimpleOK(out, fmt.Sprintf("Business %d IPO opened as %s at %s stonky.", id, symbol, formatMicros(priceMicros)))
	case "employees_candidates":
		out, err := client.ListEmployeeCandidates(ctx, sess.AccessToken)
		if err != nil {
			return err
		}
		return renderEmployeeCandidates(out)
	case "employees_list":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		out, err := client.ListBusinessEmployees(ctx, sess.AccessToken, id)
		if err != nil {
			return err
		}
		return renderBusinessEmployees(out, id)
	case "employees_hire":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		candidateID, err := promptInt64("Candidate ID", 1)
		if err != nil {
			return err
		}
		idem := uuid.NewString()
		out, err := client.HireEmployee(ctx, sess.AccessToken, id, candidateID, idem)
		if err != nil {
			return err
		}
		return renderSimpleOK(out, fmt.Sprintf("Hired candidate %d for business %d.", candidateID, id))
	case "employees_train":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		employeeID, err := promptInt64("Employee ID", 1)
		if err != nil {
			return err
		}
		idem := uuid.NewString()
		out, err := client.TrainProfessional(ctx, sess.AccessToken, id, employeeID, idem)
		if err != nil {
			return err
		}
		return renderSimpleOK(out, fmt.Sprintf("Trained employee %d in business %d.", employeeID, id))
	case "machinery_list":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		out, err := client.ListBusinessMachinery(ctx, sess.AccessToken, id)
		if err != nil {
			return err
		}
		return renderBusinessMachinery(out, id)
	case "machinery_buy":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		machineType, err := promptChoice("Machine type", []string{"assembly_line", "robotics_cell", "cloud_cluster", "bio_reactor", "quantum_rig"}, "assembly_line")
		if err != nil {
			return err
		}
		idem := uuid.NewString()
		out, err := client.BuyBusinessMachinery(ctx, sess.AccessToken, id, machineType, idem)
		if err != nil {
			return err
		}
		return renderSimpleOK(out, fmt.Sprintf("Installed %s for business %d.", machineType, id))
	case "loans_list":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		out, err := client.ListBusinessLoans(ctx, sess.AccessToken, id)
		if err != nil {
			return err
		}
		return renderBusinessLoans(out, id)
	case "loans_take":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		amount, err := promptFloat("Loan amount (stonky)", 0)
		if err != nil {
			return err
		}
		amountMicros := game.StonkyToMicros(amount)
		idem := uuid.NewString()
		out, err := client.TakeBusinessLoan(ctx, sess.AccessToken, id, amountMicros, idem)
		if err != nil {
			return err
		}
		return renderSimpleOK(out, fmt.Sprintf("Loan drawn for business %d: %s stonky.", id, formatMicros(amountMicros)))
	case "loans_repay":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		amount, err := promptFloat("Repay amount (stonky)", 0)
		if err != nil {
			return err
		}
		amountMicros := game.StonkyToMicros(amount)
		idem := uuid.NewString()
		out, err := client.RepayBusinessLoan(ctx, sess.AccessToken, id, amountMicros, idem)
		if err != nil {
			return err
		}
		return renderSimpleOK(out, fmt.Sprintf("Loan repayment submitted for business %d.", id))
	case "strategy":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		strategy, err := promptChoice("Strategy", []string{"aggressive", "balanced", "defensive"}, "balanced")
		if err != nil {
			return err
		}
		idem := uuid.NewString()
		out, err := client.SetBusinessStrategy(ctx, sess.AccessToken, id, strategy, idem)
		if err != nil {
			return err
		}
		return renderSimpleOK(out, fmt.Sprintf("Business %d strategy set to %s.", id, strategy))
	case "upgrade":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		upgrade, err := promptChoice("Upgrade", []string{"marketing", "rd", "automation", "compliance"}, "marketing")
		if err != nil {
			return err
		}
		idem := uuid.NewString()
		out, err := client.BuyBusinessUpgrade(ctx, sess.AccessToken, id, upgrade, idem)
		if err != nil {
			return err
		}
		return renderSimpleOK(out, fmt.Sprintf("Business %d upgraded: %s.", id, upgrade))
	case "reserve_deposit":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		amount, err := promptFloat("Deposit amount (stonky)", 0)
		if err != nil {
			return err
		}
		amountMicros := game.StonkyToMicros(amount)
		idem := uuid.NewString()
		out, err := client.BusinessReserveDeposit(ctx, sess.AccessToken, id, amountMicros, idem)
		if err != nil {
			return err
		}
		return renderSimpleOK(out, fmt.Sprintf("Business %d reserve deposit: %s stonky.", id, formatMicros(amountMicros)))
	case "reserve_withdraw":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		amount, err := promptFloat("Withdraw amount (stonky)", 0)
		if err != nil {
			return err
		}
		amountMicros := game.StonkyToMicros(amount)
		idem := uuid.NewString()
		out, err := client.BusinessReserveWithdraw(ctx, sess.AccessToken, id, amountMicros, idem)
		if err != nil {
			return err
		}
		return renderSimpleOK(out, fmt.Sprintf("Business %d reserve withdraw: %s stonky.", id, formatMicros(amountMicros)))
	case "sell":
		id, err := promptInt64("Business ID", 1)
		if err != nil {
			return err
		}
		idem := uuid.NewString()
		out, err := client.SellBusinessToBank(ctx, sess.AccessToken, id, idem)
		if err != nil {
			return err
		}
		return renderSimpleOK(out, fmt.Sprintf("Business %d sold to the bank.", id))
	default:
		return nil
	}
}

func runFundsGuidedFlow(cmd *cobra.Command, apiBase *string) error {
	sess, err := cl.LoadSession()
	if err != nil {
		return fmt.Errorf("login required: %w", err)
	}
	client := newClient(apiBase)
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	action, err := promptChoice("Funds action", []string{"list", "buy", "sell"}, "list")
	if err != nil {
		return err
	}
	switch action {
	case "list":
		out, err := client.ListFunds(ctx, sess.AccessToken)
		if err != nil {
			return err
		}
		return renderFundsList(out)
	case "buy", "sell":
		code, qty, err := fundCodeAndQty(nil)
		if err != nil {
			return err
		}
		units, err := game.SharesToUnits(qty)
		if err != nil {
			return err
		}
		idem := uuid.NewString()
		var out map[string]any
		if action == "buy" {
			out, err = client.BuyFund(ctx, sess.AccessToken, code, idem, units)
		} else {
			out, err = client.SellFund(ctx, sess.AccessToken, code, idem, units)
		}
		if err != nil {
			return err
		}
		label := "Bought"
		if action == "sell" {
			label = "Sold"
		}
		return renderSimpleOK(out, fmt.Sprintf("%s %.4f units of %s.", label, qty, code))
	default:
		return nil
	}
}

func newBusinessCreateCmd(apiBase *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a business (requires progression)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			name := ""
			if len(args) > 0 {
				name = strings.TrimSpace(args[0])
			} else {
				name, err = promptRequired("Business name")
				if err != nil {
					return err
				}
			}
			visibility, err := promptChoice("Visibility", []string{"private", "public"}, "private")
			if err != nil {
				return err
			}

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
			return renderBusinessCreated(out, name, visibility)
		},
	}
	return cmd
}

func newBusinessStateCmd(apiBase *string) *cobra.Command {
	return &cobra.Command{
		Use:   "state [business_id]",
		Short: "Show business state",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			id, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.BusinessState(ctx, sess.AccessToken, id)
			if err != nil {
				return err
			}
			return renderBusinessState(out)
		},
	}
}

func newBusinessVisibilityCmd(apiBase *string) *cobra.Command {
	return &cobra.Command{
		Use:   "visibility [business_id] [private|public]",
		Short: "Set business visibility",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			id, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}

			var visibility string
			if len(args) >= 2 {
				visibility = strings.ToLower(strings.TrimSpace(args[1]))
			} else {
				visibility, err = promptChoice("Visibility", []string{"private", "public"}, "private")
				if err != nil {
					return err
				}
			}

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
			return renderSimpleOK(out, fmt.Sprintf("Business %d visibility set to %s.", id, visibility))
		},
	}
}

func newBusinessIPOCmd(apiBase *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ipo [business_id]",
		Short: "List a public business on market",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			id, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}
			symbol, err := promptSymbol("Stock symbol")
			if err != nil {
				return err
			}
			price, err := promptFloat("IPO price (stonky)", 0)
			if err != nil {
				return err
			}

			idem := uuid.NewString()
			priceMicros := game.StonkyToMicros(price)
			body := map[string]any{
				"symbol":       symbol,
				"price_micros": priceMicros,
			}
			path := fmt.Sprintf("/v1/businesses/%d/ipo", id)
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.BusinessIPO(ctx, sess.AccessToken, id, symbol, priceMicros, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			return renderSimpleOK(out, fmt.Sprintf("Business %d IPO opened as %s at %s stonky.", id, symbol, formatMicros(priceMicros)))
		},
	}
	return cmd
}

func newBusinessEmployeesCmd(apiBase *string) *cobra.Command {
	employees := &cobra.Command{
		Use:   "employees",
		Short: "Employee operations",
	}
	employees.AddCommand(&cobra.Command{
		Use:   "list [business_id]",
		Short: "List employees hired by your business",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			businessID, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.ListBusinessEmployees(ctx, sess.AccessToken, businessID)
			if err != nil {
				return err
			}
			return renderBusinessEmployees(out, businessID)
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
			return renderEmployeeCandidates(out)
		},
	})
	employees.AddCommand(&cobra.Command{
		Use:   "hire [business_id] [candidate_id]",
		Short: "Hire a candidate for your business",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			businessID, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}
			candidateID, err := int64FromArgOrPrompt(args, 1, "Candidate ID")
			if err != nil {
				return err
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
			return renderSimpleOK(out, fmt.Sprintf("Hired candidate %d for business %d.", candidateID, businessID))
		},
	})
	employees.AddCommand(&cobra.Command{
		Use:   "train [business_id] [employee_id]",
		Short: "Train a professional to increase output (also raises risk)",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			businessID, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}
			employeeID, err := int64FromArgOrPrompt(args, 1, "Employee ID")
			if err != nil {
				return err
			}
			idem := uuid.NewString()
			path := fmt.Sprintf("/v1/businesses/%d/employees/%d/train", businessID, employeeID)
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.TrainProfessional(ctx, sess.AccessToken, businessID, employeeID, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           map[string]any{},
					IdempotencyKey: idem,
				})
			}
			return renderSimpleOK(out, fmt.Sprintf("Trained employee %d in business %d.", employeeID, businessID))
		},
	})
	return employees
}

func newBusinessMachineryCmd(apiBase *string) *cobra.Command {
	machinery := &cobra.Command{
		Use:   "machinery",
		Short: "Machinery operations for scaling business output",
	}
	machinery.AddCommand(&cobra.Command{
		Use:   "list [business_id]",
		Short: "List machinery installed in a business",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			businessID, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.ListBusinessMachinery(ctx, sess.AccessToken, businessID)
			if err != nil {
				return err
			}
			return renderBusinessMachinery(out, businessID)
		},
	})
	machinery.AddCommand(&cobra.Command{
		Use:   "buy [business_id] [machine_type]",
		Short: "Buy or upgrade machinery (assembly_line, robotics_cell, cloud_cluster, bio_reactor, quantum_rig)",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			businessID, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}
			machineType := ""
			if len(args) >= 2 {
				machineType = strings.ToLower(strings.TrimSpace(args[1]))
			} else {
				machineType, err = promptChoice("Machine type", []string{"assembly_line", "robotics_cell", "cloud_cluster", "bio_reactor", "quantum_rig"}, "assembly_line")
				if err != nil {
					return err
				}
			}
			idem := uuid.NewString()
			path := fmt.Sprintf("/v1/businesses/%d/machinery/buy", businessID)
			body := map[string]any{"machine_type": machineType}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.BuyBusinessMachinery(ctx, sess.AccessToken, businessID, machineType, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			return renderSimpleOK(out, fmt.Sprintf("Installed %s for business %d.", machineType, businessID))
		},
	})
	return machinery
}

func newBusinessLoansCmd(apiBase *string) *cobra.Command {
	loans := &cobra.Command{
		Use:   "loans",
		Short: "Business loan operations",
	}
	loans.AddCommand(&cobra.Command{
		Use:   "list [business_id]",
		Short: "List loans and delinquency status for a business",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			businessID, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.ListBusinessLoans(ctx, sess.AccessToken, businessID)
			if err != nil {
				return err
			}
			return renderBusinessLoans(out, businessID)
		},
	})
	loans.AddCommand(&cobra.Command{
		Use:   "take [business_id] [stonky]",
		Short: "Take a business loan",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			businessID, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}
			amount := 0.0
			if len(args) >= 2 {
				amount, err = strconv.ParseFloat(strings.TrimSpace(args[1]), 64)
				if err != nil || amount <= 0 {
					return fmt.Errorf("loan amount must be a positive number")
				}
			} else {
				amount, err = promptFloat("Loan amount (stonky)", 0)
				if err != nil {
					return err
				}
			}
			amountMicros := game.StonkyToMicros(amount)
			idem := uuid.NewString()
			path := fmt.Sprintf("/v1/businesses/%d/loans/take", businessID)
			body := map[string]any{"amount_micros": amountMicros}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.TakeBusinessLoan(ctx, sess.AccessToken, businessID, amountMicros, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			return renderSimpleOK(out, fmt.Sprintf("Loan drawn for business %d: %s stonky.", businessID, formatMicros(amountMicros)))
		},
	})
	loans.AddCommand(&cobra.Command{
		Use:   "repay [business_id] [stonky]",
		Short: "Repay business loans",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			businessID, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}
			amount := 0.0
			if len(args) >= 2 {
				amount, err = strconv.ParseFloat(strings.TrimSpace(args[1]), 64)
				if err != nil || amount <= 0 {
					return fmt.Errorf("repay amount must be a positive number")
				}
			} else {
				amount, err = promptFloat("Repay amount (stonky)", 0)
				if err != nil {
					return err
				}
			}
			amountMicros := game.StonkyToMicros(amount)
			idem := uuid.NewString()
			path := fmt.Sprintf("/v1/businesses/%d/loans/repay", businessID)
			body := map[string]any{"amount_micros": amountMicros}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.RepayBusinessLoan(ctx, sess.AccessToken, businessID, amountMicros, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			return renderSimpleOK(out, fmt.Sprintf("Loan repayment submitted for business %d.", businessID))
		},
	})
	return loans
}

func newBusinessSellCmd(apiBase *string) *cobra.Command {
	return &cobra.Command{
		Use:   "sell [business_id]",
		Short: "Sell your business to the bank at algorithmic valuation",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			businessID, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}
			idem := uuid.NewString()
			path := fmt.Sprintf("/v1/businesses/%d/sell", businessID)
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.SellBusinessToBank(ctx, sess.AccessToken, businessID, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           map[string]any{},
					IdempotencyKey: idem,
				})
			}
			return renderSimpleOK(out, fmt.Sprintf("Business %d sold to the bank.", businessID))
		},
	}
}

func newBusinessStrategyCmd(apiBase *string) *cobra.Command {
	return &cobra.Command{
		Use:   "strategy [business_id] [aggressive|balanced|defensive]",
		Short: "Set business strategy mode",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			businessID, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}
			strategy := ""
			if len(args) >= 2 {
				strategy = strings.ToLower(strings.TrimSpace(args[1]))
			} else {
				strategy, err = promptChoice("Strategy", []string{"aggressive", "balanced", "defensive"}, "balanced")
				if err != nil {
					return err
				}
			}
			idem := uuid.NewString()
			path := fmt.Sprintf("/v1/businesses/%d/strategy", businessID)
			body := map[string]any{"strategy": strategy}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.SetBusinessStrategy(ctx, sess.AccessToken, businessID, strategy, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			return renderSimpleOK(out, fmt.Sprintf("Business %d strategy set to %s.", businessID, strategy))
		},
	}
}

func newBusinessUpgradesCmd(apiBase *string) *cobra.Command {
	upgrades := &cobra.Command{
		Use:   "upgrades",
		Short: "Buy strategic business upgrades",
	}
	upgrades.AddCommand(&cobra.Command{
		Use:   "buy [business_id] [marketing|rd|automation|compliance]",
		Short: "Purchase an upgrade level",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			businessID, err := int64FromArgOrPrompt(args, 0, "Business ID")
			if err != nil {
				return err
			}
			upgrade := ""
			if len(args) >= 2 {
				upgrade = strings.ToLower(strings.TrimSpace(args[1]))
			} else {
				upgrade, err = promptChoice("Upgrade", []string{"marketing", "rd", "automation", "compliance"}, "marketing")
				if err != nil {
					return err
				}
			}
			idem := uuid.NewString()
			path := fmt.Sprintf("/v1/businesses/%d/upgrades/buy", businessID)
			body := map[string]any{"upgrade": upgrade}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.BuyBusinessUpgrade(ctx, sess.AccessToken, businessID, upgrade, idem)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			return renderSimpleOK(out, fmt.Sprintf("Business %d upgraded: %s.", businessID, upgrade))
		},
	})
	return upgrades
}

func newBusinessReserveCmd(apiBase *string) *cobra.Command {
	reserve := &cobra.Command{
		Use:   "reserve",
		Short: "Manage business reserve treasury",
	}
	reserve.AddCommand(&cobra.Command{
		Use:   "deposit [business_id] [stonky]",
		Short: "Move wallet cash into business reserve",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReserveTransfer(cmd, apiBase, args, "deposit")
		},
	})
	reserve.AddCommand(&cobra.Command{
		Use:   "withdraw [business_id] [stonky]",
		Short: "Withdraw cash from reserve to wallet",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReserveTransfer(cmd, apiBase, args, "withdraw")
		},
	})
	return reserve
}

func runReserveTransfer(cmd *cobra.Command, apiBase *string, args []string, direction string) error {
	sess, err := cl.LoadSession()
	if err != nil {
		return fmt.Errorf("login required: %w", err)
	}
	businessID, err := int64FromArgOrPrompt(args, 0, "Business ID")
	if err != nil {
		return err
	}
	amount := 0.0
	if len(args) >= 2 {
		amount, err = strconv.ParseFloat(strings.TrimSpace(args[1]), 64)
		if err != nil || amount <= 0 {
			return fmt.Errorf("amount must be a positive number")
		}
	} else {
		amount, err = promptFloat("Amount (stonky)", 0)
		if err != nil {
			return err
		}
	}
	amountMicros := game.StonkyToMicros(amount)
	idem := uuid.NewString()
	path := fmt.Sprintf("/v1/businesses/%d/reserve/%s", businessID, direction)
	body := map[string]any{"amount_micros": amountMicros}
	client := newClient(apiBase)
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()
	var out map[string]any
	if direction == "deposit" {
		out, err = client.BusinessReserveDeposit(ctx, sess.AccessToken, businessID, amountMicros, idem)
	} else {
		out, err = client.BusinessReserveWithdraw(ctx, sess.AccessToken, businessID, amountMicros, idem)
	}
	if err != nil {
		return queueOnNetworkError(err, syncq.Command{
			Method:         "POST",
			Path:           path,
			Body:           body,
			IdempotencyKey: idem,
		})
	}
	return renderSimpleOK(out, fmt.Sprintf("Business %d reserve %s: %s stonky.", businessID, direction, formatMicros(amountMicros)))
}

func newFundsCmd(apiBase *string) *cobra.Command {
	funds := &cobra.Command{
		Use:   "funds",
		Short: "Mutual fund style baskets",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFundsGuidedFlow(cmd, apiBase)
		},
	}
	funds.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available funds and NAV",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.ListFunds(ctx, sess.AccessToken)
			if err != nil {
				return err
			}
			return renderFundsList(out)
		},
	})
	funds.AddCommand(&cobra.Command{
		Use:   "buy [fund_code] [shares]",
		Short: "Buy fund units",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			code, qty, err := fundCodeAndQty(args)
			if err != nil {
				return err
			}
			units, err := game.SharesToUnits(qty)
			if err != nil {
				return err
			}
			idem := uuid.NewString()
			path := fmt.Sprintf("/v1/funds/%s/buy", code)
			body := map[string]any{"units": units}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.BuyFund(ctx, sess.AccessToken, code, idem, units)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			return renderSimpleOK(out, fmt.Sprintf("Bought %.4f units of %s.", qty, code))
		},
	})
	funds.AddCommand(&cobra.Command{
		Use:   "sell [fund_code] [shares]",
		Short: "Sell fund units",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			code, qty, err := fundCodeAndQty(args)
			if err != nil {
				return err
			}
			units, err := game.SharesToUnits(qty)
			if err != nil {
				return err
			}
			idem := uuid.NewString()
			path := fmt.Sprintf("/v1/funds/%s/sell", code)
			body := map[string]any{"units": units}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.SellFund(ctx, sess.AccessToken, code, idem, units)
			if err != nil {
				return queueOnNetworkError(err, syncq.Command{
					Method:         "POST",
					Path:           path,
					Body:           body,
					IdempotencyKey: idem,
				})
			}
			return renderSimpleOK(out, fmt.Sprintf("Sold %.4f units of %s.", qty, code))
		},
	})
	return funds
}

func fundCodeAndQty(args []string) (string, float64, error) {
	code := ""
	var qty float64
	var err error
	if len(args) >= 1 {
		code = strings.ToUpper(strings.TrimSpace(args[0]))
	} else {
		code, err = promptChoice("Fund code", []string{"TECH6X", "CORE20", "VOLT10", "DIVMAX", "AIEDGE", "STABLE"}, "CORE20")
		if err != nil {
			return "", 0, err
		}
		code = strings.ToUpper(code)
	}
	if len(args) >= 2 {
		qty, err = strconv.ParseFloat(strings.TrimSpace(args[1]), 64)
		if err != nil || qty <= 0 {
			return "", 0, fmt.Errorf("shares must be a positive number")
		}
	} else {
		qty, err = promptFloat("Units", 0)
		if err != nil {
			return "", 0, err
		}
	}
	return code, qty, nil
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
			return renderLeaderboard(out, "Global Leaderboard")
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
			return renderLeaderboard(out, "Friends Leaderboard")
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
		Use:   "add [invite_code]",
		Short: "Follow a user using invite code",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			code, err := inviteCodeFromArgsOrPrompt(args)
			if err != nil {
				return err
			}
			idem := uuid.NewString()
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
			return renderSimpleOK(out, fmt.Sprintf("Now following invite code %s.", code))
		},
	})
	friends.AddCommand(&cobra.Command{
		Use:   "remove [invite_code]",
		Short: "Unfollow a user using invite code",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := cl.LoadSession()
			if err != nil {
				return fmt.Errorf("login required: %w", err)
			}
			code, err := inviteCodeFromArgsOrPrompt(args)
			if err != nil {
				return err
			}
			client := newClient(apiBase)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			out, err := client.RemoveFriend(ctx, sess.AccessToken, code)
			if err != nil {
				return err
			}
			return renderSimpleOK(out, fmt.Sprintf("Stopped following invite code %s.", code))
		},
	})
	return friends
}

func queueOnNetworkError(err error, _ syncq.Command) error {
	if err == nil {
		return nil
	}
	if isAPIStructuredError(err) {
		return err
	}
	return fmt.Errorf("request failed (offline queue removed): %w", err)
}

func isAPIStructuredError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "api status")
}

func symbolFromArgsOrPrompt(args []string) (string, error) {
	if len(args) > 0 {
		symbol := strings.ToUpper(strings.TrimSpace(args[0]))
		if err := game.ValidateSymbol(symbol); err != nil {
			return "", err
		}
		return symbol, nil
	}
	return promptSymbol("Symbol")
}

func inviteCodeFromArgsOrPrompt(args []string) (string, error) {
	if len(args) > 0 {
		return strings.ToUpper(strings.TrimSpace(args[0])), nil
	}
	code, err := promptRequired("Invite code")
	if err != nil {
		return "", err
	}
	return strings.ToUpper(strings.TrimSpace(code)), nil
}

func int64FromArgOrPrompt(args []string, idx int, label string) (int64, error) {
	if len(args) > idx {
		v, err := strconv.ParseInt(strings.TrimSpace(args[idx]), 10, 64)
		if err != nil || v <= 0 {
			return 0, fmt.Errorf("invalid %s", strings.ToLower(label))
		}
		return v, nil
	}
	return promptInt64(label, 1)
}
