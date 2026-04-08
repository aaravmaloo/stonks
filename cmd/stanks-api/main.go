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
	"stanks/internal/admin"
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
	if err := db.EnsureTables(ctx, pool); err != nil {
		logger.Error("db bootstrap failed", "err", err)
		os.Exit(1)
	}

	authClient := auth.NewClient(pool)
	gameSvc := game.NewService(pool, logger)
	adminSvc := admin.NewService(pool)

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
	if err := gameSvc.ClampNegativeBalances(ctx, seasonID); err != nil {
		logger.Error("balance clamp failed", "err", err)
		os.Exit(1)
	}

	server := api.New(cfg, logger, authClient, gameSvc, adminSvc)
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
