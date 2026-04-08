package discordbot

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (b *Bot) handleFriends(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	token, _, err := b.requireSession(ctx, s, i)
	if err != nil {
		return err
	}
	data := i.ApplicationCommandData()
	action := strings.TrimSpace(stringOption(data.Options, "action", ""))
	inviteCode := strings.TrimSpace(stringOption(data.Options, "invite_code", ""))

	switch action {
	case "add":
		_, err = b.client.AddFriend(ctx, token, inviteCode, uuid.NewString())
		if err != nil {
			return b.respondAuthAwareError(ctx, s, i, err)
		}
		return b.respondEmbed(s, i, successEmbed("Friend Added", fmt.Sprintf("Added friend with invite code `%s`.", inviteCode), nil))

	case "remove":
		_, err = b.client.RemoveFriend(ctx, token, inviteCode)
		if err != nil {
			return b.respondAuthAwareError(ctx, s, i, err)
		}
		return b.respondEmbed(s, i, successEmbed("Friend Removed", fmt.Sprintf("Removed friend with invite code `%s`.", inviteCode), nil))

	default:
		return b.respondError(s, i, "Unknown friends action.")
	}
}
