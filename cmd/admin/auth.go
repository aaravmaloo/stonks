package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"stanks/internal/config"
)

func loadAdminEnv() {
	if custom := strings.TrimSpace(os.Getenv("STANKS_ENV_FILE")); custom != "" {
		_ = config.LoadDotEnvIfPresent(custom)
		return
	}

	_ = config.LoadDotEnvIfPresent(".env")

	exePath, err := os.Executable()
	if err != nil {
		return
	}
	exeDir := filepath.Dir(exePath)
	_ = config.LoadDotEnvIfPresent(filepath.Join(exeDir, ".env"))
}

func requireAdminAccess() error {
	if strings.TrimSpace(os.Getenv("DATABASE_URL")) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if strings.TrimSpace(os.Getenv("ADMIN_USRN")) == "" || strings.TrimSpace(os.Getenv("ADMIN_PASS")) == "" {
		return fmt.Errorf("ADMIN_USRN and ADMIN_PASS must be set in the server env")
	}
	return nil
}
