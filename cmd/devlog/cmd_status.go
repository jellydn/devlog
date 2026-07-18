package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jellydn/devlog/internal/config"
	"github.com/jellydn/devlog/internal/tmux"
)

func cmdStatus(cfg *config.Config, args []string) error {
	fmt.Printf("Project: %s\n", cfg.Project)
	fmt.Printf("Session: %s\n", cfg.Tmux.Session)
	fmt.Printf("Run mode: %s\n", cfg.RunMode)

	// Create tmux runner
	runner := tmux.NewRunner(cfg.Tmux.Session)

	// Check session status
	if !runner.SessionExists() {
		fmt.Println("\nStatus: Not running")
		return nil
	}

	// Resolve logs directory from the running session
	logsDir := resolveStatusLogsDir(runner.GetLogsDir(), cfg)
	fmt.Printf("Logs directory: %s\n", logsDir)

	// Get session info
	info, err := runner.GetSessionInfo()
	if err != nil {
		return fmt.Errorf("failed to get session info: %w", err)
	}

	fmt.Println("\nStatus: Running")
	fmt.Printf("Windows (%d):\n", len(info.Windows))
	for _, w := range info.Windows {
		fmt.Printf("  [%d] %s (%d panes)\n", w.Index, w.Name, w.PaneCount)
		for _, p := range w.Panes {
			fmt.Printf("      pane %s: %s\n", p.ID, p.Command)
		}
	}

	// Show log files
	fmt.Println("\nLog files:")
	for _, w := range cfg.Tmux.Windows {
		for _, p := range w.Panes {
			if p.Log != "" {
				logPath := filepath.Join(logsDir, p.Log)
				status := "missing"
				if fi, err := os.Stat(logPath); err == nil {
					status = fmt.Sprintf("%d bytes", fi.Size())
				}
				fmt.Printf("  %s (%s)\n", logPath, status)
			}
		}
	}

	// Show browser logging status
	fmt.Println("\nBrowser logging:")
	if len(cfg.Browser.URLs) > 0 {
		fmt.Printf("  Status: enabled\n")
		fmt.Printf("  URLs monitored (%d):\n", len(cfg.Browser.URLs))
		for _, url := range cfg.Browser.URLs {
			fmt.Printf("    - %s\n", url)
		}
		if cfg.Browser.File != "" {
			browserLogPath := filepath.Join(logsDir, cfg.Browser.File)
			status := "missing"
			if fi, err := os.Stat(browserLogPath); err == nil {
				status = fmt.Sprintf("%d bytes", fi.Size())
			}
			fmt.Printf("  Log file: %s (%s)\n", browserLogPath, status)
		}
		if len(cfg.Browser.Levels) > 0 {
			fmt.Printf("  Levels: %v\n", cfg.Browser.Levels)
		}
	} else {
		fmt.Printf("  Status: disabled (no URLs configured)\n")
	}

	return nil
}

func resolveStatusLogsDir(runningLogsDir string, cfg *config.Config) string {
	if runningLogsDir != "" {
		return runningLogsDir
	}
	if cfg.RunMode != "timestamped" {
		return cfg.LogsDir
	}
	latestRun := latestRunDir(cfg.LogsDir)
	if latestRun != "" {
		return latestRun
	}
	return cfg.LogsDir
}

func latestRunDir(baseLogsDir string) string {
	entries, err := os.ReadDir(baseLogsDir)
	if err != nil {
		return ""
	}

	latestName := ""
	latestModTime := int64(-1)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().UnixNano() > latestModTime {
			latestModTime = info.ModTime().UnixNano()
			latestName = entry.Name()
		}
	}

	if latestName == "" {
		return ""
	}
	return filepath.Join(baseLogsDir, latestName)
}
