package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Email        string `json:"email"`
	UserID       string `json:"user_id"`
}

func baseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".stk")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func sessionPath() (string, error) {
	dir, err := baseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session.json"), nil
}

func SaveSession(s Session) error {
	path, err := sessionPath()
	if err != nil {
		return err
	}
	body, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return err
	}
	return nil
}

func LoadSession() (Session, error) {
	path, err := sessionPath()
	if err != nil {
		return Session{}, err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return Session{}, err
	}
	var s Session
	if err := json.Unmarshal(body, &s); err != nil {
		return Session{}, err
	}
	if strings.TrimSpace(s.AccessToken) == "" {
		return Session{}, fmt.Errorf("no access token found in session")
	}
	return s, nil
}

func ClearSession() error {
	path, err := sessionPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	return os.Remove(path)
}
