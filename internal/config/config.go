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
	MarketTickEvery   time.Duration
	EmployeePerTick   int
	NewStocksPerTick  int
	NewStocksEvery    time.Duration
	MarketVolatility  string
	InterestAPR       float64
	StartupSeedStocks bool
}

type CLIConfig struct {
	APIBaseURL string
}

type DiscordBotConfig struct {
	DatabaseURL string
	APIBaseURL  string
	BotToken    string
	GuildID     string
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
		MarketTickEvery:   envDurationDefault("STANKS_MARKET_TICK_EVERY", 5*time.Minute),
		EmployeePerTick:   envIntDefaultAlias([]string{"EMPLOYEE_PER_TICK", "employee_per_tick"}, 1),
		NewStocksPerTick:  envIntDefaultAlias([]string{"NEW_STOCKS_PER_TICK", "new_stocks_per_tick"}, 0),
		NewStocksEvery:    envFlexibleDurationDefault([]string{"NEW_STOCKS_EVERY", "new_stocks_every"}, 0),
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
	return cfg, nil
}

func LoadCLIFromEnv() CLIConfig {
	return CLIConfig{
		APIBaseURL: normalizeCLIBaseURL(envDefault("STK_API_BASE_URL", "https://stanks-api.fxtun.dev")),
	}
}

func LoadDiscordBotFromEnv() (DiscordBotConfig, error) {
	cfg := DiscordBotConfig{
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		APIBaseURL:  normalizeCLIBaseURL(envDefault("STK_API_BASE_URL", "https://stanks-api.fxtun.dev")),
		BotToken:    strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN")),
		GuildID:     strings.TrimSpace(os.Getenv("DISCORD_GUILD_ID")),
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.APIBaseURL == "" {
		return cfg, fmt.Errorf("STK_API_BASE_URL is required")
	}
	if cfg.BotToken == "" {
		return cfg, fmt.Errorf("DISCORD_BOT_TOKEN is required")
	}
	return cfg, nil
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

func envFlexibleDurationDefault(keys []string, fallback time.Duration) time.Duration {
	for _, key := range keys {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			continue
		}
		d, err := parseFlexibleDuration(v)
		if err != nil {
			return fallback
		}
		return d
	}
	return fallback
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

func parseFlexibleDuration(raw string) (time.Duration, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d, nil
	}
	switch {
	case strings.HasSuffix(v, "min"):
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(v, "min")))
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * time.Minute, nil
	case strings.HasSuffix(v, "hr"):
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(v, "hr")))
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * time.Hour, nil
	case strings.HasSuffix(v, "d"):
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(v, "d")))
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported duration")
	}
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
