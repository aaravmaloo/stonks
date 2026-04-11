package whatsappbot

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types"
)

func (b *Bot) handleSignup(ctx context.Context, chat, sender types.JID, args []string) error {
	if len(args) < 3 {
		return b.replyText(ctx, chat, "Usage: `!signup <username> <email> <password>`")
	}
	username := args[0]
	email := args[1]
	password := strings.Join(args[2:], " ")

	record, err := b.api.Signup(ctx, email, password, username)
	if err != nil {
		return fmt.Errorf("signup failed: %v", trimAPIError(err))
	}

	err = b.store.SaveSession(ctx, sender.String(), email, record.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to save session locally: %v", err)
	}

	welcome := fmt.Sprintf("Account created!\nWelcome to Stanks, %s 🚀\nRun `!dashboard` to see your stats.", username)
	return b.replyText(ctx, chat, welcome)
}

func (b *Bot) handleLogin(ctx context.Context, chat, sender types.JID, args []string) error {
	if len(args) < 2 {
		return b.replyText(ctx, chat, "Usage: `!login <email> <password>`")
	}
	email := args[0]
	password := strings.Join(args[1:], " ")

	record, err := b.api.Login(ctx, email, password)
	if err != nil {
		return fmt.Errorf("login failed: %v", trimAPIError(err))
	}

	err = b.store.SaveSession(ctx, sender.String(), email, record.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to save session locally: %v", err)
	}

	welcome := fmt.Sprintf("Logged in successfully! Welcome back, %s 🚀\nRun `!dashboard` or `!help` to see what to do.", email)
	return b.replyText(ctx, chat, welcome)
}

func (b *Bot) handleLogout(ctx context.Context, chat, sender types.JID, args []string) error {
	_, _, err := b.requireSession(ctx, chat, sender)
	if err != nil {
		return nil
	}

	err = b.store.DeleteSession(ctx, sender.String())
	if err != nil {
		return fmt.Errorf("failed to clear session: %v", err)
	}

	return b.replyText(ctx, chat, "Logged out successfully.")
}

func (b *Bot) handleSetup(ctx context.Context, chat, sender types.JID, args []string) error {
	msg := `*Welcome to Stanks!* 📈

Stanks is a terminal-style economy simulation game. Here's how you play from WhatsApp:

*1. Account Setup*
- ` + "`!signup <user> <email> <password>`" + `
- ` + "`!login <email> <password>`" + `

*2. Looking Around*
- ` + "`!dashboard`" + ` (your summary)
- ` + "`!world`" + ` (market & global status)
- ` + "`!leaderboard global`" + ` (top players)

*3. Trading*
- ` + "`!stocks`" + ` (market listings)
- ` + "`!stock <symbol>`" + ` (stock details)
- ` + "`!order <buy|sell> <symbol> <qty>`" + `

*4. Business*
- ` + "`!bus-create <name>`" + `
- ` + "`!business <id>`" + `

*5. Action*
- ` + "`!rush <steady|surge|apex> <amount>`" + `

To see all commands or for help, simply experiment or look at the discord reference.`
	return b.replyText(ctx, chat, msg)
}

// Utility identical to discordbot's trimming
func trimAPIError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if !strings.Contains(msg, "api status") {
		return msg
	}
	idx := strings.Index(msg, ":")
	if idx < 0 || idx == len(msg)-1 {
		return msg
	}
	body := strings.TrimSpace(msg[idx+1:])
	return body
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
