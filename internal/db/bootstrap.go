package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

func EnsureTables(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS auth;
		CREATE SCHEMA IF NOT EXISTS game;
		CREATE TABLE IF NOT EXISTS auth.users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			access_token TEXT UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		ALTER TABLE auth.users
			ADD COLUMN IF NOT EXISTS password_hash TEXT,
			ADD COLUMN IF NOT EXISTS access_token TEXT,
			ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
		CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_users_access_token
			ON auth.users (access_token)
			WHERE access_token IS NOT NULL;
		CREATE TABLE IF NOT EXISTS game.discord_sessions (
			discord_user_id TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			access_token TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_discord_sessions_updated_at
			ON game.discord_sessions (updated_at DESC);
		ALTER TABLE game.businesses
			ADD COLUMN IF NOT EXISTS strategy TEXT NOT NULL DEFAULT 'balanced',
			ADD COLUMN IF NOT EXISTS marketing_level INT NOT NULL DEFAULT 0,
			ADD COLUMN IF NOT EXISTS rd_level INT NOT NULL DEFAULT 0,
			ADD COLUMN IF NOT EXISTS automation_level INT NOT NULL DEFAULT 0,
			ADD COLUMN IF NOT EXISTS compliance_level INT NOT NULL DEFAULT 0,
			ADD COLUMN IF NOT EXISTS brand_bps INT NOT NULL DEFAULT 10000,
			ADD COLUMN IF NOT EXISTS operational_health_bps INT NOT NULL DEFAULT 10000,
			ADD COLUMN IF NOT EXISTS cash_reserve_micros BIGINT NOT NULL DEFAULT 0,
			ADD COLUMN IF NOT EXISTS last_event TEXT NOT NULL DEFAULT '',
			ADD COLUMN IF NOT EXISTS seat_capacity BIGINT NOT NULL DEFAULT 60000,
			ADD COLUMN IF NOT EXISTS employee_count BIGINT NOT NULL DEFAULT 0;
		UPDATE game.businesses
		SET seat_capacity = 60000
		WHERE seat_capacity IS NULL OR seat_capacity <= 0;
		UPDATE game.businesses
		SET employee_count = 0
		WHERE employee_count IS NULL OR employee_count < 0;
		UPDATE game.businesses b
		SET employee_count = counts.employee_count
		FROM (
			SELECT business_id, COUNT(*)::bigint AS employee_count
			FROM game.business_employees
			GROUP BY business_id
		) counts
		WHERE b.id = counts.business_id
		  AND COALESCE(b.employee_count, 0) <> counts.employee_count;
	`); err != nil {
		return fmt.Errorf("ensure auth tables: %w", err)
	}

	migrationDir, err := resolveMigrationsDir()
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		return nil
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, filepath.Join(migrationDir, entry.Name()))
	}
	sort.Strings(files)

	for _, path := range files {
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", path, err)
		}
		if len(sqlBytes) == 0 {
			continue
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("apply migration %s: %w", filepath.Base(path), err)
		}
	}
	return nil
}

func resolveMigrationsDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	current := wd
	for i := 0; i < 8; i++ {
		goMod := filepath.Join(current, "go.mod")
		if _, err := os.Stat(goMod); err == nil {
			return filepath.Join(current, "migrations"), nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", fmt.Errorf("repo root not found")
}
