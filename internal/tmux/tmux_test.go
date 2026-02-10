package tmux

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestRunner_SessionExists(t *testing.T) {
	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available in PATH")
	}

	runner := NewRunner("test-exists-session")

	// Should not exist initially
	if runner.SessionExists() {
		t.Error("SessionExists() = true, want false for non-existent session")
	}

	// Create a test session
	cmd := exec.Command("tmux", "new-session", "-d", "-s", "test-exists-session")
	if err := cmd.Run(); err != nil {
		t.Skipf("Could not create tmux session: %v", err)
	}
	defer exec.Command("tmux", "kill-session", "-t", "test-exists-session").Run()

	// Should exist now
	if !runner.SessionExists() {
		t.Error("SessionExists() = false, want true for existing session")
	}
}

func TestRunner_CreateSession_AlreadyExists(t *testing.T) {
	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available in PATH")
	}

	runner := NewRunner("test-duplicate-session")

	// Create a test session
	cmd := exec.Command("tmux", "new-session", "-d", "-s", "test-duplicate-session")
	if err := cmd.Run(); err != nil {
		t.Skipf("Could not create tmux session: %v", err)
	}
	defer exec.Command("tmux", "kill-session", "-t", "test-duplicate-session").Run()

	// Try to create again - should fail
	err := runner.CreateSession("/tmp/testlogs", []WindowConfig{
		{
			Name: "main",
			Panes: []PaneConfig{
				{Cmd: "echo test", Log: "test.log"},
			},
		},
	})

	if err == nil {
		t.Error("CreateSession() expected error for existing session, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("CreateSession() error = %q, want error containing 'already exists'", err.Error())
	}
}

func TestRunner_KillSession_NotExists(t *testing.T) {
	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available in PATH")
	}

	runner := NewRunner("test-nonexistent-session")

	err := runner.KillSession()
	if err == nil {
		t.Error("KillSession() expected error for non-existent session, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("KillSession() error = %q, want error containing 'does not exist'", err.Error())
	}
}

func TestRunner_CreateAndKillSession(t *testing.T) {
	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available in PATH")
	}

	sessionName := "test-create-kill-session"
	runner := NewRunner(sessionName)

	// Clean up any existing session
	exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	logsDir := t.TempDir()
	windows := []WindowConfig{
		{
			Name: "main",
			Panes: []PaneConfig{
				{Cmd: "echo 'pane 1'", Log: "pane1.log"},
				{Cmd: "echo 'pane 2'", Log: "pane2.log"},
			},
		},
		{
			Name: "secondary",
			Panes: []PaneConfig{
				{Cmd: "echo 'pane 3'", Log: "pane3.log"},
			},
		},
	}

	// Create session
	if err := runner.CreateSession(logsDir, windows); err != nil {
		t.Fatalf("CreateSession() failed: %v", err)
	}

	// Verify session exists
	if !runner.SessionExists() {
		t.Error("Session should exist after creation")
	}

	// Get session info
	info, err := runner.GetSessionInfo()
	if err != nil {
		t.Fatalf("GetSessionInfo() failed: %v", err)
	}

	if info.Name != sessionName {
		t.Errorf("SessionInfo.Name = %q, want %q", info.Name, sessionName)
	}

	if len(info.Windows) != 2 {
		t.Errorf("len(Windows) = %d, want 2", len(info.Windows))
	}

	// Check window names
	foundMain := false
	foundSecondary := false
	for _, w := range info.Windows {
		if w.Name == "main" {
			foundMain = true
			if w.PaneCount != 2 {
				t.Errorf("Window 'main' has %d panes, want 2", w.PaneCount)
			}
		}
		if w.Name == "secondary" {
			foundSecondary = true
			if w.PaneCount != 1 {
				t.Errorf("Window 'secondary' has %d panes, want 1", w.PaneCount)
			}
		}
	}
	if !foundMain {
		t.Error("Window 'main' not found")
	}
	if !foundSecondary {
		t.Error("Window 'secondary' not found")
	}

	// Kill session
	if err := runner.KillSession(); err != nil {
		t.Fatalf("KillSession() failed: %v", err)
	}

	// Verify session no longer exists
	if runner.SessionExists() {
		t.Error("Session should not exist after killing")
	}
}

func TestRunner_GetSessionInfo_NotExists(t *testing.T) {
	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available in PATH")
	}

	runner := NewRunner("test-getinfo-nonexistent")

	_, err := runner.GetSessionInfo()
	if err == nil {
		t.Error("GetSessionInfo() expected error for non-existent session, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("GetSessionInfo() error = %q, want error containing 'does not exist'", err.Error())
	}
}

func TestRunner_CreateSession_NoWindows(t *testing.T) {
	runner := NewRunner("test-no-windows")
	logsDir := t.TempDir()

	err := runner.CreateSession(logsDir, []WindowConfig{})
	if err == nil {
		t.Error("CreateSession() expected error for empty windows, got nil")
	}
	if !strings.Contains(err.Error(), "at least one window") {
		t.Errorf("CreateSession() error = %q, want error containing 'at least one window'", err.Error())
	}
}

func TestRunner_CreateSession_CreatesLogsDir(t *testing.T) {
	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available in PATH")
	}

	sessionName := "test-logs-dir-creation"
	runner := NewRunner(sessionName)

	// Clean up any existing session
	exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	logsDir := t.TempDir() + "/nested/logs/dir"
	windows := []WindowConfig{
		{
			Name: "main",
			Panes: []PaneConfig{
				{Cmd: "echo test", Log: "test.log"},
			},
		},
	}

	// Create session - should create logs directory
	if err := runner.CreateSession(logsDir, windows); err != nil {
		t.Fatalf("CreateSession() failed: %v", err)
	}

	// Verify logs directory was created
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		t.Error("Logs directory was not created")
	}

	// Clean up
	runner.KillSession()
}
