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
	EmployeePerTick   int
	NewStocksPerTick  int
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
		EmployeePerTick:   envIntDefaultAlias([]string{"EMPLOYEE_PER_TICK", "employee_per_tick"}, 1),
		NewStocksPerTick:  envIntDefaultAlias([]string{"NEW_STOCKS_PER_TICK", "new_stocks_per_tick"}, 0),
		MarketVolatility:  envVolatilityDefault(),
		InterestAPR:       envFloatDefault("STANKS_INTEREST_APR", 0.18),
		StartupSeedStocks: envBoolDefault("STANKS_STARTUP_SEED_STOCKS", true),
	}
	if cfg.EmployeePerTick < 0 {
		cfg.EmployeePerTick = 0
	}
	if cfg.NewStocksPerTick < 0 {
		cfg.NewStocksPerTick = 0
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
		APIBaseURL: normalizeCLIBaseURL(envDefault("STK_API_BASE_URL", "https://stanks-api.fxtun.dev")),
	}
}

func normalizeCLIBaseURL(raw string) string {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		return ""
	}
	if strings.Contains(base, "://") {
		return base
	}
	if strings.HasPrefix(base, "localhost") || strings.HasPrefix(base, "127.0.0.1") {
		return "http://" + base
	}
	return "https://" + base
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

func envIntDefaultAlias(keys []string, fallback int) int {
	for _, key := range keys {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return fallback
		}
		return n
	}
	return fallback
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
