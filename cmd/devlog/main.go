package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"devlog/internal/config"
	"devlog/internal/tmux"
)

const usage = `Usage: devlog <command> [args...]

Commands:
  up       Start tmux session and browser logging
  down     Stop tmux session and flush logs
  status   Show session state and log paths
  open     Open logs directory in file manager
  help     Show this help message

Examples:
  devlog up
  devlog status
  devlog down
`

type Command func(cfg *config.Config, args []string) error

var commands = map[string]Command{
	"up":     cmdUp,
	"down":   cmdDown,
	"status": cmdStatus,
	"open":   cmdOpen,
	"help":   cmdHelp,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	command := os.Args[1]

	// Help command doesn't need config
	if command == "help" || command == "--help" || command == "-h" {
		cmdHelp(nil, nil)
		os.Exit(0)
	}

	// Find config file
	configPath := findConfigFile()
	if configPath == "" {
		fmt.Fprintln(os.Stderr, "Error: devlog.yml not found in current directory or parent directories")
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Execute command
	cmd, ok := commands[command]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n%s", command, usage)
		os.Exit(1)
	}

	if err := cmd(cfg, os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// findConfigFile searches for devlog.yml in current directory and parent directories
func findConfigFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		configPath := filepath.Join(dir, "devlog.yml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		// Go up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	return ""
}

func cmdUp(cfg *config.Config, args []string) error {
	fmt.Printf("Starting devlog session '%s'...\n", cfg.Tmux.Session)
	fmt.Printf("Logs will be written to: %s\n", cfg.ResolveLogsDir())

	// Create tmux runner
	runner := tmux.NewRunner(cfg.Tmux.Session)

	// Check if session already exists
	if runner.SessionExists() {
		return fmt.Errorf("tmux session '%s' already exists. Run 'devlog down' first or use 'tmux attach -t %s' to attach", cfg.Tmux.Session, cfg.Tmux.Session)
	}

	// Convert config windows to tmux windows
	windows := make([]tmux.WindowConfig, len(cfg.Tmux.Windows))
	for i, w := range cfg.Tmux.Windows {
		panes := make([]tmux.PaneConfig, len(w.Panes))
		for j, p := range w.Panes {
			panes[j] = tmux.PaneConfig{
				Cmd: p.Cmd,
				Log: p.Log,
			}
		}
		windows[i] = tmux.WindowConfig{
			Name:  w.Name,
			Panes: panes,
		}
	}

	// Create the tmux session
	logsDir := cfg.ResolveLogsDir()
	if err := runner.CreateSession(logsDir, windows); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	fmt.Printf("Created tmux session '%s' with %d window(s)\n", cfg.Tmux.Session, len(windows))
	fmt.Printf("Attach with: tmux attach -t %s\n", cfg.Tmux.Session)

	return nil
}

func cmdDown(cfg *config.Config, args []string) error {
	fmt.Printf("Stopping devlog session '%s'...\n", cfg.Tmux.Session)

	// Create tmux runner
	runner := tmux.NewRunner(cfg.Tmux.Session)

	// Check if session exists
	if !runner.SessionExists() {
		return fmt.Errorf("tmux session '%s' does not exist", cfg.Tmux.Session)
	}

	// Kill the session
	if err := runner.KillSession(); err != nil {
		return err
	}

	fmt.Printf("Stopped tmux session '%s'\n", cfg.Tmux.Session)

	return nil
}

func cmdStatus(cfg *config.Config, args []string) error {
	fmt.Printf("Project: %s\n", cfg.Project)
	fmt.Printf("Session: %s\n", cfg.Tmux.Session)
	fmt.Printf("Logs directory: %s\n", cfg.ResolveLogsDir())
	fmt.Printf("Run mode: %s\n", cfg.RunMode)

	// Create tmux runner
	runner := tmux.NewRunner(cfg.Tmux.Session)

	// Check session status
	if !runner.SessionExists() {
		fmt.Println("\nStatus: Not running")
		return nil
	}

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
	logsDir := cfg.ResolveLogsDir()
	fmt.Println("\nLog files:")
	for _, w := range cfg.Tmux.Windows {
		for _, p := range w.Panes {
			if p.Log != "" {
				logPath := filepath.Join(logsDir, p.Log)
				status := "missing"
				if _, err := os.Stat(logPath); err == nil {
					status = "exists"
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
			if _, err := os.Stat(browserLogPath); err == nil {
				status = "exists"
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

func cmdOpen(cfg *config.Config, args []string) error {
	logsDir := cfg.ResolveLogsDir()

	// Create logs directory if it doesn't exist
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

func cmdHelp(cfg *config.Config, args []string) error {
	fmt.Print(usage)
	return nil
}

// openInFileManager opens the given path in the system file manager
func openInFileManager(path string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{path}
	case "windows":
		cmd = "explorer"
		args = []string{path}
	default: // linux and others
		// Try xdg-open first
		cmd = "xdg-open"
		args = []string{path}
	}

	// For now, use a simpler approach - just print what we would do
	// TODO: Implement actual file manager opening (US-009)
	fmt.Printf("Would open with: %s %v\n", cmd, args)
	return nil
}
