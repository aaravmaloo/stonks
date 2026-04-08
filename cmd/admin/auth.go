package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func requireAdminLogin() error {
	wantUser := strings.TrimSpace(os.Getenv("ADMIN_USRN"))
	wantPass := strings.TrimSpace(os.Getenv("ADMIN_PASS"))
	if wantUser == "" || wantPass == "" {
		return fmt.Errorf("ADMIN_USRN and ADMIN_PASS must be set")
	}

	sess, err := loadAdminSession()
	if err != nil {
		return fmt.Errorf("admin login required: run `admin login`")
	}
	if sess.Username != wantUser {
		return fmt.Errorf("admin session is stale: run `admin login` again")
	}
	return nil
}

func adminBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".stanks-admin"), nil
}

func adminSessionPath() (string, error) {
	base, err := adminBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "session.json"), nil
}

func saveAdminSession(sess adminSession) error {
	base, err := adminBaseDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return fmt.Errorf("create admin dir: %w", err)
	}
	path, err := adminSessionPath()
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	return nil
}

func loadAdminSession() (adminSession, error) {
	path, err := adminSessionPath()
	if err != nil {
		return adminSession{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return adminSession{}, err
	}
	var sess adminSession
	if err := json.Unmarshal(raw, &sess); err != nil {
		return adminSession{}, fmt.Errorf("decode session: %w", err)
	}
	if strings.TrimSpace(sess.Username) == "" {
		return adminSession{}, fmt.Errorf("invalid session")
	}
	return sess, nil
}

func clearAdminSession() error {
	path, err := adminSessionPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove session: %w", err)
	}
	return nil
}
