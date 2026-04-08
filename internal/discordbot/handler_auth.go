package discordbot

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) openAuthModal(s *discordgo.Session, i *discordgo.InteractionCreate, customID, title string, includeUsername bool) error {
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "email", Label: "Email", Style: discordgo.TextInputShort, Placeholder: "you@example.com", Required: true},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "password", Label: "Password", Style: discordgo.TextInputShort, Placeholder: "Strong password", Required: true},
		}},
	}
	if includeUsername {
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "username", Label: "Username", Style: discordgo.TextInputShort, Placeholder: "stonkslord", Required: true},
		}})
	}
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   customID,
			Title:      title,
			Components: components,
		},
	})
}

func (b *Bot) handleSignupModal(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, values map[string]string) error {
	email := strings.TrimSpace(values["email"])
	password := strings.TrimSpace(values["password"])
	username := strings.TrimSpace(values["username"])

	session, err := b.client.Signup(ctx, email, password, username)
	if err != nil {
		return b.respondError(s, i, trimAPIError(err))
	}
	userID, err := interactionUserID(i)
	if err != nil {
		return err
	}
	if err := b.store.SaveSession(ctx, userID, session.User.Email, session.AccessToken); err != nil {
		return err
	}
	return b.respondEmbed(s, i, successEmbed("Account Ready", "Your Stanks account is live and linked to this Discord user.", []*discordgo.MessageEmbedField{
		{Name: "Email", Value: session.User.Email, Inline: true},
		{Name: "Username", Value: username, Inline: true},
	}))
}

func (b *Bot) handleLoginModal(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, values map[string]string) error {
	email := strings.TrimSpace(values["email"])
	password := strings.TrimSpace(values["password"])

	session, err := b.client.Login(ctx, email, password)
	if err != nil {
		return b.respondError(s, i, trimAPIError(err))
	}
	userID, err := interactionUserID(i)
	if err != nil {
		return err
	}
	if err := b.store.SaveSession(ctx, userID, session.User.Email, session.AccessToken); err != nil {
		return err
	}
	return b.respondEmbed(s, i, successEmbed("Logged In", "Your Discord account is now connected to Stanks.", []*discordgo.MessageEmbedField{
		{Name: "Email", Value: session.User.Email, Inline: true},
	}))
}

func (b *Bot) handleLogout(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	userID, err := interactionUserID(i)
	if err != nil {
		return err
	}
	if err := b.store.DeleteSession(ctx, userID); err != nil {
		return err
	}
	return b.respondEmbed(s, i, infoEmbed("Logged Out", "This Discord account is no longer linked to a Stanks session.", nil))
}
