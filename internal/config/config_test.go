package config

import "testing"

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
