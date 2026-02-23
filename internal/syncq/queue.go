package syncq

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Command struct {
	Method         string         `json:"method"`
	Path           string         `json:"path"`
	Body           map[string]any `json:"body,omitempty"`
	IdempotencyKey string         `json:"idempotency_key"`
}

func queuePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".stk")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "queue.json"), nil
}

func Load() ([]Command, error) {
	path, err := queuePath()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Command{}, nil
		}
		return nil, err
	}
	if len(raw) == 0 {
		return []Command{}, nil
	}
	var out []Command
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func Save(commands []Command) error {
	path, err := queuePath()
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(commands, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func Push(cmd Command) error {
	commands, err := Load()
	if err != nil {
		return err
	}
	commands = append(commands, cmd)
	return Save(commands)
}
