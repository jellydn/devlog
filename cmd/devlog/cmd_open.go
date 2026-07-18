package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/jellydn/devlog/internal/config"
	"github.com/jellydn/devlog/internal/tmux"
)

func cmdOpen(cfg *config.Config, args []string) error {
	runner := tmux.NewRunner(cfg.Tmux.Session)

	logsDir := ""
	if runner.SessionExists() {
		logsDir = runner.GetLogsDir()
	}
	if logsDir == "" {
		logsDir = cfg.LogsDir
	}

	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Open in file manager
	if err := openInFileManager(logsDir); err != nil {
		return fmt.Errorf("failed to open logs directory: %w", err)
	}

	fmt.Printf("Opened: %s\n", logsDir)
	return nil
}

func openInFileManager(path string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{path}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", "", path}
	default:
		cmd = "xdg-open"
		args = []string{path}
	}

	return exec.Command(cmd, args...).Start()
}
