package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"stanks/internal/config"
	"stanks/internal/db"
	"stanks/internal/discordbot"
)

func main() {
	_ = config.LoadDotEnvIfPresent(".env")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadDiscordBotFromEnv()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.EnsureTables(ctx, pool); err != nil {
		logger.Error("db bootstrap failed", "err", err)
		os.Exit(1)
	}

	bot, err := discordbot.New(cfg, logger, discordbot.NewStore(pool))
	if err != nil {
		logger.Error("bot init failed", "err", err)
		os.Exit(1)
	}

	if err := bot.Run(ctx); err != nil {
		logger.Error("bot failed", "err", err)
		os.Exit(1)
	}
}
