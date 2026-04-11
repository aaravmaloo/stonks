package whatsappbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"stanks/internal/cli"
	"stanks/internal/config"

	_ "github.com/lib/pq"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	"github.com/mdp/qrterminal/v3"
)

type Bot struct {
	log    *slog.Logger
	client *whatsmeow.Client
	api    *cli.Client
	store  *Store
}

func New(cfg config.WhatsAppBotConfig, logger *slog.Logger, store *Store) (*Bot, error) {
	if logger == nil {
		logger = slog.Default()
	}

	dbLog := waLog.Stdout("WhatsmeowDB", "ERROR", true)
	container, err := sqlstore.New(context.Background(), "postgres", cfg.DatabaseURL, dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to whatsapp store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	clientLog := waLog.Stdout("WhatsmeowClient", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	bot := &Bot{
		log:    logger,
		client: client,
		api:    cli.NewClient(cfg.APIBaseURL),
		store:  store,
	}

	client.AddEventHandler(bot.eventHandler)
	return bot, nil
}

func (b *Bot) Run(ctx context.Context) error {
	if b.client.Store.ID == nil {
		qrChan, _ := b.client.GetQRChannel(context.Background())
		err := b.client.Connect()
		if err != nil {
			return err
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				b.log.Info("Scan the QR code above to authenticate WhatsApp bot")
			} else {
				b.log.Info("QR channel event", "event", evt.Event)
			}
		}
	} else {
		err := b.client.Connect()
		if err != nil {
			return err
		}
	}

	b.log.Info("WhatsApp bot actively listening")
	<-ctx.Done()
	b.client.Disconnect()
	return nil
}

func (b *Bot) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		var text string
		if v.Message.GetConversation() != "" {
			text = v.Message.GetConversation()
		} else if v.Message.ExtendedTextMessage != nil {
			text = v.Message.ExtendedTextMessage.GetText()
		}

		text = strings.TrimSpace(text)
		if strings.HasPrefix(text, "!") {
			b.handleCommand(v.Info.Chat, v.Info.Sender, text)
		}
	}
}

func (b *Bot) handleCommand(chat, sender types.JID, text string) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return
	}
	command := strings.ToLower(strings.TrimPrefix(parts[0], "!"))
	args := parts[1:]

	ctx := context.Background()

	var err error
	switch command {
	case "signup":
		err = b.handleSignup(ctx, chat, sender, args)
	case "login":
		err = b.handleLogin(ctx, chat, sender, args)
	case "logout":
		err = b.handleLogout(ctx, chat, sender, args)
	case "dashboard":
		err = b.handleDashboard(ctx, chat, sender, args)
	case "world":
		err = b.handleWorld(ctx, chat, sender, args)
	case "wallet":
		err = b.handleWallet(ctx, chat, sender, args)
	case "friends":
		err = b.handleFriends(ctx, chat, sender, args)
	case "leaderboard":
		err = b.handleLeaderboard(ctx, chat, sender, args)
	case "stocks":
		err = b.handleStocks(ctx, chat, sender, args)
	case "stock":
		err = b.handleStock(ctx, chat, sender, args)
	case "order":
		err = b.handleOrder(ctx, chat, sender, args)
	case "portfolio":
		err = b.handlePortfolio(ctx, chat, sender, args)
	case "transfer":
		err = b.handleTransfer(ctx, chat, sender, args)
	case "funds":
		err = b.handleFunds(ctx, chat, sender, args)
	case "fund-order":
		err = b.handleFundOrder(ctx, chat, sender, args)
	case "business-create":
		err = b.handleBusinessCreate(ctx, chat, sender, args)
	case "business":
		err = b.handleBusiness(ctx, chat, sender, args)
	case "employees":
		err = b.handleEmployees(ctx, chat, sender, args)
	case "candidates":
		err = b.handleCandidates(ctx, chat, sender, args)
	case "hire-many":
		err = b.handleHireMany(ctx, chat, sender, args)
	case "machinery":
		err = b.handleMachinery(ctx, chat, sender, args)
	case "loans":
		err = b.handleLoans(ctx, chat, sender, args)
	case "strategy":
		err = b.handleStrategy(ctx, chat, sender, args)
	case "upgrades":
		err = b.handleUpgrades(ctx, chat, sender, args)
	case "reserve":
		err = b.handleReserve(ctx, chat, sender, args)
	case "ipo":
		err = b.handleIPO(ctx, chat, sender, args)
	case "sell-business":
		err = b.handleSellBusiness(ctx, chat, sender, args)
	case "stakes":
		err = b.handleStakes(ctx, chat, sender, args)
	case "give-stake":
		err = b.handleGiveStake(ctx, chat, sender, args)
	case "revoke-stakes":
		err = b.handleRevokeStakes(ctx, chat, sender, args)
	case "rush":
		err = b.handleRush(ctx, chat, sender, args)
	case "setup", "help":
		err = b.handleSetup(ctx, chat, sender, args)
	default:
		err = b.replyText(ctx, chat, "Unknown command. Send `!help` to get started.")
	}

	if err != nil {
		b.log.Error("whatsapp command failed", "command", command, "err", err)
		b.replyText(ctx, chat, fmt.Sprintf("Error: %s", err.Error()))
	}
}

func (b *Bot) replyText(ctx context.Context, to types.JID, text string) error {
	_, err := b.client.SendMessage(ctx, to, &waE2E.Message{
		Conversation: proto.String(text),
	})
	return err
}

func (b *Bot) requireSession(ctx context.Context, chat types.JID, sender types.JID) (string, string, error) {
	record, err := b.store.GetSession(ctx, sender.String())
	if err == nil {
		return record.AccessToken, record.Email, nil
	}
	if errors.Is(err, ErrNoSession) {
		b.replyText(ctx, chat, "You need to `!signup <username> <email> <password>` or `!login <email> <password>` first.")
		return "", "", ErrNoSession
	}
	return "", "", err
}
