package discordbot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func customID(parts ...string) string {
	return strings.Join(parts, ":")
}

func parseCustomID(id string) []string {
	return strings.Split(id, ":")
}

func primaryButton(label, cid string) discordgo.Button {
	return discordgo.Button{
		Label:    label,
		Style:    discordgo.PrimaryButton,
		CustomID: cid,
	}
}

func secondaryButton(label, cid string) discordgo.Button {
	return discordgo.Button{
		Label:    label,
		Style:    discordgo.SecondaryButton,
		CustomID: cid,
	}
}

func successButton(label, cid string) discordgo.Button {
	return discordgo.Button{
		Label:    label,
		Style:    discordgo.SuccessButton,
		CustomID: cid,
	}
}

func dangerButton(label, cid string) discordgo.Button {
	return discordgo.Button{
		Label:    label,
		Style:    discordgo.DangerButton,
		CustomID: cid,
	}
}

func disabledButton(label string) discordgo.Button {
	return discordgo.Button{
		Label:    label,
		Style:    discordgo.SecondaryButton,
		CustomID: "noop:" + label,
		Disabled: true,
	}
}

func actionRow(components ...discordgo.MessageComponent) discordgo.ActionsRow {
	return discordgo.ActionsRow{Components: components}
}

const pageSize = 10

func paginationRow(namespace string, currentPage, totalPages int, extraArgs ...string) discordgo.ActionsRow {
	extra := ""
	if len(extraArgs) > 0 {
		extra = ":" + strings.Join(extraArgs, ":")
	}

	prevID := customID("page", namespace, strconv.Itoa(currentPage-1)) + extra
	nextID := customID("page", namespace, strconv.Itoa(currentPage+1)) + extra
	refreshID := customID("page", namespace, strconv.Itoa(currentPage)) + extra

	prevBtn := secondaryButton("Prev", prevID)
	nextBtn := secondaryButton("Next", nextID)
	pageBtn := disabledButton(fmt.Sprintf("Page %d/%d", currentPage+1, totalPages))
	refreshBtn := secondaryButton("Refresh", refreshID)

	if currentPage <= 0 {
		prevBtn.Disabled = true
	}
	if currentPage >= totalPages-1 {
		nextBtn.Disabled = true
	}

	return actionRow(prevBtn, pageBtn, nextBtn, refreshBtn)
}

func stockActionButtons(symbol string) discordgo.ActionsRow {
	return actionRow(
		successButton("Buy "+symbol, customID("quickbuy", symbol)),
		dangerButton("Sell "+symbol, customID("quicksell", symbol)),
		secondaryButton("Refresh", customID("refresh_stock", symbol)),
	)
}

func businessActionButtons(businessID int64) discordgo.ActionsRow {
	bid := strconv.FormatInt(businessID, 10)
	return actionRow(
		primaryButton("Employees", customID("biz_employees", bid)),
		primaryButton("Machinery", customID("biz_machinery", bid)),
		secondaryButton("Loans", customID("biz_loans", bid)),
		secondaryButton("Refresh", customID("refresh_biz", bid)),
	)
}

func dashboardButtons() discordgo.ActionsRow {
	return actionRow(
		primaryButton("Portfolio", customID("nav", "portfolio")),
		primaryButton("Stocks", customID("nav", "stocks")),
		primaryButton("Funds", customID("nav", "funds")),
		secondaryButton("Refresh", customID("refresh", "dashboard")),
	)
}

func setupButtons() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		actionRow(
			primaryButton("How To Start", customID("setup", "start")),
			primaryButton("Command Guide", customID("setup", "commands")),
			primaryButton("Game Loop", customID("setup", "loop")),
		),
		actionRow(
			secondaryButton("Browse Stocks", customID("nav", "stocks")),
			secondaryButton("Browse Funds", customID("nav", "funds")),
			secondaryButton("Intro", customID("setup", "intro")),
		),
	}
}

func confirmCancelRow(confirmID, cancelID string) discordgo.ActionsRow {
	return actionRow(
		successButton("Confirm", confirmID),
		dangerButton("Cancel", cancelID),
	)
}
