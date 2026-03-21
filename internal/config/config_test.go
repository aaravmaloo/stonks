package config

import (
	"testing"
	"time"
)

func TestNormalizeCLIBaseURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "keeps https", in: "https://stanks-api.fxtun.dev/", want: "https://stanks-api.fxtun.dev"},
		{name: "adds https for host", in: "stanks-api.fxtun.dev", want: "https://stanks-api.fxtun.dev"},
		{name: "adds http for localhost", in: "localhost:8080", want: "http://localhost:8080"},
		{name: "adds http for loopback", in: "127.0.0.1:8080/", want: "http://127.0.0.1:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeCLIBaseURL(tt.in); got != tt.want {
				t.Fatalf("normalizeCLIBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestLoadCLIFromEnvDefaultUsesHTTPS(t *testing.T) {
	t.Setenv("STK_API_BASE_URL", "")

	cfg := LoadCLIFromEnv()
	if cfg.APIBaseURL != "https://stanks-api.fxtun.dev" {
		t.Fatalf("LoadCLIFromEnv().APIBaseURL = %q, want %q", cfg.APIBaseURL, "https://stanks-api.fxtun.dev")
	}
}

func TestLoadAPIFromEnvEmployeePerTickAlias(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("SUPABASE_URL", "https://example.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "anon")
	t.Setenv("employee_per_tick", "7")

	cfg, err := LoadAPIFromEnv()
	if err != nil {
		t.Fatalf("LoadAPIFromEnv() error = %v", err)
	}
	if cfg.EmployeePerTick != 7 {
		t.Fatalf("LoadAPIFromEnv().EmployeePerTick = %d, want 7", cfg.EmployeePerTick)
	}
}

func TestLoadAPIFromEnvEmployeePerTickClampsNegative(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("SUPABASE_URL", "https://example.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "anon")
	t.Setenv("EMPLOYEE_PER_TICK", "-4")

	cfg, err := LoadAPIFromEnv()
	if err != nil {
		t.Fatalf("LoadAPIFromEnv() error = %v", err)
	}
	if cfg.EmployeePerTick != 0 {
		t.Fatalf("LoadAPIFromEnv().EmployeePerTick = %d, want 0", cfg.EmployeePerTick)
	}
}

func TestLoadAPIFromEnvNewStocksPerTickAlias(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("SUPABASE_URL", "https://example.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "anon")
	t.Setenv("new_stocks_per_tick", "9")

	cfg, err := LoadAPIFromEnv()
	if err != nil {
		t.Fatalf("LoadAPIFromEnv() error = %v", err)
	}
	if cfg.NewStocksPerTick != 9 {
		t.Fatalf("LoadAPIFromEnv().NewStocksPerTick = %d, want 9", cfg.NewStocksPerTick)
	}
}

func TestParseFlexibleDuration(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{in: "5s", want: 5 * time.Second},
		{in: "7min", want: 7 * time.Minute},
		{in: "2hr", want: 2 * time.Hour},
		{in: "3d", want: 72 * time.Hour},
	}

	for _, tt := range tests {
		got, err := parseFlexibleDuration(tt.in)
		if err != nil {
			t.Fatalf("parseFlexibleDuration(%q) error = %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("parseFlexibleDuration(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
