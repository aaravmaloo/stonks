package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"stanks/internal/config"
	whatsappbot "stanks/internal/whatsapp_bot"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.LoadWhatsAppBotFromEnv()
	if err != nil {
		logger.Error("failed to load configs", "err", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	store := whatsappbot.NewStore(pool)

	bot, err := whatsappbot.New(cfg, logger, store)
	if err != nil {
		logger.Error("failed to create whatsapp bot", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := bot.Run(ctx); err != nil {
		logger.Error("whatsapp bot failed", "err", err)
		os.Exit(1)
	}
}
