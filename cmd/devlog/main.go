package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jellydn/devlog/internal/config"
	"github.com/jellydn/devlog/internal/natmsg"
	"github.com/jellydn/devlog/internal/tmux"
)

const usage = `Usage: devlog <command> [args...]

Commands:
  init     Create a devlog.yml template in current directory
  up       Start tmux session and browser logging
  down     Stop tmux session and flush logs
  attach   Attach to the running tmux session
  status   Show session state and log paths
  ls       List log runs
  open     Open logs directory in file manager
  register Register native messaging host for browser logging
  help     Show this help message

Examples:
  devlog init
  devlog up
  devlog attach
  devlog status
  devlog ls
  devlog down
  devlog register --chrome --extension-id abcdefghijklmnop
`

type Command func(cfg *config.Config, args []string) error

var commands = map[string]Command{
	"init":     cmdInit,
	"up":       cmdUp,
	"down":     cmdDown,
	"attach":   cmdAttach,
	"status":   cmdStatus,
	"ls":       cmdLs,
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

	// Commands that don't need config
	if command == "init" || command == "register" {
		runCommandWithoutConfig(command)
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

func runCommandWithoutConfig(command string) {
	cmd, ok := commands[command]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n%s", command, usage)
		os.Exit(1)
	}
	if err := cmd(nil, os.Args[2:]); err != nil {
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

func cmdInit(cfg *config.Config, args []string) error {
	// Check if devlog.yml already exists
	configPath := filepath.Join(".", "devlog.yml")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("devlog.yml already exists in current directory")
	}

	// Get current directory name for defaults
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(cwd)

	// Detect if this is a monorepo by checking for common patterns
	isMonorepo := false
	monorepoIndicators := []string{"packages", "apps", "services"}
	for _, dir := range monorepoIndicators {
		if _, err := os.Stat(dir); err == nil {
			isMonorepo = true
			break
		}
	}

	// Create template content
	template := generateTemplate(projectName, isMonorepo)

	// Write the file
	if err := os.WriteFile(configPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to write devlog.yml: %w", err)
	}

	fmt.Printf("Created devlog.yml in current directory\n")
	fmt.Printf("Project: %s\n", projectName)
	if isMonorepo {
		fmt.Printf("Detected monorepo structure\n")
	}
	fmt.Printf("\nEdit devlog.yml to customize your configuration, then run:\n")
	fmt.Printf("  devlog up\n")

	return nil
}

// generateTemplate creates the YAML template content
func generateTemplate(projectName string, isMonorepo bool) string {
	if isMonorepo {
		return fmt.Sprintf(`version: "1.0"
project: %s
logs_dir: ./logs
run_mode: timestamped # timestamped | overwrite

tmux:
  session: %s
  windows:
    - name: dev
      panes:
        - cmd: npm run dev
          log: server/web.log
        - cmd: pnpm --filter api dev
          log: server/api.log

browser:
  # Matches any port and path on localhost (e.g., http://localhost:3000/app)
  urls:
    - "http://localhost:*/*"
  file: browser/console.log
  levels:
    - error
    - warn
    - info
    - log
`, projectName, projectName)
	}

	return fmt.Sprintf(`version: "1.0"
project: %s
logs_dir: ./logs
run_mode: timestamped # timestamped | overwrite

tmux:
  session: %s
  windows:
    - name: dev
      panes:
        - cmd: npm run dev
          log: server.log

browser:
  # Matches any port and path on localhost (e.g., http://localhost:3000/app)
  urls:
    - "http://localhost:*/*"
  file: browser.log
  levels:
    - error
    - warn
    - info
    - log
`, projectName, projectName)
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

	// Set up browser logging wrapper if configured
	if len(cfg.Browser.URLs) > 0 && cfg.Browser.File != "" {
		browserLogPath := filepath.Join(logsDir, cfg.Browser.File)
		if err := writeBrowserHostWrapper(cfg.Tmux.Session, browserLogPath, cfg.Browser.Levels); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to set up browser logging wrapper: %v\n", err)
		} else {
			fmt.Println("Browser logging: ready (wrapper updated)")
		}
	}

	fmt.Printf("Attach with: devlog attach\n")

	return nil
}

func cmdDown(cfg *config.Config, args []string) error {
	fmt.Printf("Stopping devlog session '%s'...\n", cfg.Tmux.Session)

	// Create tmux runner
	runner := tmux.NewRunner(cfg.Tmux.Session)

	// Check if session exists
	if !runner.SessionExists() {
		restoreBrowserHostWrapper(cfg.Tmux.Session)
		return fmt.Errorf("tmux session '%s' does not exist", cfg.Tmux.Session)
	}

	// Kill the session
	if err := runner.KillSession(); err != nil {
		return err
	}

	// Restore native messaging manifest to point to the real binary
	restoreBrowserHostWrapper(cfg.Tmux.Session)

	fmt.Printf("Stopped tmux session '%s'\n", cfg.Tmux.Session)

	return nil
}

func cmdAttach(cfg *config.Config, args []string) error {
	runner := tmux.NewRunner(cfg.Tmux.Session)

	if !runner.SessionExists() {
		return fmt.Errorf("tmux session '%s' is not running. Run 'devlog up' first", cfg.Tmux.Session)
	}

	cmd := exec.Command("tmux", "attach", "-t", cfg.Tmux.Session)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

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
	logsDir := runner.GetLogsDir()
	if logsDir == "" {
		logsDir = cfg.ResolveLogsDir()
	}
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

func cmdLs(cfg *config.Config, args []string) error {
	logsDir := cfg.LogsDir

	entries, err := os.ReadDir(logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("No log runs found (logs directory '%s' does not exist)\n", logsDir)
			return nil
		}
		return fmt.Errorf("failed to read logs directory: %w", err)
	}

	var dirs []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e)
		}
	}

	if cfg.RunMode == "timestamped" {
		if len(dirs) == 0 {
			fmt.Println("No log runs found")
			return nil
		}
		fmt.Printf("Log runs in %s (%d):\n", logsDir, len(dirs))
		for _, d := range dirs {
			info, err := d.Info()
			if err != nil {
				fmt.Printf("  %s\n", d.Name())
				continue
			}
			logFiles := countFiles(filepath.Join(logsDir, d.Name()))
			fmt.Printf("  %s  (%d files, %s)\n", d.Name(), logFiles, info.ModTime().Format("Jan 02 15:04"))
		}
	} else {
		files := countFiles(logsDir)
		fmt.Printf("Logs directory: %s (%d files)\n", logsDir, files)
	}

	return nil
}

func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}
	return count
}

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
		if extensionID != "" {
			installChrome = true
		}
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
		dirs := natmsg.GetFirefoxNativeMessagingDirs()
		fmt.Printf("  Installed to:\n")
		for _, dir := range dirs {
			fmt.Printf("    - %s\n", dir)
		}
	}

	fmt.Println("Registration complete!")
	return nil
}

func browserHostWrapperPath(session string) string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	return filepath.Join(
		cacheDir,
		"devlog",
		"wrappers",
		fmt.Sprintf("devlog-host-wrapper-%s.sh", sanitizeSessionForFileName(session)),
	)
}

func sanitizeSessionForFileName(session string) string {
	if session == "" {
		return "default"
	}

	var b strings.Builder
	b.Grow(len(session))
	for _, r := range session {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}

	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "default"
	}
	return out
}

func writeBrowserHostWrapper(session string, browserLogPath string, levels []string) error {
	hostPath, err := natmsg.FindDevlogHostBinary()
	if err != nil {
		return err
	}

	absLogPath, err := filepath.Abs(browserLogPath)
	if err != nil {
		return err
	}

	wrapperPath := browserHostWrapperPath(session)
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0755); err != nil {
		return err
	}

	// Build script with proper shell escaping using positional parameters
	// Use "$@" to safely pass arguments without re-parsing by the shell
	var scriptArgs []string
	scriptArgs = append(scriptArgs, shellQuote(hostPath), shellQuote(absLogPath))
	for _, level := range levels {
		scriptArgs = append(scriptArgs, shellQuote(level))
	}
	script := fmt.Sprintf("#!/bin/sh\nexec %s\n", strings.Join(scriptArgs, " "))
	if err := os.WriteFile(wrapperPath, []byte(script), 0755); err != nil {
		return err
	}

	if err := natmsg.UpdateManifestPath(wrapperPath); err != nil {
		return fmt.Errorf("failed to update native messaging manifest: %w", err)
	}

	return nil
}

func restoreBrowserHostWrapper(session string) {
	hostPath, err := natmsg.FindDevlogHostBinary()
	if err != nil {
		return
	}

	wrapperPath := browserHostWrapperPath(session)
	inUse, err := natmsg.IsManifestPathInUse(wrapperPath)
	if err == nil && inUse {
		natmsg.UpdateManifestPath(hostPath)
	}

	os.Remove(wrapperPath)
}

// shellQuote returns a shell-escaped version of the string using single quotes.
// Any single quotes in the input are escaped as '\”' to safely include them.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
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
