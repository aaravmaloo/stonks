package whatsappbot

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"stanks/internal/game"
)

type leaderboardPayload struct {
	Rows []game.LeaderboardRow `json:"rows"`
}

func (b *Bot) handleDashboard(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}

	raw, err := b.api.Dashboard(ctx, token)
	if err != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(err))
	}

	wallet := raw["wallet"].(map[string]any)
	balance := formatMaybeMicros(wallet["balance_micros"])

	username := fmt.Sprint(raw["username"])

	msg := fmt.Sprintf("*Dashboard for %s*\n\n*Wallet Balance*: %s\n\nRun `!order`, `!stocks`, or `!portfolio` for more.", username, balance)
	return b.replyText(ctx, chat, msg)
}

func (b *Bot) handleWorld(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}

	raw, err := b.api.World(ctx, token)
	if err != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(err))
	}
	st, ok := raw["state"].(map[string]any)
	if !ok {
		return b.replyText(ctx, chat, "Could not understand world state")
	}

	msg := fmt.Sprintf(`*🌍 World Status*

*Volatility*: %v
*Catalyst*: %v
*Ticks Served*: %v
*Stocks Listed*: %v
*Total Players*: %v
*Active Players*: %v`,
		st["volatility"], st["catalyst"], st["ticks_served"],
		st["stocks_listed"], st["total_players"], st["active_players"])

	return b.replyText(ctx, chat, msg)
}

func (b *Bot) handleLeaderboard(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}

	scope := "global"
	if len(args) > 0 {
		scope = args[0]
	}

	var raw map[string]any
	if scope == "friends" {
		raw, err = b.api.LeaderboardFriends(ctx, token)
	} else {
		raw, err = b.api.LeaderboardGlobal(ctx, token)
	}

	if err != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(err))
	}

	out, err := decodeInto[leaderboardPayload](raw)
	if err != nil {
		return b.replyText(ctx, chat, "Failed to parse data.")
	}

	if len(out.Rows) == 0 {
		return b.replyText(ctx, chat, "No leaderboard data found.")
	}

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("*%s Leaderboard*\n\n", strings.Title(scope)))
	for i, r := range out.Rows {
		sb.WriteString(fmt.Sprintf("%d. %s - NW: %s\n", i+1, r.Username, fmtStonky(r.NetWorthMicros)))
	}
	return b.replyText(ctx, chat, strings.TrimSpace(sb.String()))
}

func (b *Bot) handleFriends(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 2 {
		return b.replyText(ctx, chat, "Usage: `!friends <add|remove> <invite_code>`")
	}

	action := args[0]
	inviteCode := args[1]

	var errResp error
	if action == "add" {
		_, errResp = b.api.AddFriend(ctx, token, inviteCode, "")
	} else if action == "remove" {
		_, errResp = b.api.RemoveFriend(ctx, token, inviteCode)
	} else {
		return b.replyText(ctx, chat, "Unknown action. Use add or remove.")
	}

	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}
	return b.replyText(ctx, chat, "Friend list updated successfully.")
}

func (b *Bot) handleTransfer(ctx context.Context, chat, sender types.JID, args []string) error {
	token, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}
	if len(args) < 2 {
		return b.replyText(ctx, chat, "Usage: `!transfer <username> <amount_micros>`")
	}
	username := args[0]
	var amount int64
	fmt.Sscanf(args[1], "%d", &amount)

	_, errResp := b.api.TransferStonky(ctx, token, username, "", amount)
	if errResp != nil {
		return b.replyText(ctx, chat, "Error: "+trimAPIError(errResp))
	}

	return b.replyText(ctx, chat, fmt.Sprintf("Successfully transferred money to %s.", username))
}
