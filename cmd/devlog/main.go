package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"devlog/internal/config"
	"devlog/internal/natmsg"
	"devlog/internal/tmux"
)

const usage = `Usage: devlog <command> [args...]

Commands:
  up       Start tmux session and browser logging
  down     Stop tmux session and flush logs
  status   Show session state and log paths
  open     Open logs directory in file manager
  register Register native messaging host for browser logging
  help     Show this help message

Examples:
  devlog up
  devlog status
  devlog down
  devlog register --chrome --extension-id abcdefghijklmnop
`

type Command func(cfg *config.Config, args []string) error

var commands = map[string]Command{
	"up":       cmdUp,
	"down":     cmdDown,
	"status":   cmdStatus,
	"open":     cmdOpen,
	"help":     cmdHelp,
	"register": cmdRegister,
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

	// Register command doesn't need config (just needs to find the binary)
	if command == "register" {
		cmd, ok := commands[command]
		if !ok {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n%s", command, usage)
			os.Exit(1)
		}
		if err := cmd(nil, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
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

func cmdRegister(cfg *config.Config, args []string) error {
	hostPath, err := natmsg.FindDevlogHostBinary()
	if err != nil {
		return fmt.Errorf("failed to find devlog-host binary: %w", err)
	}

	installChrome := false
	installFirefox := false
	extensionID := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--chrome":
			installChrome = true
		case "--firefox":
			installFirefox = true
		case "--extension-id":
			if i+1 < len(args) {
				extensionID = args[i+1]
				i++
			} else {
				return fmt.Errorf("--extension-id requires a value")
			}
		case "--help", "-h":
			fmt.Print(`Usage: devlog register [options]

Register native messaging host for browser logging.

Options:
  --chrome         Register for Google Chrome
  --firefox        Register for Mozilla Firefox
  --extension-id   Chrome extension ID (required for --chrome)
  --help, -h       Show this help message

Examples:
  devlog register --chrome --extension-id abcdefghijklmnopqrstuvwxyz123456
  devlog register --firefox
  devlog register --chrome --firefox --extension-id abcdefghijklmnopqrstuvwxyz123456
`)
			return nil
		default:
			return fmt.Errorf("unknown argument: %s (use --help for usage)", arg)
		}
	}

	if !installChrome && !installFirefox {
		installChrome = true
		installFirefox = true
	}

	if installChrome && extensionID == "" {
		return fmt.Errorf("--extension-id is required when registering for Chrome")
	}

	fmt.Printf("devlog-host binary: %s\n", hostPath)

	if installChrome {
		fmt.Printf("Registering for Chrome...\n")
		if err := natmsg.InstallChromeManifest(hostPath, extensionID); err != nil {
			return fmt.Errorf("failed to register Chrome manifest: %w", err)
		}
		dir := natmsg.GetChromeNativeMessagingDir()
		fmt.Printf("  Installed to: %s\n", dir)
	}

	if installFirefox {
		fmt.Printf("Registering for Firefox...\n")
		if err := natmsg.InstallFirefoxManifest(hostPath); err != nil {
			return fmt.Errorf("failed to register Firefox manifest: %w", err)
		}
		dir := natmsg.GetFirefoxNativeMessagingDir()
		fmt.Printf("  Installed to: %s\n", dir)
	}

	fmt.Println("Registration complete!")
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
		cmd = "cmd"
		args = []string{"/c", "start", "", path}
	default:
		cmd = "xdg-open"
		args = []string{path}
	}

	return exec.Command(cmd, args...).Start()
}
