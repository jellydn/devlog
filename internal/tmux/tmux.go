package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jellydn/devlog/internal/config"
	"github.com/jellydn/devlog/internal/fileutil"
	"github.com/jellydn/devlog/internal/shellescape"
)

// Runner handles tmux session operations
type Runner struct {
	sessionName string
	logsDir     string // resolved logs directory for this session (set by CreateSession)
}

// SessionConfig provides everything needed to create a tmux session.
// Window/pane types remain in internal/config (Option A: keep dependency direction).
type SessionConfig struct {
	Session string
	LogsDir string // base logs directory (e.g. "./logs")
	RunMode string // "timestamped" or "overwrite"
	Windows []config.WindowConfig
}

// NewRunner creates a new tmux runner for the given session
func NewRunner(sessionName string) *Runner {
	return &Runner{sessionName: sessionName}
}

// SessionExists checks if the tmux session already exists
func (r *Runner) SessionExists() bool {
	cmd := exec.Command("tmux", "has-session", "-t", r.sessionName)
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	return err == nil
}

// CreateSession creates a new tmux session with the given windows and panes.
// It resolves the logs directory from cfg (timestamped subdirectory when needed),
// stores it on the Runner, and exports DEVLOG_LOGS_DIR in the tmux session env.
// A failure after the session is created kills that session.
func (r *Runner) CreateSession(cfg SessionConfig) (err error) {
	if r.SessionExists() {
		return fmt.Errorf("tmux session '%s' already exists", r.sessionName)
	}

	logsDir, err := resolveRunLogsDir(cfg.LogsDir, cfg.RunMode, time.Now())
	if err != nil {
		return err
	}
	absLogsDir, err := filepath.Abs(logsDir)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for logs dir: %w", err)
	}
	r.logsDir = absLogsDir

	if err = os.MkdirAll(absLogsDir, fileutil.DirMode); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}
	if err = ensurePaneLogFiles(absLogsDir, cfg.Windows, cfg.RunMode == "overwrite"); err != nil {
		return err
	}

	if len(cfg.Windows) == 0 || len(cfg.Windows[0].Panes) == 0 {
		return fmt.Errorf("at least one window with one pane is required")
	}

	firstWindow := cfg.Windows[0]
	firstPane := firstWindow.Panes[0]

	var sessionStarted bool
	defer func() {
		if err != nil && sessionStarted {
			_ = exec.Command("tmux", "kill-session", "-t", r.sessionName).Run()
		}
	}()

	// -P prints the window id. Later commands target that id, not the name,
	// so names that contain tmux target punctuation cannot select the wrong window.
	out, err := exec.Command("tmux", "new-session", "-d", "-P", "-F", "#{window_id}", "-s", r.sessionName, "-n", firstWindow.Name).Output()
	if err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}
	sessionStarted = true
	firstWindowID := strings.TrimSpace(string(out))

	if err = exec.Command("tmux", "set-environment", "-t", r.sessionName, "DEVLOG_LOGS_DIR", absLogsDir).Run(); err != nil {
		return fmt.Errorf("failed to set logs dir env: %w", err)
	}

	if err = r.sendCommandWithLogging(firstWindowID, firstPane.Cmd, firstPane.Log); err != nil {
		return fmt.Errorf("failed to run command in first pane: %w", err)
	}

	for i := 1; i < len(firstWindow.Panes); i++ {
		pane := firstWindow.Panes[i]
		if err = r.splitWindow(firstWindowID, pane.Cmd, pane.Log); err != nil {
			return fmt.Errorf("failed to create pane %d in window %s: %w", i, firstWindow.Name, err)
		}
	}

	for i := 1; i < len(cfg.Windows); i++ {
		window := cfg.Windows[i]
		if err = r.createWindow(window); err != nil {
			return fmt.Errorf("failed to create window %s: %w", window.Name, err)
		}
	}

	return nil
}

// resolveRunLogsDir picks the directory for this run.
// Timestamped names are YYYYMMDD-HHMMSS. A second start in the same second gets a -N suffix.
func resolveRunLogsDir(base, runMode string, now time.Time) (string, error) {
	if runMode != "timestamped" {
		return base, nil
	}
	stamp := now.Format("20060102-150405")
	dir := filepath.Join(base, stamp)
	for n := 2; n < 10000; n++ {
		_, statErr := os.Stat(dir)
		if os.IsNotExist(statErr) {
			return dir, nil
		}
		if statErr != nil {
			return "", fmt.Errorf("failed to stat log directory: %w", statErr)
		}
		dir = filepath.Join(base, fmt.Sprintf("%s-%d", stamp, n))
	}
	return "", fmt.Errorf("too many log directories for timestamp %s", stamp)
}

func ensurePaneLogFiles(logsDir string, windows []config.WindowConfig, truncate bool) error {
	seen := make(map[string]struct{})
	for _, window := range windows {
		for _, pane := range window.Panes {
			if pane.Log == "" {
				continue
			}

			logPath, err := fileutil.SafeJoin(logsDir, pane.Log)
			if err != nil {
				return err
			}
			if _, ok := seen[logPath]; ok {
				continue
			}
			seen[logPath] = struct{}{}

			if err := fileutil.PrepareLogFile(logPath, truncate); err != nil {
				return fmt.Errorf("failed to create log file '%s': %w", logPath, err)
			}
		}
	}
	return nil
}

// createWindow creates a new window with its panes and returns once each pane command is sent.
func (r *Runner) createWindow(window config.WindowConfig) error {
	out, err := exec.Command("tmux", "new-window", "-P", "-F", "#{window_id}", "-t", r.sessionName, "-n", window.Name).Output()
	if err != nil {
		return fmt.Errorf("failed to create window: %w", err)
	}
	if len(window.Panes) == 0 {
		return fmt.Errorf("window %s has no panes", window.Name)
	}

	windowID := strings.TrimSpace(string(out))
	firstPane := window.Panes[0]
	if err := r.sendCommandWithLogging(windowID, firstPane.Cmd, firstPane.Log); err != nil {
		return fmt.Errorf("failed to run command in first pane: %w", err)
	}

	for i := 1; i < len(window.Panes); i++ {
		pane := window.Panes[i]
		if err := r.splitWindow(windowID, pane.Cmd, pane.Log); err != nil {
			return fmt.Errorf("failed to create pane %d: %w", i, err)
		}
	}

	return nil
}

// splitWindow splits the window horizontally, then vertically if the window is too narrow.
func (r *Runner) splitWindow(target string, command, logFile string) error {
	if err := exec.Command("tmux", "split-window", "-h", "-t", target).Run(); err != nil {
		if errV := exec.Command("tmux", "split-window", "-v", "-t", target).Run(); errV != nil {
			return fmt.Errorf("failed to split window: %w", errV)
		}
	}

	// After split-window, the new pane is active, so the window target hits that pane.
	return r.sendCommandWithLogging(target, command, logFile)
}

// sendCommandWithLogging waits for the pane process, then captures output and sends the command.
func (r *Runner) sendCommandWithLogging(target, command, logFile string) error {
	if err := waitForPaneProcess(target); err != nil {
		return err
	}

	if logFile != "" {
		logPath, err := fileutil.SafeJoin(r.logsDir, logFile)
		if err != nil {
			return err
		}

		// Quote the path to prevent command injection. Append within the run.
		// Overwrite mode truncates the file before the session starts.
		pipeCmd := fmt.Sprintf("cat >> %s", shellescape.Quote(logPath))
		cmd := exec.Command("tmux", "pipe-pane", "-t", target, "-o", pipeCmd)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set up pipe-pane logging: %w", err)
		}
	}

	// Run pane commands through POSIX sh so bash-style syntax works even when the
	// user's interactive shell is fish/zsh.
	shCommand := fmt.Sprintf("sh -lc %s", shellescape.Quote(command))
	cmd := exec.Command("tmux", "send-keys", "-t", target, shCommand, "C-m")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	return nil
}

// waitForPaneProcess returns when the pane has a current command, or after two seconds.
func waitForPaneProcess(target string) error {
	deadline := time.Now().Add(2 * time.Second)
	for {
		out, err := exec.Command("tmux", "display-message", "-p", "-t", target, "#{pane_current_command}").Output()
		if err == nil && strings.TrimSpace(string(out)) != "" {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for pane %s to start", target)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// KillSession sends SIGINT to every pane in every window, waits, then kills the session.
// kill-session is what ends processes that ignore SIGINT. A second Ctrl-C is not a force kill.
func (r *Runner) KillSession() error {
	if !r.SessionExists() {
		return fmt.Errorf("tmux session '%s' does not exist", r.sessionName)
	}

	paneIDs, paneErr := r.getPaneIDs()
	if paneErr == nil {
		signalPanes(paneIDs)
		// Wait in the Go process so the grace period is deterministic.
		time.Sleep(500 * time.Millisecond)
	}

	cmd := exec.Command("tmux", "kill-session", "-t", r.sessionName)
	if err := cmd.Run(); err != nil {
		if paneErr != nil {
			return fmt.Errorf("failed to get pane list: %w", paneErr)
		}
		return fmt.Errorf("failed to kill tmux session: %w", err)
	}
	if paneErr != nil {
		return fmt.Errorf("tmux session killed, but pane signals were not sent: %w", paneErr)
	}
	return nil
}

// signalPanes sends Ctrl-C to each pane id. Pane ids look like %7 and are already
// global targets. Prefixing the session name makes tmux read %7 as a window name.
func signalPanes(paneIDs []string) {
	for _, paneID := range paneIDs {
		_ = exec.Command("tmux", "send-keys", "-t", paneID, "C-c").Run()
	}
}

// getPaneIDs returns pane ids for every window in the session.
func (r *Runner) getPaneIDs() ([]string, error) {
	cmd := exec.Command("tmux", "list-panes", "-s", "-t", r.sessionName, "-F", "#{pane_id}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var ids []string
	for _, line := range lines {
		if line != "" {
			ids = append(ids, line)
		}
	}
	return ids, nil
}

// GetLogsDir returns the resolved logs directory for this session.
// Prefers the in-memory field set by CreateSession; falls back to the tmux
// DEVLOG_LOGS_DIR environment variable for sessions started by older versions.
func (r *Runner) GetLogsDir() string {
	if r.logsDir != "" {
		return r.logsDir
	}
	cmd := exec.Command("tmux", "show-environment", "-t", r.sessionName, "DEVLOG_LOGS_DIR")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(output))
	if _, val, ok := strings.Cut(line, "="); ok {
		return val
	}
	return ""
}

// GetSessionInfo returns information about the session
func (r *Runner) GetSessionInfo() (*SessionInfo, error) {
	if !r.SessionExists() {
		return nil, fmt.Errorf("tmux session '%s' does not exist", r.sessionName)
	}

	info := &SessionInfo{
		Name:    r.sessionName,
		Windows: []WindowInfo{},
	}

	// Use tabs as field separators — window/pane names can contain '|' but not tabs
	// in tmux -F output.
	cmd := exec.Command("tmux", "list-windows", "-t", r.sessionName, "-F", "#{window_index}\t#{window_name}\t#{window_panes}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list windows: %w", err)
	}

	for _, line := range strings.Split(string(output), "\n") {
		window, ok := parseWindowLine(line)
		if !ok {
			continue
		}

		panes, err := r.getWindowPanes(window.Index)
		if err == nil {
			window.Panes = panes
		}

		info.Windows = append(info.Windows, window)
	}

	return info, nil
}

// getWindowPanes returns information about all panes in a window
func (r *Runner) getWindowPanes(windowIndex int) ([]PaneInfo, error) {
	windowTarget := fmt.Sprintf("%s:%d", r.sessionName, windowIndex)
	cmd := exec.Command("tmux", "list-panes", "-t", windowTarget, "-F", "#{pane_id}\t#{pane_index}\t#{pane_current_command}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var panes []PaneInfo
	for _, line := range strings.Split(string(output), "\n") {
		pane, ok := parsePaneLine(line)
		if !ok {
			continue
		}
		panes = append(panes, pane)
	}

	return panes, nil
}

// parseWindowLine parses a tab-delimited tmux list-windows -F line:
// index\tname\tpane_count
func parseWindowLine(line string) (WindowInfo, bool) {
	var window WindowInfo
	if line == "" {
		return window, false
	}
	parts := strings.Split(line, "\t")
	if len(parts) != 3 {
		return window, false
	}
	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return window, false
	}
	paneCount, err := strconv.Atoi(parts[2])
	if err != nil {
		return window, false
	}
	window.Index = index
	window.Name = parts[1]
	window.PaneCount = paneCount
	return window, true
}

// parsePaneLine parses a tab-delimited tmux list-panes -F line:
// id\tindex\tcommand
func parsePaneLine(line string) (PaneInfo, bool) {
	var pane PaneInfo
	if line == "" {
		return pane, false
	}
	parts := strings.Split(line, "\t")
	if len(parts) != 3 {
		return pane, false
	}
	index, err := strconv.Atoi(parts[1])
	if err != nil {
		return pane, false
	}
	pane.ID = parts[0]
	pane.Index = index
	pane.Command = parts[2]
	return pane, true
}

// SessionInfo holds information about a tmux session
type SessionInfo struct {
	Name    string
	Windows []WindowInfo
}

// WindowInfo holds information about a tmux window
type WindowInfo struct {
	Index     int
	Name      string
	PaneCount int
	Panes     []PaneInfo
}

// PaneInfo holds information about a tmux pane
type PaneInfo struct {
	ID      string
	Index   int
	Command string
}

// CheckVersion returns the tmux version string or an error if tmux is not installed
func CheckVersion() (string, error) {
	cmd := exec.Command("tmux", "-V")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tmux not found")
	}
	return strings.TrimSpace(string(output)), nil
}
