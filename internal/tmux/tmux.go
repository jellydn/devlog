package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner handles tmux session operations
type Runner struct {
	sessionName string
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

// CreateSession creates a new tmux session with the given windows and panes
func (r *Runner) CreateSession(logsDir string, windows []WindowConfig) error {
	if r.SessionExists() {
		return fmt.Errorf("tmux session '%s' already exists", r.sessionName)
	}

	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	if len(windows) == 0 || len(windows[0].Panes) == 0 {
		return fmt.Errorf("at least one window with one pane is required")
	}

	firstWindow := windows[0]
	firstPane := firstWindow.Panes[0]

	cmd := exec.Command("tmux", "new-session", "-d", "-s", r.sessionName, "-n", firstWindow.Name, "-P", "-F", "#{pane_id}")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}
	firstPaneID := strings.TrimSpace(string(out))

	absLogsDir, err := filepath.Abs(logsDir)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for logs dir: %w", err)
	}
	setEnv := exec.Command("tmux", "set-environment", "-t", r.sessionName, "DEVLOG_LOGS_DIR", absLogsDir)
	if err := setEnv.Run(); err != nil {
		return fmt.Errorf("failed to set logs dir env: %w", err)
	}

	firstWindowTarget := fmt.Sprintf("%s:%s", r.sessionName, firstWindow.Name)
	if err := r.sendCommandWithLogging(firstPaneID, firstPane.Cmd, absLogsDir, firstPane.Log); err != nil {
		return fmt.Errorf("failed to run command in first pane: %w", err)
	}

	for i := 1; i < len(firstWindow.Panes); i++ {
		pane := firstWindow.Panes[i]
		if err := r.splitWindow(firstWindowTarget, pane.Cmd, absLogsDir, pane.Log); err != nil {
			return fmt.Errorf("failed to create pane %d in window %s: %w", i, firstWindow.Name, err)
		}
	}

	for i := 1; i < len(windows); i++ {
		window := windows[i]
		if err := r.createWindow(i, window, absLogsDir); err != nil {
			return fmt.Errorf("failed to create window %s: %w", window.Name, err)
		}
	}

	return nil
}

// createWindow creates a new window with its panes
func (r *Runner) createWindow(windowIndex int, window WindowConfig, logsDir string) error {
	cmd := exec.Command("tmux", "new-window", "-t", r.sessionName, "-n", window.Name, "-P", "-F", "#{pane_id}")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to create window: %w", err)
	}
	firstPaneID := strings.TrimSpace(string(out))

	firstPane := window.Panes[0]
	windowTarget := fmt.Sprintf("%s:%s", r.sessionName, window.Name)
	if err := r.sendCommandWithLogging(firstPaneID, firstPane.Cmd, logsDir, firstPane.Log); err != nil {
		return fmt.Errorf("failed to run command in first pane: %w", err)
	}

	for i := 1; i < len(window.Panes); i++ {
		pane := window.Panes[i]
		if err := r.splitWindow(windowTarget, pane.Cmd, logsDir, pane.Log); err != nil {
			return fmt.Errorf("failed to create pane %d: %w", i, err)
		}
	}

	return nil
}

// splitWindow splits the current window and runs a command with logging
func (r *Runner) splitWindow(target string, command, logsDir, logFile string) error {
	cmd := exec.Command("tmux", "split-window", "-h", "-t", target, "-P", "-F", "#{pane_id}")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to split window: %w", err)
	}

	paneID := strings.TrimSpace(string(out))
	if err := r.sendCommandWithLogging(paneID, command, logsDir, logFile); err != nil {
		return err
	}

	return nil
}

// sendCommandWithLogging sends a command to a pane with output captured via pipe-pane
func (r *Runner) sendCommandWithLogging(target, command, logsDir, logFile string) error {
	if logFile != "" {
		logPath := filepath.Join(logsDir, logFile)

		if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
			return fmt.Errorf("failed to create log subdirectory: %w", err)
		}

		// Quote the path to prevent command injection
		escapedPath := "'" + strings.ReplaceAll(logPath, "'", "'\\''") + "'"
		pipeCmd := fmt.Sprintf("cat >> %s", escapedPath)
		cmd := exec.Command("tmux", "pipe-pane", "-t", target, "-o", pipeCmd)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set up pipe-pane logging: %w", err)
		}
	}

	cmd := exec.Command("tmux", "send-keys", "-t", target, command, "C-m")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	return nil
}

// KillSession gracefully terminates all panes and kills the tmux session
func (r *Runner) KillSession() error {
	if !r.SessionExists() {
		return fmt.Errorf("tmux session '%s' does not exist", r.sessionName)
	}

	paneIDs, err := r.getPaneIDs()
	if err != nil {
		return fmt.Errorf("failed to get pane list: %w", err)
	}

	// Send Ctrl+C to all panes to gracefully terminate processes
	for _, paneID := range paneIDs {
		target := fmt.Sprintf("%s:%s", r.sessionName, paneID)
		cmd := exec.Command("tmux", "send-keys", "-t", target, "C-c")
		cmd.Run() // Ignore errors - pane might not have a process
	}

	// Wait briefly for processes to terminate
	exec.Command("tmux", "send-keys", "-t", r.sessionName, "sleep 0.5", "C-m").Run()

	// Force kill any remaining processes with C-c again
	for _, paneID := range paneIDs {
		target := fmt.Sprintf("%s:%s", r.sessionName, paneID)
		cmd := exec.Command("tmux", "send-keys", "-t", target, "C-c")
		cmd.Run()
	}

	// Kill the session (this will close all panes and flush logs)
	cmd := exec.Command("tmux", "kill-session", "-t", r.sessionName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to kill tmux session: %w", err)
	}

	return nil
}

// getPaneIDs returns all pane IDs in the session
func (r *Runner) getPaneIDs() ([]string, error) {
	cmd := exec.Command("tmux", "list-panes", "-t", r.sessionName, "-F", "#{pane_id}")
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

// GetLogsDir retrieves the logs directory stored in the tmux session environment
func (r *Runner) GetLogsDir() string {
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

	cmd := exec.Command("tmux", "list-windows", "-t", r.sessionName, "-F", "#{window_index}|#{window_name}|#{window_panes}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list windows: %w", err)
	}

	// Parse window list (format: index|name|pane_count)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			continue
		}
		var window WindowInfo
		fmt.Sscanf(parts[0], "%d", &window.Index)
		window.Name = parts[1]
		fmt.Sscanf(parts[2], "%d", &window.PaneCount)

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
	cmd := exec.Command("tmux", "list-panes", "-t", windowTarget, "-F", "#{pane_id}|#{pane_index}|#{pane_current_command}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var panes []PaneInfo
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			continue
		}
		var pane PaneInfo
		pane.ID = parts[0]
		fmt.Sscanf(parts[1], "%d", &pane.Index)
		pane.Command = parts[2]
		panes = append(panes, pane)
	}

	return panes, nil
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

// WindowConfig represents a tmux window configuration
type WindowConfig struct {
	Name  string
	Panes []PaneConfig
}

// PaneConfig represents a tmux pane configuration
type PaneConfig struct {
	Cmd string
	Log string
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
