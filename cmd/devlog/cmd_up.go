package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jellydn/devlog/internal/config"
	"github.com/jellydn/devlog/internal/tmux"
)

func cmdUp(cfg *config.Config, args []string) error {
	fmt.Printf("Starting devlog session '%s'...\n", cfg.Tmux.Session)
	fmt.Printf("Logs will be written to: %s\n", cfg.ResolveLogsDir())

	// Create tmux runner
	runner := tmux.NewRunner(cfg.Tmux.Session)

	// Check if session already exists
	if runner.SessionExists() {
		return fmt.Errorf("tmux session '%s' already exists. Run 'devlog down' first or use 'tmux attach -t %s' to attach", cfg.Tmux.Session, cfg.Tmux.Session)
	}

	// Clean up old log runs if retention policy is configured
	if result, err := cfg.CleanupOldRuns(false); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cleanup old runs: %v\n", err)
	} else if result != nil {
		for _, dir := range result.Removed {
			fmt.Printf("Removed old log directory: %s\n", dir)
		}
		for dir, remErr := range result.Failed {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove %s: %v\n", dir, remErr)
		}
	}

	// Create the tmux session
	logsDir := cfg.ResolveLogsDir()
	if err := runner.CreateSession(logsDir, cfg.Tmux.Windows); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	fmt.Printf("Created tmux session '%s' with %d window(s)\n", cfg.Tmux.Session, len(cfg.Tmux.Windows))

	// Set up browser logging wrapper if configured
	if len(cfg.Browser.URLs) > 0 && cfg.Browser.File != "" {
		browserLogPath := filepath.Join(logsDir, cfg.Browser.File)
		if err := ensureFileExists(browserLogPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to prepare browser log file: %v\n", err)
		}
		if err := writeBrowserHostWrapper(cfg.Tmux.Session, browserLogPath, cfg.Browser.Levels); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to set up browser logging wrapper: %v\n", err)
		} else {
			fmt.Println("Browser logging: ready (wrapper updated)")
		}
	}

	fmt.Printf("Attach with: devlog attach\n")

	return nil
}
