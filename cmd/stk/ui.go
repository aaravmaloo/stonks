package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"stanks/internal/game"

	"github.com/fatih/color"
)

var (
	stdinReader = bufio.NewReader(os.Stdin)
	accent      = color.New(color.FgCyan, color.Bold)
	success     = color.New(color.FgGreen, color.Bold)
	warn        = color.New(color.FgYellow, color.Bold)
	danger      = color.New(color.FgRed, color.Bold)
	neutral     = color.New(color.FgHiWhite)
)

type stocksPayload struct {
	Stocks []game.StockView `json:"stocks"`
}

type candidatesPayload struct {
	Candidates []employeeCandidate `json:"candidates"`
}

type businessEmployeesPayload struct {
	Employees []businessEmployee `json:"employees"`
}

type machineryPayload struct {
	Machinery []businessMachine `json:"machinery"`
}

type fundsPayload struct {
	Funds []fundView `json:"funds"`
}

type loansPayload struct {
	Loans []businessLoan `json:"loans"`
}

type leaderboardPayload struct {
	Rows []game.LeaderboardRow `json:"rows"`
}

type createBusinessPayload struct {
	ID int64 `json:"id"`
}

type employeeCandidate struct {
	ID                   int64  `json:"id"`
	FullName             string `json:"full_name"`
	Role                 string `json:"role"`
	Trait                string `json:"trait"`
	HireCostMicros       int64  `json:"hire_cost_micros"`
	RevenuePerTickMicros int64  `json:"revenue_per_tick_micros"`
	RiskBps              int32  `json:"risk_bps"`
}

type businessEmployee struct {
	ID                   int64     `json:"id"`
	FullName             string    `json:"full_name"`
	Role                 string    `json:"role"`
	Trait                string    `json:"trait"`
	RevenuePerTickMicros int64     `json:"revenue_per_tick_micros"`
	RiskBps              int32     `json:"risk_bps"`
	CreatedAt            time.Time `json:"created_at"`
}

type businessMachine struct {
	ID                int64     `json:"id"`
	MachineType       string    `json:"machine_type"`
	Level             int32     `json:"level"`
	OutputBonusMicros int64     `json:"output_bonus_micros"`
	UpkeepMicros      int64     `json:"upkeep_micros"`
	ReliabilityBps    int32     `json:"reliability_bps"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type fundView struct {
	Code       string   `json:"code"`
	Components []string `json:"components"`
	NavMicros  int64    `json:"nav_micros"`
}

type businessLoan struct {
	ID                int64     `json:"id"`
	PrincipalMicros   int64     `json:"principal_micros"`
	OutstandingMicros int64     `json:"outstanding_micros"`
	InterestBps       int32     `json:"interest_bps"`
	MissedTicks       int32     `json:"missed_ticks"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func printSuccess(msg string) {
	success.Println(msg)
}

func printWarn(msg string) {
	warn.Println(msg)
}

func printError(msg string) {
	danger.Println(msg)
}

func printInfo(msg string) {
	neutral.Println(msg)
}

func promptRequired(label string) (string, error) {
	for {
		fmt.Printf("%s: ", label)
		text, err := stdinReader.ReadString('\n')
		if err != nil {
			return "", err
		}
		text = strings.TrimSpace(text)
		if text != "" {
			return text, nil
		}
		printWarn(label + " is required.")
	}
}

func promptOptional(label string) (string, error) {
	fmt.Printf("%s: ", label)
	text, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func promptChoice(label string, options []string, defaultValue string) (string, error) {
	normalized := make(map[string]struct{}, len(options))
	for _, opt := range options {
		normalized[strings.ToLower(strings.TrimSpace(opt))] = struct{}{}
	}
	for {
		fmt.Printf("%s (%s) [%s]: ", label, strings.Join(options, "/"), defaultValue)
		text, err := stdinReader.ReadString('\n')
		if err != nil {
			return "", err
		}
		text = strings.ToLower(strings.TrimSpace(text))
		if text == "" {
			text = strings.ToLower(strings.TrimSpace(defaultValue))
		}
		if _, ok := normalized[text]; ok {
			return text, nil
		}
		printWarn("Invalid option. Please pick one of the listed values.")
	}
}

func promptFloat(label string, min float64) (float64, error) {
	for {
		text, err := promptRequired(label)
		if err != nil {
			return 0, err
		}
		v, err := strconv.ParseFloat(text, 64)
		if err != nil {
			printWarn("Enter a valid number.")
			continue
		}
		if v <= min {
			printWarn(fmt.Sprintf("Value must be > %.4f", min))
			continue
		}
		return v, nil
	}
}

func promptInt64(label string, min int64) (int64, error) {
	for {
		text, err := promptRequired(label)
		if err != nil {
			return 0, err
		}
		v, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			printWarn("Enter a whole number.")
			continue
		}
		if v < min {
			printWarn(fmt.Sprintf("Value must be >= %d", min))
			continue
		}
		return v, nil
	}
}

func promptSymbol(label string) (string, error) {
	for {
		symbol, err := promptRequired(label)
		if err != nil {
			return "", err
		}
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if err := game.ValidateSymbol(symbol); err != nil {
			printWarn(err.Error())
			continue
		}
		return symbol, nil
	}
}

func renderDashboard(raw map[string]any) error {
	d, err := decodeInto[game.Dashboard](raw)
	if err != nil {
		return err
	}

	accent.Printf("\n== DASHBOARD (Season %d) ==\n", d.SeasonID)
	startingPL := d.NetWorthMicros - game.StarterBalanceMicros
	openPL := int64(0)
	for _, p := range d.Positions {
		openPL += p.UnrealizedMicros
	}
	downFromPeak := d.NetWorthMicros - d.PeakNetWorthMicros

	fmt.Printf("Balance:            %s stonky\n", formatMicros(d.BalanceMicros))
	fmt.Printf("Net Worth:          %s stonky\n", formatMicros(d.NetWorthMicros))
	fmt.Printf("Peak Net Worth:     %s stonky\n", formatMicros(d.PeakNetWorthMicros))
	fmt.Printf("P/L vs Start:       %s stonky\n", colorizeMicros(startingPL))
	fmt.Printf("Open Position P/L:  %s stonky\n", colorizeMicros(openPL))
	fmt.Printf("From Peak:          %s stonky\n", colorizeMicros(downFromPeak))

	fmt.Println()
	accent.Println("Positions")
	if len(d.Positions) == 0 {
		printInfo("No open positions yet.")
	} else {
		fmt.Printf("%-8s %-22s %10s %12s %12s %12s %9s %14s %14s\n", "SYMBOL", "NAME", "QTY", "BUY", "NOW", "DELTA", "DELTA%", "VALUE", "P/L")
		for _, p := range d.Positions {
			valueMicros := orderNotional(p.CurrentPriceMicros, p.QuantityUnits)
			priceDeltaMicros := p.CurrentPriceMicros - p.AvgPriceMicros
			priceDeltaPct := 0.0
			if p.AvgPriceMicros != 0 {
				priceDeltaPct = (float64(priceDeltaMicros) / float64(p.AvgPriceMicros)) * 100
			}
			fmt.Printf("%-8s %-22s %10.4f %12s %12s %12s %9s %14s %14s\n",
				p.Symbol,
				truncate(p.DisplayName, 22),
				game.UnitsToShares(p.QuantityUnits),
				formatMicros(p.AvgPriceMicros),
				formatMicros(p.CurrentPriceMicros),
				colorizeMicros(priceDeltaMicros),
				colorizePercent(priceDeltaPct),
				formatMicros(valueMicros),
				colorizeMicros(p.UnrealizedMicros),
			)
		}
	}

	fmt.Println()
	accent.Println("Businesses")
	if len(d.Businesses) == 0 {
		printInfo("No businesses yet.")
	} else {
		fmt.Printf("%-6s %-20s %-9s %-8s %-10s %10s %8s %12s %12s %12s %10s\n", "ID", "NAME", "VISIBILITY", "LISTED", "STRATEGY", "EMPLOYEES", "MACH", "REV/TICK", "UPKEEP", "LOANS", "RESERVE")
		for _, b := range d.Businesses {
			listed := "no"
			if b.IsListed {
				listed = "yes"
			}
			fmt.Printf("%-6d %-20s %-9s %-8s %-10s %10d %8d %12s %12s %12s %10s\n",
				b.ID,
				truncate(b.Name, 20),
				b.Visibility,
				listed,
				truncate(b.Strategy, 10),
				b.EmployeeCount,
				b.MachineryCount,
				formatMicros(b.RevenuePerTickMicros),
				formatMicros(b.MachineryUpkeepMicros),
				formatMicros(b.LoanOutstandingMicros),
				formatMicros(b.CashReserveMicros),
			)
		}
	}
	fmt.Println()
	return nil
}

func renderStocksList(raw map[string]any) error {
	payload, err := decodeInto[stocksPayload](raw)
	if err != nil {
		return err
	}
	accent.Println("\n== STOCK MARKET ==")
	if len(payload.Stocks) == 0 {
		printInfo("No stocks found.")
		return nil
	}
	fmt.Printf("%-8s %-24s %12s %-8s\n", "SYMBOL", "NAME", "PRICE", "LISTED")
	for _, s := range payload.Stocks {
		listed := "yes"
		if !s.ListedPublic {
			listed = "no"
		}
		fmt.Printf("%-8s %-24s %12s %-8s\n",
			s.Symbol,
			truncate(s.DisplayName, 24),
			formatMicros(s.CurrentPriceMicros),
			listed,
		)
	}
	fmt.Println()
	return nil
}

func renderStockDetail(raw map[string]any) error {
	detail, err := decodeInto[game.StockDetail](raw)
	if err != nil {
		return err
	}
	accent.Printf("\n== %s (%s) ==\n", detail.Symbol, detail.DisplayName)
	fmt.Printf("Current Price: %s stonky\n", formatMicros(detail.CurrentPriceMicros))
	fmt.Printf("Listed Public: %t\n", detail.ListedPublic)

	if len(detail.Series) > 1 {
		latest := detail.Series[0].PriceMicros
		oldest := detail.Series[len(detail.Series)-1].PriceMicros
		delta := latest - oldest
		fmt.Printf("Trend (recent): %s stonky\n", colorizeMicros(delta))
	}

	if len(detail.Series) > 0 {
		fmt.Println()
		accent.Println("Recent Ticks")
		fmt.Printf("%-20s %12s\n", "TIME", "PRICE")
		limit := len(detail.Series)
		if limit > 8 {
			limit = 8
		}
		for i := 0; i < limit; i++ {
			point := detail.Series[i]
			fmt.Printf("%-20s %12s\n", point.TickAt.Local().Format("2006-01-02 15:04"), formatMicros(point.PriceMicros))
		}
	}
	fmt.Println()
	return nil
}

func renderOrderResult(raw map[string]any, side, symbol string, qty float64) error {
	out, err := decodeInto[game.OrderResult](raw)
	if err != nil {
		return err
	}
	action := strings.ToUpper(side)
	accent.Printf("\n== ORDER %s ==\n", action)
	fmt.Printf("Symbol:  %s\n", strings.ToUpper(symbol))
	fmt.Printf("Shares:  %.4f\n", qty)
	fmt.Printf("Price:   %s stonky\n", formatMicros(out.PriceMicros))
	fmt.Printf("Notional:%s stonky\n", formatMicros(out.NotionalMicros))
	fmt.Printf("Fee:     %s stonky\n", formatMicros(out.FeeMicros))
	fmt.Printf("Balance: %s stonky\n", formatMicros(out.BalanceMicros))
	fmt.Println()
	return nil
}

func renderBusinessCreated(raw map[string]any, name, visibility string) error {
	out, err := decodeInto[createBusinessPayload](raw)
	if err != nil {
		return err
	}
	printSuccess(fmt.Sprintf("Business created: #%d %s (%s)", out.ID, name, visibility))
	return nil
}

func renderBusinessState(raw map[string]any) error {
	out, err := decodeInto[game.BusinessView](raw)
	if err != nil {
		return err
	}
	accent.Printf("\n== BUSINESS #%d ==\n", out.ID)
	fmt.Printf("Name:        %s\n", out.Name)
	fmt.Printf("Visibility:  %s\n", out.Visibility)
	fmt.Printf("Listed:      %t\n", out.IsListed)
	fmt.Printf("Strategy:    %s\n", out.Strategy)
	fmt.Printf("Employees:   %d\n", out.EmployeeCount)
	fmt.Printf("Machinery:   %d\n", out.MachineryCount)
	fmt.Printf("Upgrades:    mkt=%d rd=%d auto=%d comp=%d\n", out.MarketingLevel, out.RDLevel, out.AutomationLevel, out.ComplianceLevel)
	fmt.Printf("Brand:       %.2f%%\n", float64(out.BrandBps)/100)
	fmt.Printf("Op Health:   %.2f%%\n", float64(out.OperationalHealthBps)/100)
	fmt.Printf("Reserve:     %s stonky\n", formatMicros(out.CashReserveMicros))
	fmt.Printf("Revenue/tick:%s stonky\n", formatMicros(out.RevenuePerTickMicros))
	fmt.Printf("Mach output: %s stonky\n", formatMicros(out.MachineryOutputMicros))
	fmt.Printf("Mach upkeep: %s stonky\n", formatMicros(out.MachineryUpkeepMicros))
	fmt.Printf("Loan debt:   %s stonky\n", formatMicros(out.LoanOutstandingMicros))
	if strings.TrimSpace(out.LastEvent) != "" {
		fmt.Printf("Last event:  %s\n", out.LastEvent)
	}
	fmt.Println()
	return nil
}

func renderEmployeeCandidates(raw map[string]any) error {
	out, err := decodeInto[candidatesPayload](raw)
	if err != nil {
		return err
	}
	accent.Println("\n== EMPLOYEE CANDIDATES ==")
	if len(out.Candidates) == 0 {
		printInfo("No candidates available.")
		return nil
	}
	fmt.Printf("%-4s %-18s %-10s %-12s %12s %12s %8s\n", "ID", "NAME", "ROLE", "TRAIT", "HIRE COST", "REV/TICK", "RISK")
	for _, c := range out.Candidates {
		fmt.Printf("%-4d %-18s %-10s %-12s %12s %12s %7.2f%%\n",
			c.ID,
			truncate(c.FullName, 18),
			truncate(c.Role, 10),
			truncate(c.Trait, 12),
			formatMicros(c.HireCostMicros),
			formatMicros(c.RevenuePerTickMicros),
			float64(c.RiskBps)/100,
		)
	}
	fmt.Println()
	return nil
}

func renderBusinessEmployees(raw map[string]any, businessID int64) error {
	out, err := decodeInto[businessEmployeesPayload](raw)
	if err != nil {
		return err
	}
	accent.Printf("\n== BUSINESS #%d EMPLOYEES ==\n", businessID)
	if len(out.Employees) == 0 {
		printInfo("No employees hired yet.")
		return nil
	}
	fmt.Printf("%-4s %-18s %-10s %-12s %12s %8s %-16s\n", "ID", "NAME", "ROLE", "TRAIT", "REV/TICK", "RISK", "HIRED")
	for _, e := range out.Employees {
		fmt.Printf("%-4d %-18s %-10s %-12s %12s %7.2f%% %-16s\n",
			e.ID,
			truncate(e.FullName, 18),
			truncate(e.Role, 10),
			truncate(e.Trait, 12),
			formatMicros(e.RevenuePerTickMicros),
			float64(e.RiskBps)/100,
			e.CreatedAt.Local().Format("2006-01-02 15:04"),
		)
	}
	fmt.Println()
	return nil
}

func renderBusinessMachinery(raw map[string]any, businessID int64) error {
	out, err := decodeInto[machineryPayload](raw)
	if err != nil {
		return err
	}
	accent.Printf("\n== BUSINESS #%d MACHINERY ==\n", businessID)
	if len(out.Machinery) == 0 {
		printInfo("No machinery installed yet.")
		return nil
	}
	fmt.Printf("%-4s %-16s %8s %12s %12s %10s\n", "ID", "TYPE", "LEVEL", "OUTPUT", "UPKEEP", "RELIAB.")
	for _, m := range out.Machinery {
		fmt.Printf("%-4d %-16s %8d %12s %12s %9.2f%%\n",
			m.ID,
			truncate(m.MachineType, 16),
			m.Level,
			formatMicros(m.OutputBonusMicros),
			formatMicros(m.UpkeepMicros),
			float64(m.ReliabilityBps)/100,
		)
	}
	fmt.Println()
	return nil
}

func renderBusinessLoans(raw map[string]any, businessID int64) error {
	out, err := decodeInto[loansPayload](raw)
	if err != nil {
		return err
	}
	accent.Printf("\n== BUSINESS #%d LOANS ==\n", businessID)
	if len(out.Loans) == 0 {
		printInfo("No loans on this business.")
		return nil
	}
	fmt.Printf("%-4s %12s %12s %9s %8s %-10s\n", "ID", "PRINCIPAL", "OUTSTAND", "RATE", "MISSED", "STATUS")
	for _, l := range out.Loans {
		fmt.Printf("%-4d %12s %12s %8.2f%% %8d %-10s\n",
			l.ID,
			formatMicros(l.PrincipalMicros),
			formatMicros(l.OutstandingMicros),
			float64(l.InterestBps)/100,
			l.MissedTicks,
			l.Status,
		)
	}
	fmt.Println()
	return nil
}

func renderFundsList(raw map[string]any) error {
	out, err := decodeInto[fundsPayload](raw)
	if err != nil {
		return err
	}
	accent.Println("\n== MUTUAL FUNDS ==")
	if len(out.Funds) == 0 {
		printInfo("No funds available.")
		return nil
	}
	fmt.Printf("%-8s %12s %-60s\n", "CODE", "NAV", "COMPONENTS")
	for _, f := range out.Funds {
		fmt.Printf("%-8s %12s %-60s\n",
			f.Code,
			formatMicros(f.NavMicros),
			truncate(strings.Join(f.Components, ","), 60),
		)
	}
	fmt.Println()
	return nil
}

func renderLeaderboard(raw map[string]any, title string) error {
	out, err := decodeInto[leaderboardPayload](raw)
	if err != nil {
		return err
	}
	accent.Printf("\n== %s ==\n", strings.ToUpper(title))
	if len(out.Rows) == 0 {
		printInfo("No leaderboard rows yet.")
		return nil
	}
	fmt.Printf("%-6s %-18s %-12s %14s\n", "RANK", "PLAYER", "INVITE", "NET WORTH")
	for _, row := range out.Rows {
		fmt.Printf("%-6d %-18s %-12s %14s\n",
			row.Rank,
			truncate(row.Username, 18),
			truncate(row.InviteCode, 12),
			formatMicros(row.NetWorthMicros),
		)
	}
	fmt.Println()
	return nil
}

func renderSimpleOK(raw map[string]any, successMessage string) error {
	ok := false
	if v, has := raw["ok"]; has {
		switch t := v.(type) {
		case bool:
			ok = t
		case string:
			ok = strings.EqualFold(strings.TrimSpace(t), "true")
		}
	}
	if ok || successMessage != "" {
		printSuccess(successMessage)
		return nil
	}
	printInfo("Done.")
	return nil
}

func decodeInto[T any](in any) (T, error) {
	var out T
	raw, err := json.Marshal(in)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}

func colorizeMicros(v int64) string {
	text := signedMicros(v)
	switch {
	case v > 0:
		return success.Sprint(text)
	case v < 0:
		return danger.Sprint(text)
	default:
		return neutral.Sprint(text)
	}
}

func colorizePercent(v float64) string {
	text := fmt.Sprintf("%+.2f%%", v)
	switch {
	case v > 0:
		return success.Sprint(text)
	case v < 0:
		return danger.Sprint(text)
	default:
		return neutral.Sprint(text)
	}
}

func formatMicros(v int64) string {
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	whole := v / game.MicrosPerStonky
	frac := (v % game.MicrosPerStonky) / 10_000
	return fmt.Sprintf("%s%s.%02d", sign, comma(whole), frac)
}

func signedMicros(v int64) string {
	if v > 0 {
		return "+" + formatMicros(v)
	}
	return formatMicros(v)
}

func comma(v int64) string {
	s := strconv.FormatInt(v, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
		if len(s) > pre {
			b.WriteByte(',')
		}
	}
	for i := pre; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteByte(',')
		}
	}
	return b.String()
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func orderNotional(priceMicros, qtyUnits int64) int64 {
	return (priceMicros*qtyUnits + (game.ShareScale / 2)) / game.ShareScale
}
