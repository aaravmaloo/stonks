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

func requireAdminAccess(store *adminStore) error {
	if strings.TrimSpace(store.baseURL) == "" {
		return fmt.Errorf("STK_API_BASE_URL is required")
	}
	if store.username != "" && store.password != "" {
		return nil
	}
	username, err := promptRequired("Admin username")
	if err != nil {
		return err
	}
	password, err := promptPassword("Admin password")
	if err != nil {
		return err
	}
	store.username = username
	store.password = password
	return nil
}
