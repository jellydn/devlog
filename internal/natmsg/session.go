package natmsg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SessionState represents the active devlog session's browser logging config
type SessionState struct {
	LogPath string   `json:"log_path"`
	Levels  []string `json:"levels"`
}

// sessionStatePath returns the path to the active session state file
func sessionStatePath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get cache directory: %w", err)
	}
	return filepath.Join(cacheDir, "devlog", "active-session.json"), nil
}

// WriteSessionState writes the active session state file
func WriteSessionState(logPath string, levels []string) error {
	statePath, err := sessionStatePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	state := SessionState{LogPath: logPath, Levels: levels}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session state: %w", err)
	}
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session state: %w", err)
	}
	return nil
}

// ReadSessionState reads the active session state file
func ReadSessionState() (*SessionState, error) {
	statePath, err := sessionStatePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session state: %w", err)
	}
	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse session state: %w", err)
	}
	return &state, nil
}

// RemoveSessionState removes the active session state file
func RemoveSessionState() error {
	statePath, err := sessionStatePath()
	if err != nil {
		return err
	}
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove session state: %w", err)
	}
	return nil
}
