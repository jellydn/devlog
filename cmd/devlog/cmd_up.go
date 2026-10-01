package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jellydn/devlog/internal/browsersession"
	"github.com/jellydn/devlog/internal/config"
	"github.com/jellydn/devlog/internal/logrotate"
	"github.com/jellydn/devlog/internal/tmux"
)

func cmdUp(cfg *config.Config, args []string) error {
	fmt.Printf("Starting devlog session '%s'...\n", cfg.Tmux.Session)

	// Create tmux runner
	runner := tmux.NewRunner(cfg.Tmux.Session)

	// Check if session already exists
	if runner.SessionExists() {
		return fmt.Errorf("tmux session '%s' already exists. Run 'devlog down' first or use 'tmux attach -t %s' to attach", cfg.Tmux.Session, cfg.Tmux.Session)
	}

	// Clean up old log runs if retention policy is configured
	if cfg.RunMode == "timestamped" {
		policy := logrotate.Policy{MaxRuns: cfg.MaxRuns, RetentionDays: cfg.RetentionDays}
		if result, err := logrotate.Cleanup(cfg.LogsDir, policy, false); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to cleanup old runs: %v\n", err)
		} else {
			for _, dir := range result.Removed {
				fmt.Printf("Removed old log directory: %s\n", dir)
			}
			for dir, remErr := range result.Failed {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove %s: %v\n", dir, remErr)
			}
		}
	}

	// Create the tmux session (Runner resolves logs dir internally)
	sessionCfg := tmux.SessionConfig{
		Session: cfg.Tmux.Session,
		LogsDir: cfg.LogsDir,
		RunMode: cfg.RunMode,
		Windows: cfg.Tmux.Windows,
	}
	if err := runner.CreateSession(sessionCfg); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}
	logsDir := runner.GetLogsDir()
	fmt.Printf("Logs will be written to: %s\n", logsDir)

	fmt.Printf("Created tmux session '%s' with %d window(s)\n", cfg.Tmux.Session, len(cfg.Tmux.Windows))

	// Set up browser logging wrapper if configured
	if len(cfg.Browser.URLs) > 0 && cfg.Browser.File != "" {
		browserLogPath := filepath.Join(logsDir, cfg.Browser.File)
		if err := ensureFileExists(browserLogPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to prepare browser log file: %v\n", err)
		}
		bs := browsersession.New(manifestAdapter{}, tmuxSessionChecker{})
		if err := bs.Start(cfg.Tmux.Session, browserLogPath, cfg.Browser.Levels); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to set up browser logging wrapper: %v\n", err)
		} else {
			fmt.Println("Browser logging: ready (wrapper updated)")
		}
	}

	fmt.Printf("Attach with: devlog attach\n")

	return nil
}
