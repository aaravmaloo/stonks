package whatsappbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"stanks/internal/game"
)

type stocksPayload struct {
	Stocks []game.StockView `json:"stocks"`
}

func (b *Bot) handleWallet(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	raw, err := b.api.WalletSummary(ctx, token)
	if err != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(err))
	}
	w, ok := raw["wallet"].(map[string]any)
	if !ok {
		return b.replyText(ctx, chat, "Failed to parse wallet info.")
	}

	msg := fmt.Sprintf("*Wallet Info*\nBalance: %s\nReserved: %s",
		formatMaybeMicros(w["balance_micros"]),
		formatMaybeMicros(w["reserved_micros"]))
	return b.replyText(ctx, chat, msg)
}

func (b *Bot) handleStocks(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	all := false
	page := 1
	if len(args) > 0 {
		if args[0] == "all" {
			all = true
			if len(args) > 1 {
				page, _ = strconv.Atoi(args[1])
			}
		} else {
			page, _ = strconv.Atoi(args[0])
		}
	}
	if page < 1 {
		page = 1
	}

	raw, err := b.api.ListStocks(ctx, token, all)
	if err != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(err))
	}

	out, err := decodeInto[stocksPayload](raw)
	if err != nil {
		return b.replyText(ctx, chat, "Failed to parse data.")
	}

	total := len(out.Stocks)
	pageSize := 10
	start := (page - 1) * pageSize
	if start >= total {
		return b.replyText(ctx, chat, "No stocks on this page.")
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("*Stocks (Page %d)*\n\n", page))
	for _, st := range out.Stocks[start:end] {
		sb.WriteString(fmt.Sprintf("*- %s* (%s): %s\n", st.Symbol, st.DisplayName, fmtStonky(st.CurrentPriceMicros)))
	}
	sb.WriteString(fmt.Sprintf("\nType `!stocks %d` for next page.", page+1))

	return b.replyText(ctx, chat, strings.TrimSpace(sb.String()))
}

func (b *Bot) handleStock(ctx context.Context, chat, sender types.JID, args []string) error {
	if len(args) < 1 {
		return b.replyText(ctx, chat, "Usage: `!stock <symbol>`")
	}
	symbol := strings.ToUpper(args[0])

	raw, err := b.api.StockDetail(ctx, "", symbol)
	if err != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(err))
	}
	out, err := decodeInto[game.StockDetail](raw)
	if err != nil {
		return err
	}

	desc := fmt.Sprintf("*%s* (%s)\nCurrent price: %s", out.Symbol, out.DisplayName, fmtStonky(out.CurrentPriceMicros))
	if len(out.Series) > 0 {
		prices := make([]int64, len(out.Series))
		for idx, p := range out.Series {
			prices[idx] = p.PriceMicros
		}
		desc += "\n\nTrend: `" + sparkline(prices) + "`\n"
	}

	desc += fmt.Sprintf("\nListed: %t", out.ListedPublic)
	return b.replyText(ctx, chat, desc)
}

func (b *Bot) handleOrder(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 3 {
		return b.replyText(ctx, chat, "Usage: `!order <buy/sell> <symbol> <qty>`")
	}
	side := strings.ToLower(args[0])
	symbol := strings.ToUpper(args[1])
	qty, _ := strconv.ParseInt(args[2], 10, 64)

	_, errResp := b.api.PlaceOrder(ctx, token, symbol, side, "", qty)
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}

	return b.replyText(ctx, chat, fmt.Sprintf("Successfully placed %s order for %d shares of %s.", side, qty, symbol))
}

func (b *Bot) handlePortfolio(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}

	raw, err := b.api.Dashboard(ctx, token)
	if err != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(err))
	}

	positions, ok := raw["positions"].([]any)
	if !ok || len(positions) == 0 {
		return b.replyText(ctx, chat, "Your portfolio is empty.")
	}

	sb := strings.Builder{}
	sb.WriteString("*Your Portfolio*\n\n")
	for _, pos := range positions {
		p := pos.(map[string]any)
		symbol := fmt.Sprint(p["symbol"])
		shares := int64(p["quantity_units"].(float64))
		val := formatMaybeMicros(p["current_value_micros"])
		sb.WriteString(fmt.Sprintf("%s: %d shares (Val: %s)\n", symbol, shares, val))
	}

	return b.replyText(ctx, chat, strings.TrimSpace(sb.String()))
}

func (b *Bot) handleFunds(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	raw, err := b.api.ListFunds(ctx, token)
	if err != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(err))
	}
	funds, ok := raw["funds"].([]any)
	if !ok || len(funds) == 0 {
		return b.replyText(ctx, chat, "No funds available.")
	}

	sb := strings.Builder{}
	sb.WriteString("*Mutual Funds*\n\n")
	for _, f := range funds {
		m := f.(map[string]any)
		sb.WriteString(fmt.Sprintf("*- %s* (%s): NAV %s\n", m["code"], m["name"], formatMaybeMicros(m["nav_micros"])))
	}
	return b.replyText(ctx, chat, strings.TrimSpace(sb.String()))
}

func (b *Bot) handleFundOrder(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 3 {
		return b.replyText(ctx, chat, "Usage: `!fund-order <buy/sell> <code> <units>`")
	}
	side := strings.ToLower(args[0])
	code := strings.ToUpper(args[1])
	units, _ := strconv.ParseInt(args[2], 10, 64)

	var errResp error
	if side == "buy" {
		_, errResp = b.api.BuyFund(ctx, token, code, "", units)
	} else if side == "sell" {
		_, errResp = b.api.SellFund(ctx, token, code, "", units)
	} else {
		return b.replyText(ctx, chat, "Side must be buy or sell.")
	}

	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}
	return b.replyText(ctx, chat, fmt.Sprintf("Successfully processed %s order for %d units of fund %s.", side, units, code))
}

func (b *Bot) handleRush(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 2 {
		return b.replyText(ctx, chat, "Usage: `!rush <mode> <amount_micros>`, modes: steady, surge, apex")
	}
	mode := strings.ToLower(args[0])
	amount, _ := strconv.ParseInt(args[1], 10, 64)

	raw, errResp := b.api.PlayRush(ctx, token, mode, "", amount)
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}
	win := raw["win"].(bool)
	payout := formatMaybeMicros(raw["payout_micros"])

	if win {
		return b.replyText(ctx, chat, fmt.Sprintf("🎉 You won! Payout: %s", payout))
	}
	return b.replyText(ctx, chat, "💀 You lost the rush.")
}
