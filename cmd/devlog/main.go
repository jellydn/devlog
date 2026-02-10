package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"devlog/internal/config"
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

	// TODO: Implement actual tmux session creation (US-003)
	fmt.Println("(tmux session creation not yet implemented)")

	return nil
}

func cmdDown(cfg *config.Config, args []string) error {
	fmt.Printf("Stopping devlog session '%s'...\n", cfg.Tmux.Session)

	// TODO: Implement actual tmux session cleanup (US-004)
	fmt.Println("(tmux session cleanup not yet implemented)")

	return nil
}

func cmdStatus(cfg *config.Config, args []string) error {
	fmt.Printf("Project: %s\n", cfg.Project)
	fmt.Printf("Session: %s\n", cfg.Tmux.Session)
	fmt.Printf("Logs directory: %s\n", cfg.ResolveLogsDir())
	fmt.Printf("Run mode: %s\n", cfg.RunMode)

	// TODO: Check actual tmux session status (US-005)
	fmt.Println("\n(tmux session status check not yet implemented)")

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
