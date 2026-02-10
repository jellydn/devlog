//go:build integration

package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// skipIfNoTmux skips the test if tmux is not available
func skipIfNoTmux(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available in PATH")
	}
}

// generateTestSessionName creates a unique session name for test isolation
func generateTestSessionName() string {
	return fmt.Sprintf("devlog-test-%d", time.Now().UnixNano())
}

// TestTmuxIntegration_CreateSession tests creating a session with windows and panes
func TestTmuxIntegration_CreateSession(t *testing.T) {
	skipIfNoTmux(t)

	// Use unique session name for test isolation
	sessionName := generateTestSessionName()
	runner := NewRunner(sessionName)

	// Ensure cleanup
	defer func() {
		if runner.SessionExists() {
			runner.KillSession()
		}
	}()

	// Test session creation
	windows := []WindowConfig{
		{
			Name: "test",
			Panes: []PaneConfig{
				{Cmd: "echo 'test'", Log: "test.log"},
			},
		},
	}

	tmpDir := t.TempDir()
	if err := runner.CreateSession(tmpDir, windows); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Verify session exists
	if !runner.SessionExists() {
		t.Error("session should exist after creation")
	}

	// Wait a bit for log file to be created
	time.Sleep(100 * time.Millisecond)

	// Verify log file was created
	logPath := filepath.Join(tmpDir, "test.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("log file was not created: %s", logPath)
	}
}

// TestTmuxIntegration_MultipleWindowsAndPanes tests creating a session with multiple windows and panes
func TestTmuxIntegration_MultipleWindowsAndPanes(t *testing.T) {
	skipIfNoTmux(t)

	sessionName := generateTestSessionName()
	runner := NewRunner(sessionName)

	defer func() {
		if runner.SessionExists() {
			runner.KillSession()
		}
	}()

	tmpDir := t.TempDir()
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
				{Cmd: "echo 'pane 4'", Log: "pane4.log"},
			},
		},
		{
			Name: "tertiary",
			Panes: []PaneConfig{
				{Cmd: "echo 'pane 5'", Log: "pane5.log"},
			},
		},
	}

	if err := runner.CreateSession(tmpDir, windows); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Verify session structure
	info, err := runner.GetSessionInfo()
	if err != nil {
		t.Fatalf("GetSessionInfo failed: %v", err)
	}

	if info.Name != sessionName {
		t.Errorf("SessionInfo.Name = %q, want %q", info.Name, sessionName)
	}

	if len(info.Windows) != 3 {
		t.Errorf("len(Windows) = %d, want 3", len(info.Windows))
	}

	// Verify each window
	expectedWindows := map[string]int{
		"main":      2,
		"secondary": 2,
		"tertiary":  1,
	}

	for _, window := range info.Windows {
		expectedPanes, ok := expectedWindows[window.Name]
		if !ok {
			t.Errorf("unexpected window name: %q", window.Name)
			continue
		}
		if window.PaneCount != expectedPanes {
			t.Errorf("window %q has %d panes, want %d", window.Name, window.PaneCount, expectedPanes)
		}
	}

	// Wait for logs to be written
	time.Sleep(200 * time.Millisecond)

	// Verify all log files were created
	for _, window := range windows {
		for _, pane := range window.Panes {
			logPath := filepath.Join(tmpDir, pane.Log)
			if _, err := os.Stat(logPath); os.IsNotExist(err) {
				t.Errorf("log file was not created: %s", logPath)
			}
		}
	}
}

// TestTmuxIntegration_LogCapture tests that pipe-pane correctly captures output
func TestTmuxIntegration_LogCapture(t *testing.T) {
	skipIfNoTmux(t)

	sessionName := generateTestSessionName()
	runner := NewRunner(sessionName)

	defer func() {
		if runner.SessionExists() {
			runner.KillSession()
		}
	}()

	tmpDir := t.TempDir()
	testMessage := "integration-test-message"
	windows := []WindowConfig{
		{
			Name: "test",
			Panes: []PaneConfig{
				{Cmd: fmt.Sprintf("echo %s", testMessage), Log: "capture.log"},
			},
		},
	}

	if err := runner.CreateSession(tmpDir, windows); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Wait for command to execute and log to be written
	time.Sleep(300 * time.Millisecond)

	// Read log file
	logPath := filepath.Join(tmpDir, "capture.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	// Verify the message was logged
	if !strings.Contains(string(content), testMessage) {
		t.Errorf("log file does not contain expected message %q, got: %q", testMessage, string(content))
	}
}

// TestTmuxIntegration_SessionLifecycle tests the full lifecycle of a session
func TestTmuxIntegration_SessionLifecycle(t *testing.T) {
	skipIfNoTmux(t)

	sessionName := generateTestSessionName()
	runner := NewRunner(sessionName)

	// Session should not exist initially
	if runner.SessionExists() {
		t.Error("session should not exist initially")
	}

	tmpDir := t.TempDir()
	windows := []WindowConfig{
		{
			Name: "test",
			Panes: []PaneConfig{
				{Cmd: "echo test", Log: "test.log"},
			},
		},
	}

	// Create session
	if err := runner.CreateSession(tmpDir, windows); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Session should exist now
	if !runner.SessionExists() {
		t.Error("session should exist after creation")
	}

	// Get session info
	info, err := runner.GetSessionInfo()
	if err != nil {
		t.Fatalf("GetSessionInfo failed: %v", err)
	}

	if info.Name != sessionName {
		t.Errorf("SessionInfo.Name = %q, want %q", info.Name, sessionName)
	}

	// Kill session
	if err := runner.KillSession(); err != nil {
		t.Fatalf("KillSession failed: %v", err)
	}

	// Session should not exist after killing
	if runner.SessionExists() {
		t.Error("session should not exist after killing")
	}

	// GetSessionInfo should fail after killing
	_, err = runner.GetSessionInfo()
	if err == nil {
		t.Error("GetSessionInfo should fail for non-existent session")
	}
}

// TestTmuxIntegration_DuplicateSession tests error handling for duplicate session names
func TestTmuxIntegration_DuplicateSession(t *testing.T) {
	skipIfNoTmux(t)

	sessionName := generateTestSessionName()
	runner := NewRunner(sessionName)

	defer func() {
		if runner.SessionExists() {
			runner.KillSession()
		}
	}()

	tmpDir := t.TempDir()
	windows := []WindowConfig{
		{
			Name: "test",
			Panes: []PaneConfig{
				{Cmd: "echo test", Log: "test.log"},
			},
		},
	}

	// Create session
	if err := runner.CreateSession(tmpDir, windows); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Try to create again - should fail
	err := runner.CreateSession(tmpDir, windows)
	if err == nil {
		t.Error("CreateSession should fail for duplicate session name")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should contain 'already exists', got: %v", err)
	}
}

// TestTmuxIntegration_SpecialCharactersInPaths tests handling of special characters in log paths
func TestTmuxIntegration_SpecialCharactersInPaths(t *testing.T) {
	skipIfNoTmux(t)

	sessionName := generateTestSessionName()
	runner := NewRunner(sessionName)

	defer func() {
		if runner.SessionExists() {
			runner.KillSession()
		}
	}()

	tmpDir := t.TempDir()

	// Test with special characters in log file names
	testCases := []struct {
		name    string
		logFile string
	}{
		{"spaces", "test log file.log"},
		{"dash", "test-log-file.log"},
		{"underscore", "test_log_file.log"},
		{"dots", "test.log.file.log"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			windows := []WindowConfig{
				{
					Name: "test",
					Panes: []PaneConfig{
						{Cmd: "echo test", Log: tc.logFile},
					},
				},
			}

			// Create unique session for each test case
			uniqueSession := fmt.Sprintf("%s-%s", sessionName, tc.name)
			testRunner := NewRunner(uniqueSession)

			defer func() {
				if testRunner.SessionExists() {
					testRunner.KillSession()
				}
			}()

			if err := testRunner.CreateSession(tmpDir, windows); err != nil {
				t.Fatalf("CreateSession failed for %s: %v", tc.name, err)
			}

			// Wait for log file
			time.Sleep(100 * time.Millisecond)

			logPath := filepath.Join(tmpDir, tc.logFile)
			if _, err := os.Stat(logPath); os.IsNotExist(err) {
				t.Errorf("log file was not created: %s", logPath)
			}
		})
	}
}

// TestTmuxIntegration_EmptyLogFile tests panes without log files
func TestTmuxIntegration_EmptyLogFile(t *testing.T) {
	skipIfNoTmux(t)

	sessionName := generateTestSessionName()
	runner := NewRunner(sessionName)

	defer func() {
		if runner.SessionExists() {
			runner.KillSession()
		}
	}()

	tmpDir := t.TempDir()
	windows := []WindowConfig{
		{
			Name: "test",
			Panes: []PaneConfig{
				{Cmd: "echo 'logged'", Log: "with-log.log"},
				{Cmd: "echo 'not logged'", Log: ""}, // No log file
			},
		},
	}

	if err := runner.CreateSession(tmpDir, windows); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Wait for log file
	time.Sleep(100 * time.Millisecond)

	// Verify only the first log file exists
	logPath := filepath.Join(tmpDir, "with-log.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("log file should have been created for first pane")
	}

	// Session should still work fine
	if !runner.SessionExists() {
		t.Error("session should exist")
	}
}

// TestTmuxIntegration_GetLogsDir tests retrieving the logs directory from session environment
func TestTmuxIntegration_GetLogsDir(t *testing.T) {
	skipIfNoTmux(t)

	sessionName := generateTestSessionName()
	runner := NewRunner(sessionName)

	defer func() {
		if runner.SessionExists() {
			runner.KillSession()
		}
	}()

	tmpDir := t.TempDir()
	windows := []WindowConfig{
		{
			Name: "test",
			Panes: []PaneConfig{
				{Cmd: "echo test", Log: "test.log"},
			},
		},
	}

	if err := runner.CreateSession(tmpDir, windows); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Get logs directory from session
	logsDir := runner.GetLogsDir()
	if logsDir == "" {
		t.Error("GetLogsDir should return non-empty string")
	}

	// Should be an absolute path
	if !filepath.IsAbs(logsDir) {
		t.Errorf("GetLogsDir should return absolute path, got: %s", logsDir)
	}

	// Should match or be equivalent to tmpDir
	absTmpDir, _ := filepath.Abs(tmpDir)
	if logsDir != absTmpDir {
		t.Errorf("GetLogsDir = %q, want %q", logsDir, absTmpDir)
	}
}

// TestTmuxIntegration_MultiplePanesSameLogFile tests multiple panes logging to the same file
func TestTmuxIntegration_MultiplePanesSameLogFile(t *testing.T) {
	skipIfNoTmux(t)

	sessionName := generateTestSessionName()
	runner := NewRunner(sessionName)

	defer func() {
		if runner.SessionExists() {
			runner.KillSession()
		}
	}()

	tmpDir := t.TempDir()
	sharedLog := "shared.log"
	windows := []WindowConfig{
		{
			Name: "test",
			Panes: []PaneConfig{
				{Cmd: "echo 'message-from-pane-1'", Log: sharedLog},
				{Cmd: "echo 'message-from-pane-2'", Log: sharedLog},
			},
		},
	}

	if err := runner.CreateSession(tmpDir, windows); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Wait for logs to be written
	time.Sleep(300 * time.Millisecond)

	// Read shared log file
	logPath := filepath.Join(tmpDir, sharedLog)
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	// Verify both messages are in the log
	contentStr := string(content)
	if !strings.Contains(contentStr, "message-from-pane-1") {
		t.Error("log should contain message from pane 1")
	}
	if !strings.Contains(contentStr, "message-from-pane-2") {
		t.Error("log should contain message from pane 2")
	}
}

// TestTmuxIntegration_LongSessionName tests handling of very long session names
func TestTmuxIntegration_LongSessionName(t *testing.T) {
	skipIfNoTmux(t)

	// Create a long but valid session name (tmux has limits)
	sessionName := fmt.Sprintf("devlog-test-very-long-session-name-%d", time.Now().UnixNano())
	runner := NewRunner(sessionName)

	defer func() {
		if runner.SessionExists() {
			runner.KillSession()
		}
	}()

	tmpDir := t.TempDir()
	windows := []WindowConfig{
		{
			Name: "test",
			Panes: []PaneConfig{
				{Cmd: "echo test", Log: "test.log"},
			},
		},
	}

	if err := runner.CreateSession(tmpDir, windows); err != nil {
		t.Fatalf("CreateSession failed with long session name: %v", err)
	}

	if !runner.SessionExists() {
		t.Error("session should exist after creation")
	}

	info, err := runner.GetSessionInfo()
	if err != nil {
		t.Fatalf("GetSessionInfo failed: %v", err)
	}

	if info.Name != sessionName {
		t.Errorf("SessionInfo.Name = %q, want %q", info.Name, sessionName)
	}
}

// TestTmuxIntegration_PaneCommands tests that different commands execute correctly
func TestTmuxIntegration_PaneCommands(t *testing.T) {
	skipIfNoTmux(t)

	sessionName := generateTestSessionName()
	runner := NewRunner(sessionName)

	defer func() {
		if runner.SessionExists() {
			runner.KillSession()
		}
	}()

	tmpDir := t.TempDir()
	windows := []WindowConfig{
		{
			Name: "test",
			Panes: []PaneConfig{
				{Cmd: "echo hello", Log: "echo.log"},
				{Cmd: "date", Log: "date.log"},
				{Cmd: "pwd", Log: "pwd.log"},
			},
		},
	}

	if err := runner.CreateSession(tmpDir, windows); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Wait for commands to execute
	time.Sleep(300 * time.Millisecond)

	// Verify each log file has content
	for _, pane := range windows[0].Panes {
		logPath := filepath.Join(tmpDir, pane.Log)
		info, err := os.Stat(logPath)
		if err != nil {
			t.Errorf("log file not found: %s", logPath)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("log file is empty: %s", logPath)
		}
	}
}
