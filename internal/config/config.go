package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type APIConfig struct {
	Addr              string
	DatabaseURL       string
	SupabaseURL       string
	SupabaseAnonKey   string
	MarketTickEvery   time.Duration
	MarketVolatility  string
	InterestAPR       float64
	StartupSeedStocks bool
}

type CLIConfig struct {
	APIBaseURL string
}

func LoadAPIFromEnv() (APIConfig, error) {
	addr := os.Getenv("PORT")
	if addr != "" {
		if !strings.HasPrefix(addr, ":") {
			addr = ":" + addr
		}
	} else {
		addr = envDefault("STANKS_API_ADDR", ":8080")
	}

	cfg := APIConfig{
		Addr:              addr,
		DatabaseURL:       strings.TrimSpace(os.Getenv("DATABASE_URL")),
		SupabaseURL:       strings.TrimRight(strings.TrimSpace(os.Getenv("SUPABASE_URL")), "/"),
		SupabaseAnonKey:   strings.TrimSpace(os.Getenv("SUPABASE_ANON_KEY")),
		MarketTickEvery:   envDurationDefault("STANKS_MARKET_TICK_EVERY", 5*time.Minute),
		MarketVolatility:  envVolatilityDefault(),
		InterestAPR:       envFloatDefault("STANKS_INTEREST_APR", 0.18),
		StartupSeedStocks: envBoolDefault("STANKS_STARTUP_SEED_STOCKS", true),
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.SupabaseURL == "" {
		return cfg, fmt.Errorf("SUPABASE_URL is required")
	}
	if cfg.SupabaseAnonKey == "" {
		return cfg, fmt.Errorf("SUPABASE_ANON_KEY is required")
	}
	return cfg, nil
}

func LoadCLIFromEnv() CLIConfig {
	return CLIConfig{
		APIBaseURL: strings.TrimRight(envDefault("STK_API_BASE_URL", "http://localhost:8080"), "/"),
	}
}

func envDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func envDurationDefault(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func envFloatDefault(key string, fallback float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func envBoolDefault(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envVolatilityDefault() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("VOLATILITY")))
	if v == "" {
		v = strings.ToLower(strings.TrimSpace(os.Getenv("STANKS_MARKET_VOLATILITY")))
	}
	switch v {
	case "calm", "mor", "wild":
		return v
	default:
		return "mor"
	}
}
