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

	startingPL := out.NetWorthMicros - game.StarterBalanceMicros
	openPL := int64(0)
	for _, pos := range out.Positions {
		openPL += pos.UnrealizedMicros
	}
	downFromPeak := out.NetWorthMicros - out.PeakNetWorthMicros

	sections := []string{
		codeBlock(
			fmt.Sprintf("Balance:            %s stonky", fmtMicrosExact(out.BalanceMicros)),
			fmt.Sprintf("Net Worth:          %s stonky", fmtMicrosExact(out.NetWorthMicros)),
			fmt.Sprintf("Peak Net Worth:     %s stonky", fmtMicrosExact(out.PeakNetWorthMicros)),
			fmt.Sprintf("P/L vs Start:       %s stonky", signedMicrosExact(startingPL)),
			fmt.Sprintf("Open Position P/L:  %s stonky", signedMicrosExact(openPL)),
			fmt.Sprintf("From Peak:          %s stonky", signedMicrosExact(downFromPeak)),
			fmt.Sprintf("Reputation:         %s (%d/10000)", out.Progression.ReputationTitle, out.Progression.ReputationScore),
			fmt.Sprintf("Profit Streak:      %d (best %d)", out.Progression.CurrentProfitStreak, out.Progression.BestProfitStreak),
			fmt.Sprintf("Risk Appetite:      %.2f%%", float64(out.Progression.RiskAppetiteBps)/100),
			fmt.Sprintf("Catalyst:           %s (%d ticks)", out.World.CatalystName, out.World.CatalystTicksRemaining),
		),
	}

	if len(out.Positions) == 0 {
		sections = append(sections, "**Positions**\nNo open positions yet.")
	} else {
		lines := []string{
			fmt.Sprintf("%-8s %-12s %-12s %-12s", "SYMBOL", "QTY", "NOW", "P/L"),
		}
		for _, pos := range out.Positions {
			lines = append(lines, fmt.Sprintf("%-8s %-12.4f %-12s %-12s",
				strings.TrimSpace(pos.Symbol),
				game.UnitsToShares(pos.QuantityUnits),
				fmtMicrosExact(pos.CurrentPriceMicros),
				signedMicrosExact(pos.UnrealizedMicros),
			))
		}
		sections = append(sections, "**Positions**\n"+codeBlock(lines...))
	}

	if len(out.Businesses) == 0 {
		sections = append(sections, "**Businesses**\nNo businesses yet.")
	} else {
		lines := []string{
			fmt.Sprintf("%-4s %-18s %-10s %-10s %-10s", "ID", "NAME", "ARC", "REV/TICK", "RESERVE"),
		}
		for _, biz := range out.Businesses {
			lines = append(lines, fmt.Sprintf("%-4d %-18s %-10s %-10s %-10s",
				biz.ID,
				truncateText(biz.Name, 18),
				truncateText(biz.NarrativeArc, 10),
				fmtMicrosExact(biz.RevenuePerTickMicros),
				fmtMicrosExact(biz.CashReserveMicros),
			))
		}
		sections = append(sections, "**Businesses**\n"+codeBlock(lines...))
	}

	eb := NewEmbed().Title(fmt.Sprintf("Dashboard | Season %d", out.SeasonID)).Color(colorInfo).Desc(strings.Join(sections, "\n\n"))
	return b.respondEmbedWithComponents(s, i, eb.Build(), []discordgo.MessageComponent{dashboardButtons()})
}

func (b *Bot) handleWorld(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	raw, err := b.client.World(ctx, token)
	if err != nil {
		return b.respondAuthAwareError(ctx, s, i, err)
	}
	out, err := decodeInto[game.WorldView](raw)
	if err != nil {
		return err
	}

	lines := []string{
		fmt.Sprintf("Regime:        %s", out.Regime),
		fmt.Sprintf("Politics:      %s", out.PoliticalClimate),
		fmt.Sprintf("Policy Focus:  %s", out.PolicyFocus),
		fmt.Sprintf("Catalyst:      %s (%d ticks)", out.CatalystName, out.CatalystTicksRemaining),
		fmt.Sprintf("Risk Bias:     %.2f%%", float64(out.RiskRewardBiasBps)/100),
	}
	for _, region := range out.Regions {
		lines = append(lines, fmt.Sprintf("%-12s %8.2f%%", region.Name, float64(region.TrendBps)/100))
	}

	desc := codeBlock(lines...) + "\n\n" + out.CatalystSummary + "\n\n" + out.Headline
	if len(out.RecentEvents) > 0 {
		eventLines := make([]string, 0, len(out.RecentEvents)+1)
		eventLines = append(eventLines, "CATEGORY      HEADLINE")
		for _, event := range out.RecentEvents {
			eventLines = append(eventLines, fmt.Sprintf("%-12s %s", truncateText(strings.ToUpper(event.Category), 12), truncateText(event.Headline, 48)))
		}
		desc += "\n\nRecent Events\n" + codeBlock(eventLines...)
	}

	eb := NewEmbed().Title("World State").Color(colorMarket).Desc(desc)
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
	lines := []string{
		fmt.Sprintf("Email:              %s", email),
		fmt.Sprintf("Balance:            %s stonky", fmtMicrosExact(out.BalanceMicros)),
		fmt.Sprintf("Peak Net Worth:     %s stonky", fmtMicrosExact(out.PeakNetWorthMicros)),
	}
	if out.ActiveBusinessID != nil {
		lines = append(lines, fmt.Sprintf("Active Business:    %d", *out.ActiveBusinessID))
	}
	eb := NewEmbed().Title("Wallet").Color(colorInfo).Desc(codeBlock(lines...))
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

	lines := []string{
		fmt.Sprintf("%-8s %-22s %10s %12s %12s", "SYMBOL", "NAME", "QTY", "NOW", "P/L"),
	}
	var totalPL int64
	for _, pos := range out.Positions {
		totalPL += pos.UnrealizedMicros
		lines = append(lines, fmt.Sprintf("%-8s %-22s %10.4f %12s %12s",
			strings.TrimSpace(pos.Symbol),
			truncateText(pos.DisplayName, 22),
			game.UnitsToShares(pos.QuantityUnits),
			fmtMicrosExact(pos.CurrentPriceMicros),
			signedMicrosExact(pos.UnrealizedMicros),
		))
	}

	color := colorInfo
	if totalPL > 0 {
		color = colorSuccess
	} else if totalPL < 0 {
		color = colorError
	}

	eb := NewEmbed().Title("Portfolio").Color(color).
		Desc(codeBlock(lines...)).
		Field("Total P/L", signedMicrosExact(totalPL)+" stonky", true).
		Field("Positions", strconv.Itoa(len(out.Positions)), true).
		Field("Balance", fmtMicrosExact(out.BalanceMicros)+" stonky", true)

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

	lines := []string{
		fmt.Sprintf("%-8s %-24s %12s %-8s", "SYMBOL", "NAME", "PRICE", "LISTED"),
	}
	for _, stock := range stocks[start:end] {
		status := "no"
		if stock.ListedPublic {
			status = "yes"
		}
		lines = append(lines, fmt.Sprintf("%-8s %-24s %12s %-8s",
			strings.TrimSpace(stock.Symbol),
			truncateText(stock.DisplayName, 24),
			fmtMicrosExact(stock.CurrentPriceMicros),
			status,
		))
	}

	eb := NewEmbed().Title("Stock Market").Color(colorMarket).
		Desc(codeBlock(lines...)).
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

	header := []string{
		fmt.Sprintf("Current Price: %s stonky", fmtMicrosExact(out.CurrentPriceMicros)),
		fmt.Sprintf("Listed Public: %t", out.ListedPublic),
	}

	if len(out.Series) > 0 {
		prices := make([]int64, len(out.Series))
		for idx, p := range out.Series {
			prices[idx] = p.PriceMicros
		}
		header = append(header, "Trend (recent): "+sparkline(prices))

		recent := out.Series
		if len(recent) > 5 {
			recent = recent[len(recent)-5:]
		}
		points := []string{fmt.Sprintf("%-20s %12s", "TIME", "PRICE")}
		for _, point := range recent {
			points = append(points, fmt.Sprintf("%-20s %12s", point.TickAt.Local().Format("2006-01-02 15:04"), fmtMicrosExact(point.PriceMicros)))
		}
		header = append(header, "", "Recent Ticks", codeBlock(points...))
	}

	eb := NewEmbed().Title("Stock | " + strings.TrimSpace(out.Symbol)).Color(colorMarket).
		Desc("**" + out.DisplayName + "**\n\n" + strings.Join(header, "\n"))

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
		Field("Price", fmtStonky(out.PriceMicros), true)

	for _, field := range spendSummaryFields(out.NotionalMicros, out.FeeMicros, out.BalanceMicros) {
		eb.Field(field.Name, field.Value, field.Inline)
	}

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

	lines := []string{
		fmt.Sprintf("%-8s %12s %-40s", "CODE", "NAV", "COMPONENTS"),
	}
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
		lines = append(lines, fmt.Sprintf("%-8s %12s %-40s", code, navStr, truncateText(name, 40)))
	}

	eb := NewEmbed().Title("Mutual Funds").Color(colorInfo).
		Desc(codeBlock(lines...)).
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

	fields := []*discordgo.MessageEmbedField{
		{Name: "Fund", Value: fundCode, Inline: true},
		{Name: "Units", Value: strconv.FormatInt(units, 10), Inline: true},
	}
	if nav, ok := int64FromMapKeys(raw, "nav_micros"); ok {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "NAV", Value: fmtStonky(nav), Inline: true})
	}
	if notional, ok := int64FromMapKeys(raw, "notional_micros"); ok {
		fee, _ := int64FromMapKeys(raw, "fee_micros")
		balance, _ := int64FromMapKeys(raw, "balance_micros")
		fields = append(fields, spendSummaryFields(notional, fee, balance)...)
	}

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

	lines := []string{
		fmt.Sprintf("%-6s %-18s %-12s %14s", "RANK", "PLAYER", "INVITE", "NET WORTH"),
	}
	for idx, row := range out.Rows {
		if idx >= 15 {
			break
		}
		lines = append(lines, fmt.Sprintf("%-6d %-18s %-12s %14s",
			row.Rank,
			truncateText(row.Username, 18),
			truncateText(row.InviteCode, 12),
			fmtMicrosExact(row.NetWorthMicros),
		))
	}

	return b.respondEmbed(s, i, leaderboardEmbed(strings.Title(scope)+" Leaderboard", codeBlock(lines...), nil))
}
