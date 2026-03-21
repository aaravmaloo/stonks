package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

	svc := game.NewService(pool, logger)
	seasonID, err := svc.ActiveSeasonID(ctx)
	if err != nil {
		logger.Error("active season init failed", "err", err)
		os.Exit(1)
	}
	if cfg.StartupSeedStocks {
		if err := svc.SeedDefaults(ctx, seasonID); err != nil {
			logger.Error("seed defaults failed", "err", err)
			os.Exit(1)
		}
	}

	runOnce := strings.EqualFold(strings.TrimSpace(os.Getenv("STANKS_WORKER_RUN_ONCE")), "true")
	if runOnce {
		stocksThisTick := cfg.NewStocksPerTick
		if cfg.NewStocksEvery > 0 {
			stocksThisTick = 0
		}
		if err := svc.RunMarketTick(ctx, seasonID, cfg.MarketTickEvery, cfg.EmployeePerTick, stocksThisTick, cfg.InterestAPR, cfg.MarketVolatility); err != nil {
			logger.Error("tick failed", "err", err)
			os.Exit(1)
		}
		logger.Info("worker run-once completed")
		return
	}

	ticker := time.NewTicker(cfg.MarketTickEvery)
	defer ticker.Stop()

	lastStocksSpawnAt := time.Time{}
	logger.Info("worker started", "tick_every", cfg.MarketTickEvery.String(), "employee_per_tick", cfg.EmployeePerTick, "new_stocks_per_tick", cfg.NewStocksPerTick, "new_stocks_every", cfg.NewStocksEvery.String(), "volatility", cfg.MarketVolatility)
	for {
		select {
		case <-ctx.Done():
			logger.Info("worker shutdown")
			return
		case <-ticker.C:
			seasonID, err := svc.ActiveSeasonID(ctx)
			if err != nil {
				logger.Error("season read failed", "err", err)
				continue
			}
			stocksThisTick := 0
			if cfg.NewStocksPerTick > 0 {
				if cfg.NewStocksEvery <= 0 {
					stocksThisTick = cfg.NewStocksPerTick
				} else if lastStocksSpawnAt.IsZero() || time.Since(lastStocksSpawnAt) >= cfg.NewStocksEvery {
					stocksThisTick = cfg.NewStocksPerTick
				}
			}
			if err := svc.RunMarketTick(ctx, seasonID, cfg.MarketTickEvery, cfg.EmployeePerTick, stocksThisTick, cfg.InterestAPR, cfg.MarketVolatility); err != nil {
				logger.Error("market tick failed", "err", err)
				continue
			}
			if stocksThisTick > 0 {
				lastStocksSpawnAt = time.Now()
			}
			logger.Info("market tick complete", "season_id", seasonID)
		}
	}
}
