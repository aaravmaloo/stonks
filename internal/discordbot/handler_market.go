package discordbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"stanks/internal/game"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (b *Bot) handleDashboard(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	raw, err := b.client.Dashboard(ctx, token)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[game.Dashboard](raw)
	if err != nil {
		return err
	}

	eb := NewEmbed().Title("Dashboard").Color(colorInfo).
		Field("Balance", fmtStonky(out.BalanceMicros), true).
		Field("Net Worth", fmtStonky(out.NetWorthMicros), true).
		Field("Peak", fmtStonky(out.PeakNetWorthMicros), true)

	if out.ActiveBusinessID != nil {
		eb.Field("Active Business", strconv.FormatInt(*out.ActiveBusinessID, 10), true)
	}

	if len(out.Positions) > 0 {
		lines := make([]string, 0, min(len(out.Positions), 5))
		for idx, pos := range out.Positions {
			if idx >= 5 {
				break
			}
			lines = append(lines, fmt.Sprintf("`%s` %s | %s", strings.TrimSpace(pos.Symbol), fmtShares(pos.QuantityUnits), fmtPL(pos.UnrealizedMicros)))
		}
		eb.Field("Positions", strings.Join(lines, "\n"), false)
	}

	if len(out.Businesses) > 0 {
		lines := make([]string, 0, min(len(out.Businesses), 5))
		for idx, biz := range out.Businesses {
			if idx >= 5 {
				break
			}
			lines = append(lines, fmt.Sprintf("`#%d` **%s** | rev/tick %s | %d/%d staff", biz.ID, biz.Name, fmtStonky(biz.RevenuePerTickMicros), biz.EmployeeCount, biz.EmployeeLimit))
		}
		eb.Field("Businesses", strings.Join(lines, "\n"), false)
	}

	if len(out.Businesses) == 0 && len(out.Positions) == 0 {
		eb.Desc("No positions or businesses yet. Use `/stocks` to browse the market.")
	}

	return b.respondEmbedWithComponents(s, i, eb.Build(), []discordgo.MessageComponent{dashboardButtons()})
}

func (b *Bot) handleWallet(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, email, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	raw, err := b.client.WalletSummary(ctx, token)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[game.WalletSummary](raw)
	if err != nil {
		return err
	}
	eb := NewEmbed().Title("Wallet").Color(colorInfo).Desc("Current account balance and progression.").
		Field("Email", email, true).
		Field("Balance", fmtStonky(out.BalanceMicros), true).
		Field("Peak Net Worth", fmtStonky(out.PeakNetWorthMicros), true)

	if out.ActiveBusinessID != nil {
		eb.Field("Active Business", strconv.FormatInt(*out.ActiveBusinessID, 10), true)
	}
	return b.respondEmbed(s, i, eb.Build())
}

func (b *Bot) handlePortfolio(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	raw, err := b.client.Dashboard(ctx, token)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[game.Dashboard](raw)
	if err != nil {
		return err
	}

	if len(out.Positions) == 0 {
		return b.respondEmbed(s, i, infoEmbed("Portfolio", "You have no stock positions. Use `/order` to buy shares.", nil))
	}

	lines := make([]string, 0, len(out.Positions))
	var totalPL int64
	for _, pos := range out.Positions {
		totalPL += pos.UnrealizedMicros
		lines = append(lines, fmt.Sprintf("`%s` %s shares @ %s | P/L: %s",
			strings.TrimSpace(pos.Symbol),
			fmtShares(pos.QuantityUnits),
			fmtStonky(pos.AvgPriceMicros),
			fmtPL(pos.UnrealizedMicros),
		))
	}

	color := colorInfo
	if totalPL > 0 {
		color = colorSuccess
	} else if totalPL < 0 {
		color = colorError
	}

	eb := NewEmbed().Title("Portfolio").Color(color).
		Desc(strings.Join(lines, "\n")).
		Field("Total P/L", fmtPL(totalPL), true).
		Field("Positions", strconv.Itoa(len(out.Positions)), true).
		Field("Balance", fmtStonky(out.BalanceMicros), true)

	return b.respondEmbed(s, i, eb.Build())
}

func (b *Bot) handleStocks(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	all := boolOption(i.ApplicationCommandData().Options, "all")
	raw, err := b.client.ListStocks(ctx, token, all)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[stocksPayload](raw)
	if err != nil {
		return err
	}
	if len(out.Stocks) == 0 {
		return b.respondEmbed(s, i, infoEmbed("Stocks", "No stocks are available right now.", nil))
	}

	return b.renderStockPage(s, i, out.Stocks, 0, all)
}

func (b *Bot) renderStockPage(s *discordgo.Session, i *discordgo.InteractionCreate, stocks []game.StockView, page int, all bool) error {
	totalPages := (len(stocks) + pageSize - 1) / pageSize
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * pageSize
	end := start + pageSize
	if end > len(stocks) {
		end = len(stocks)
	}

	lines := make([]string, 0, end-start)
	for _, stock := range stocks[start:end] {
		status := "private"
		if stock.ListedPublic {
			status = "public"
		}
		lines = append(lines, fmt.Sprintf("`%s` **%s** | %s | %s", strings.TrimSpace(stock.Symbol), stock.DisplayName, fmtStonky(stock.CurrentPriceMicros), status))
	}

	eb := NewEmbed().Title("Stock Market").Color(colorMarket).
		Desc(strings.Join(lines, "\n")).
		Field("Total", strconv.Itoa(len(stocks)), true).
		Field("Mode", ternary(all, "all", "public only"), true)

	allStr := ternary(all, "true", "false")
	components := []discordgo.MessageComponent{paginationRow("stocks", page, totalPages, allStr)}

	return b.respondEmbedWithComponents(s, i, eb.Build(), components)
}

func (b *Bot) handleStock(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	symbol := strings.ToUpper(strings.TrimSpace(stringOption(i.ApplicationCommandData().Options, "symbol", "")))
	raw, err := b.client.StockDetail(ctx, token, symbol)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[game.StockDetail](raw)
	if err != nil {
		return err
	}

	desc := fmt.Sprintf("**%s**\nCurrent price: %s", out.DisplayName, fmtStonky(out.CurrentPriceMicros))

	if len(out.Series) > 0 {
		prices := make([]int64, len(out.Series))
		for idx, p := range out.Series {
			prices[idx] = p.PriceMicros
		}
		desc += "\n\nPrice trend: `" + sparkline(prices) + "`"

		recent := out.Series
		if len(recent) > 5 {
			recent = recent[len(recent)-5:]
		}
		points := make([]string, 0, len(recent))
		for _, point := range recent {
			points = append(points, fmt.Sprintf("%s %s", point.TickAt.UTC().Format("Jan 02 15:04"), fmtStonky(point.PriceMicros)))
		}
		desc += "\n\n" + strings.Join(points, "\n")
	}

	eb := NewEmbed().Title("Stock | "+strings.TrimSpace(out.Symbol)).Color(colorMarket).
		Desc(desc).
		Field("Symbol", strings.TrimSpace(out.Symbol), true).
		Field("Listed", ternary(out.ListedPublic, "yes", "no"), true)

	return b.respondEmbedWithComponents(s, i, eb.Build(), []discordgo.MessageComponent{stockActionButtons(strings.TrimSpace(out.Symbol))})
}

func (b *Bot) handleOrder(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	symbol := strings.ToUpper(strings.TrimSpace(stringOption(data.Options, "symbol", "")))
	side := strings.ToLower(strings.TrimSpace(stringOption(data.Options, "side", "")))
	shares := numberOption(data.Options, "shares", 0)
	units, err := game.SharesToUnits(shares)
	if err != nil {
		return b.respondError(s, i, err.Error())
	}
	raw, err := b.client.PlaceOrder(ctx, token, symbol, side, uuid.NewString(), units)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[game.OrderResult](raw)
	if err != nil {
		return err
	}

	eb := NewEmbed().Title("Order Filled").Color(colorSuccess).
		Desc(fmt.Sprintf("%s %.4f shares of `%s`.", strings.Title(side), shares, symbol)).
		Field("Order ID", strconv.FormatInt(out.OrderID, 10), true).
		Field("Price", fmtStonky(out.PriceMicros), true).
		Field("Notional", fmtStonky(out.NotionalMicros), true).
		Field("Fee", fmtStonky(out.FeeMicros), true).
		Field("New Balance", fmtStonky(out.BalanceMicros), true)

	return b.respondEmbed(s, i, eb.Build())
}

func (b *Bot) handleFunds(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	raw, err := b.client.ListFunds(ctx, token)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}

	funds, ok := raw["funds"]
	if !ok {
		return b.respondEmbed(s, i, infoEmbed("Mutual Funds", "No funds available right now.", nil))
	}
	items, ok := funds.([]any)
	if !ok || len(items) == 0 {
		return b.respondEmbed(s, i, infoEmbed("Mutual Funds", "No funds available right now.", nil))
	}

	lines := make([]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		code := fmt.Sprint(m["code"])
		name := fmt.Sprint(m["name"])
		navStr := ""
		if nav, ok := toInt64(m["nav_micros"]); ok {
			navStr = fmtStonky(nav)
		}
		lines = append(lines, fmt.Sprintf("`%s` **%s** | NAV: %s", code, name, navStr))
	}

	eb := NewEmbed().Title("Mutual Funds").Color(colorInfo).
		Desc(strings.Join(lines, "\n")).
		Field("Total Funds", strconv.Itoa(len(items)), true)

	return b.respondEmbed(s, i, eb.Build())
}

func (b *Bot) handleFundOrder(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	fundCode := strings.TrimSpace(stringOption(data.Options, "fund", ""))
	side := strings.ToLower(strings.TrimSpace(stringOption(data.Options, "side", "")))
	units := integerOption(data.Options, "units", 0)

	var raw map[string]any
	if side == "buy" {
		raw, err = b.client.BuyFund(ctx, token, fundCode, uuid.NewString(), units)
	} else {
		raw, err = b.client.SellFund(ctx, token, fundCode, uuid.NewString(), units)
	}
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}

	fields := fieldsFromMap(raw, []fieldMapping{
		{Key: "fund_code", Label: "Fund"},
		{Key: "units", Label: "Units"},
		{Key: "nav_micros", Label: "NAV", Micros: true},
		{Key: "total_cost_micros", Label: "Total Cost", Micros: true},
		{Key: "balance_micros", Label: "New Balance", Micros: true},
	})

	return b.respondEmbed(s, i, successEmbed("Fund Order Complete", fmt.Sprintf("%s %d units of `%s`.", strings.Title(side), units, fundCode), fields))
}

func (b *Bot) handleLeaderboard(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	scope := strings.TrimSpace(stringOption(i.ApplicationCommandData().Options, "scope", "global"))

	var raw map[string]any
	var err error
	if scope == "friends" {
		token, _, tokenErr := b.requireSession(ctx, s, i)
		if tokenErr != nil {
			return tokenErr
		}
		raw, err = b.client.LeaderboardFriends(ctx, token)
		if err != nil {
			return b.respondAuthAwareError(ctx, s, i, err)
		}
	} else {
		raw, err = b.client.LeaderboardGlobal(ctx, "")
		if err != nil {
			return b.respondError(s, i, trimAPIError(err))
		}
	}

	out, err := decodeInto[leaderboardPayload](raw)
	if err != nil {
		return err
	}
	if len(out.Rows) == 0 {
		return b.respondEmbed(s, i, leaderboardEmbed(strings.Title(scope)+" Leaderboard", "No leaderboard data yet.", nil))
	}

	lines := make([]string, 0, min(len(out.Rows), 15))
	for idx, row := range out.Rows {
		if idx >= 15 {
			break
		}
		medal := ""
		if row.Rank == 1 {
			medal = " [1st]"
		} else if row.Rank == 2 {
			medal = " [2nd]"
		} else if row.Rank == 3 {
			medal = " [3rd]"
		}
		lines = append(lines, fmt.Sprintf("`#%d` **%s**%s | %s", row.Rank, row.Username, medal, fmtStonky(row.NetWorthMicros)))
	}

	return b.respondEmbed(s, i, leaderboardEmbed(strings.Title(scope)+" Leaderboard", strings.Join(lines, "\n"), nil))
}
