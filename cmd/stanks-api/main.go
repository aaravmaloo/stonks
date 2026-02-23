package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stanks/internal/api"
	"stanks/internal/auth"
	"stanks/internal/config"
	"stanks/internal/db"
	"stanks/internal/game"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadAPIFromEnv()
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

	authClient := auth.NewSupabaseClient(cfg.SupabaseURL, cfg.SupabaseAnonKey)
	gameSvc := game.NewService(pool, logger)

	seasonID, err := gameSvc.ActiveSeasonID(ctx)
	if err != nil {
		logger.Error("active season init failed", "err", err)
		os.Exit(1)
	}
	if cfg.StartupSeedStocks {
		if err := gameSvc.SeedDefaults(ctx, seasonID); err != nil {
			logger.Error("seed defaults failed", "err", err)
			os.Exit(1)
		}
	}

	server := api.New(cfg, logger, authClient, gameSvc)
	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	logger.Info("stanks api listening", "addr", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server failed", "err", err)
		os.Exit(1)
	}
}
