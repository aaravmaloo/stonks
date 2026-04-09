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
		"Stanks is a stock-market sandbox where you build net worth by trading, running businesses, and playing the world state.",
		"",
		status,
	}, "\n")

	return NewEmbed().Title("Welcome To Stanks").Color(colorGold).Desc(desc).
		Field("Start Cash", "25,000 stonky plus a 2,000 stonky signup bonus", false).
		Field("Core Loop", "Trade stocks, build narrative-driven companies, manage reputation, and react to politics/global markets", false).
		Field("Best First Command", "`/setup` now, then `/world`, `/stocks`, and `/signup` when ready", false).
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
		"`/world` -> read the catalyst, political pressure, and global market drift",
		"`/dashboard` -> see wallet, positions, businesses, reputation, and streaks",
		"`/stakes` -> track your business ownership and passive P/L",
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
		"`/world` read the political layer and mid-term catalyst",
		"`/business-create name:\"Acme Labs\" visibility:public` open a company",
		"`/candidates`, `/employees`, `/hire-many` grow staff",
		"`/stakes` inspect your company ownership and mark-to-market P/L",
		"`/give-stake business_id:1 username:friend percent:10` transfer ownership",
		"`/leaderboard` see who is winning",
	}, "\n")

	return NewEmbed().Title("Command Guide").Color(colorMarket).Desc(desc).
		Field("Trading", "Use `/world`, `/stocks`, `/stock`, `/order`, `/portfolio`", false).
		Field("Business", "Use `/business-create`, `/business`, `/machinery`, `/loans`, `/ipo`", false).
		Field("Progression", "Reputation, risk appetite, streak rewards, and passive business ownership live in `/dashboard` and `/stakes`", false).
		Build()
}

func (b *Bot) setupLoopEmbed() *discordgo.MessageEmbed {
	desc := strings.Join([]string{
		"1. Check `/world` so you know the catalyst, political tone, and strongest region.",
		"2. Take positions that fit the current risk/reward window.",
		"3. Open businesses and steer their narrative arc, region exposure, and operating pressure.",
		"4. Stack profitable ticks to build streak rewards and reputation before the world flips again.",
	}, "\n")

	return NewEmbed().Title("Game Loop").Color(colorBusiness).Desc(desc).
		Field("Win Condition", "There is no hard ending. The goal is to survive shifts, compound streaks, and own stronger companies than everyone else.", false).
		Field("Risk", "Aggressive play pays more when the world is hot and gets punished harder when politics/global markets turn.", false).
		Field("Good Habit", "Check `/world` and `/dashboard` often so your plan matches the next few ticks, not the last few.", false).
		Build()
}
