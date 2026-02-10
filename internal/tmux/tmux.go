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

	// Create logs directory
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create the first window with the first pane
	if len(windows) == 0 || len(windows[0].Panes) == 0 {
		return fmt.Errorf("at least one window with one pane is required")
	}

	firstWindow := windows[0]
	firstPane := firstWindow.Panes[0]

	// Create session with first window and pane
	cmd := exec.Command("tmux", "new-session", "-d", "-s", r.sessionName, "-n", firstWindow.Name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Send command to first pane with logging
	// Use window name in target - tmux will target the active pane in that window
	firstWindowTarget := fmt.Sprintf("%s:%s", r.sessionName, firstWindow.Name)
	if err := r.sendCommandWithLogging(firstWindowTarget, firstPane.Cmd, logsDir, firstPane.Log); err != nil {
		return fmt.Errorf("failed to run command in first pane: %w", err)
	}

	// Create additional panes in the first window
	for i := 1; i < len(firstWindow.Panes); i++ {
		pane := firstWindow.Panes[i]
		if err := r.splitWindow(firstWindowTarget, pane.Cmd, logsDir, pane.Log); err != nil {
			return fmt.Errorf("failed to create pane %d in window %s: %w", i, firstWindow.Name, err)
		}
	}

	// Create additional windows
	for i := 1; i < len(windows); i++ {
		window := windows[i]
		if err := r.createWindow(i, window, logsDir); err != nil {
			return fmt.Errorf("failed to create window %s: %w", window.Name, err)
		}
	}

	return nil
}

// createWindow creates a new window with its panes
func (r *Runner) createWindow(windowIndex int, window WindowConfig, logsDir string) error {
	// Create new window
	cmd := exec.Command("tmux", "new-window", "-t", r.sessionName, "-n", window.Name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create window: %w", err)
	}

	// Send command to first pane with logging
	// Target the window by name - tmux will use the active pane
	firstPane := window.Panes[0]
	windowTarget := fmt.Sprintf("%s:%s", r.sessionName, window.Name)
	if err := r.sendCommandWithLogging(windowTarget, firstPane.Cmd, logsDir, firstPane.Log); err != nil {
		return fmt.Errorf("failed to run command in first pane: %w", err)
	}

	// Create additional panes
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
	// Split window horizontally
	cmd := exec.Command("tmux", "split-window", "-h", "-t", target)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to split window: %w", err)
	}

	// Get the last pane index to target the newly created pane
	// For simplicity, we just send to the window and tmux will use the active pane
	// Actually, after split-window, the new pane is active, so we can send to the window
	if err := r.sendCommandWithLogging(target, command, logsDir, logFile); err != nil {
		return err
	}

	return nil
}

// sendCommandWithLogging sends a command to a pane with output redirected to log file
func (r *Runner) sendCommandWithLogging(target, command, logsDir, logFile string) error {
	// Create log file path
	logPath := filepath.Join(logsDir, logFile)

	// Create the log file if it doesn't exist (so tee can write to it)
	if logFile != "" {
		f, err := os.Create(logPath)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}
		f.Close()
	}

	// Send keys to start script that runs command and logs output
	// We use a subshell to capture both stdout and stderr
	script := fmt.Sprintf("(%s) 2>&1 | tee -a %s", command, logPath)
	cmd := exec.Command("tmux", "send-keys", "-t", target, script, "C-m")
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

	// Step 1: Get all pane IDs in the session
	paneIDs, err := r.getPaneIDs()
	if err != nil {
		return fmt.Errorf("failed to get pane list: %w", err)
	}

	// Step 2: Send Ctrl+C to all panes to gracefully terminate processes
	for _, paneID := range paneIDs {
		target := fmt.Sprintf("%s:%s", r.sessionName, paneID)
		cmd := exec.Command("tmux", "send-keys", "-t", target, "C-c")
		cmd.Run() // Ignore errors - pane might not have a process
	}

	// Step 3: Wait briefly for processes to terminate
	// Use tmux's built-in mechanism to wait a bit
	exec.Command("tmux", "send-keys", "-t", r.sessionName, "sleep 0.5", "C-m").Run()

	// Step 4: Force kill any remaining processes with C-c again
	for _, paneID := range paneIDs {
		target := fmt.Sprintf("%s:%s", r.sessionName, paneID)
		cmd := exec.Command("tmux", "send-keys", "-t", target, "C-c")
		cmd.Run()
	}

	// Step 5: Kill the session (this will close all panes and flush logs)
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

// GetSessionInfo returns information about the session
func (r *Runner) GetSessionInfo() (*SessionInfo, error) {
	if !r.SessionExists() {
		return nil, fmt.Errorf("tmux session '%s' does not exist", r.sessionName)
	}

	info := &SessionInfo{
		Name:    r.sessionName,
		Windows: []WindowInfo{},
	}

	// Get list of windows
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
		info.Windows = append(info.Windows, window)
	}

	return info, nil
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
