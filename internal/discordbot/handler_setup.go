package discordbot

import (
	"context"
	"errors"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleSetup(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return b.respondEmbedWithComponents(s, i, b.setupEmbed(ctx, i, "intro"), setupButtons())
}

func (b *Bot) handleSetupButton(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) error {
	page := "intro"
	if len(parts) >= 2 {
		page = strings.TrimSpace(parts[1])
	}
	return b.respondEmbedWithComponents(s, i, b.setupEmbed(ctx, i, page), setupButtons())
}

func (b *Bot) setupEmbed(ctx context.Context, i *discordgo.InteractionCreate, page string) *discordgo.MessageEmbed {
	switch page {
	case "start":
		return b.setupStartEmbed(ctx, i)
	case "commands":
		return b.setupCommandsEmbed()
	case "loop":
		return b.setupLoopEmbed()
	default:
		return b.setupIntroEmbed(ctx, i)
	}
}

func (b *Bot) setupIntroEmbed(ctx context.Context, i *discordgo.InteractionCreate) *discordgo.MessageEmbed {
	status := "Run `/signup` to create an account, or `/login` if you already have one."
	if userID, err := interactionUserID(i); err == nil {
		if record, getErr := b.store.GetSession(ctx, userID); getErr == nil {
			status = "You are already linked as `" + record.Email + "`. Use `/dashboard` to jump back in."
		} else if getErr != nil && !errors.Is(getErr, ErrNoSession) {
			status = "Account status is unavailable right now, but you can still use `/signup` or `/login`."
		}
	}

	desc := strings.Join([]string{
		"Stanks is a stock-market sandbox where you build net worth by trading, running businesses, and scaling into funds.",
		"",
		status,
	}, "\n")

	return NewEmbed().Title("Welcome To Stanks").Color(colorGold).Desc(desc).
		Field("Start Cash", "25,000 stonky plus a 2,000 stonky signup bonus", false).
		Field("Core Loop", "Trade stocks, grow businesses, hire staff, and climb the leaderboard", false).
		Field("Best First Command", "`/setup` now, then `/stocks`, then `/signup` when ready", false).
		Build()
}

func (b *Bot) setupStartEmbed(ctx context.Context, i *discordgo.InteractionCreate) *discordgo.MessageEmbed {
	ready := "No account linked yet."
	if userID, err := interactionUserID(i); err == nil {
		if _, getErr := b.store.GetSession(ctx, userID); getErr == nil {
			ready = "Account linked. You can skip straight to `/dashboard`."
		}
	}

	desc := strings.Join([]string{
		"Use these commands in order to get moving fast:",
		"`/signup` -> create your Stanks account",
		"`/login` -> reconnect an existing account",
		"`/dashboard` -> see wallet, positions, and businesses",
	}, "\n")

	return NewEmbed().Title("How To Start").Color(colorInfo).Desc(desc).
		Field("Account", ready, true).
		Field("Browse First", "`/stocks` and `/funds` work even before you place trades", true).
		Field("Need A Session?", "If `/dashboard` says your session expired, run `/login` again.", false).
		Build()
}

func (b *Bot) setupCommandsEmbed() *discordgo.MessageEmbed {
	desc := strings.Join([]string{
		"`/stocks` list the market",
		"`/stock symbol:COBOLT` inspect one ticker",
		"`/order symbol:COBOLT side:buy shares:5` place a trade",
		"`/business-create name:\"Acme Labs\" visibility:public` open a company",
		"`/candidates`, `/employees`, `/hire-many` grow staff",
		"`/leaderboard` see who is winning",
	}, "\n")

	return NewEmbed().Title("Command Guide").Color(colorMarket).Desc(desc).
		Field("Trading", "Use `/stocks`, `/stock`, `/order`, `/portfolio`", false).
		Field("Business", "Use `/business-create`, `/business`, `/machinery`, `/loans`, `/ipo`", false).
		Field("Social", "Use `/friends` and `/leaderboard`", false).
		Build()
}

func (b *Bot) setupLoopEmbed() *discordgo.MessageEmbed {
	desc := strings.Join([]string{
		"1. Scan the market and buy a few positions.",
		"2. Grow net worth until you can open a business.",
		"3. Hire employees, buy machinery, manage loans, and push revenue per tick higher.",
		"4. IPO or sell businesses, rotate into funds, and compound faster than everyone else.",
	}, "\n")

	return NewEmbed().Title("Game Loop").Color(colorBusiness).Desc(desc).
		Field("Win Condition", "There is no hard ending. The goal is higher net worth and better businesses.", false).
		Field("Risk", "Prices move every market tick. Loans and weak businesses can drag you down fast.", false).
		Field("Good Habit", "Check `/dashboard` and `/leaderboard` often so you can react before a bad run compounds.", false).
		Build()
}
