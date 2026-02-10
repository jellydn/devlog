package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jellydn/devlog/internal/natmsg"
)

func TestNew_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "subdir", "browser.log")

	logger, err := New(logPath, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	// Check directory was created
	if _, err := os.Stat(filepath.Dir(logPath)); os.IsNotExist(err) {
		t.Error("log directory was not created")
	}
}

func TestNew_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "browser.log")

	logger, err := New(logPath, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	// Check file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("log file was not created")
	}
}

func TestNew_AppendsToExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "browser.log")

	// Create initial file with content
	if err := os.WriteFile(logPath, []byte("existing content\n"), 0644); err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	logger, err := New(logPath, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Write a message
	msg := &natmsg.Message{
		Type:      "console",
		Level:     "log",
		Message:   "test message",
		Timestamp: 1234567890000,
	}
	if err := logger.Log(msg); err != nil {
		t.Fatalf("failed to log message: %v", err)
	}
	logger.Close()

	// Read file and verify both old and new content exist
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "existing content") {
		t.Error("existing content was not preserved")
	}
	if !strings.Contains(string(content), "test message") {
		t.Error("new message was not appended")
	}
}

func TestShouldLog_NoLevels(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "browser.log")

	logger, err := New(logPath, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	// With no levels specified, all should be logged
	if !logger.ShouldLog("log") {
		t.Error("ShouldLog('log') = false, want true")
	}
	if !logger.ShouldLog("error") {
		t.Error("ShouldLog('error') = false, want true")
	}
	if !logger.ShouldLog("warn") {
		t.Error("ShouldLog('warn') = false, want true")
	}
}

func TestShouldLog_WithLevels(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "browser.log")

	logger, err := New(logPath, []string{"error", "warn"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	// Only error and warn should be logged
	if logger.ShouldLog("log") {
		t.Error("ShouldLog('log') = true, want false")
	}
	if !logger.ShouldLog("error") {
		t.Error("ShouldLog('error') = false, want true")
	}
	if !logger.ShouldLog("warn") {
		t.Error("ShouldLog('warn') = false, want true")
	}
}

func TestShouldLog_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "browser.log")

	logger, err := New(logPath, []string{"ERROR", "Warn"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	// Should match regardless of case
	if !logger.ShouldLog("error") {
		t.Error("ShouldLog('error') = false, want true")
	}
	if !logger.ShouldLog("ERROR") {
		t.Error("ShouldLog('ERROR') = false, want true")
	}
	if !logger.ShouldLog("warn") {
		t.Error("ShouldLog('warn') = false, want true")
	}
}

func TestLog_FormatsMessage(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "browser.log")

	logger, err := New(logPath, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := &natmsg.Message{
		Type:      "console",
		Level:     "error",
		Message:   "Something went wrong",
		URL:       "http://example.com/page",
		Timestamp: 1704067200000, // 2024-01-01 00:00:00.000
		Source:    "app.js",
		Line:      42,
		Column:    10,
	}

	if err := logger.Log(msg); err != nil {
		t.Fatalf("failed to log message: %v", err)
	}
	logger.Close()

	// Read and verify format
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	expectedParts := []string{
		"[2024-01-01",               // timestamp
		"[ERROR]",                   // level
		"[http://example.com/page]", // URL
		"app.js:42:10",              // source location
		"Something went wrong",      // message
	}

	for _, part := range expectedParts {
		if !strings.Contains(string(content), part) {
			t.Errorf("log missing expected part %q: %s", part, string(content))
		}
	}
}

func TestLog_SkipsFilteredLevels(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "browser.log")

	logger, err := New(logPath, []string{"error"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Log messages at different levels
	messages := []*natmsg.Message{
		{Type: "console", Level: "log", Message: "info message", Timestamp: 1234567890000},
		{Type: "console", Level: "error", Message: "error message", Timestamp: 1234567890000},
		{Type: "console", Level: "warn", Message: "warn message", Timestamp: 1234567890000},
	}

	for _, msg := range messages {
		if err := logger.Log(msg); err != nil {
			t.Fatalf("failed to log message: %v", err)
		}
	}
	logger.Close()

	// Read and verify only error was logged
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if strings.Contains(string(content), "info message") {
		t.Error("log should not contain info message")
	}
	if !strings.Contains(string(content), "error message") {
		t.Error("log should contain error message")
	}
	if strings.Contains(string(content), "warn message") {
		t.Error("log should not contain warn message")
	}
}

func TestLog_NoSourceInfo(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "browser.log")

	logger, err := New(logPath, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := &natmsg.Message{
		Type:      "console",
		Level:     "log",
		Message:   "Simple message",
		URL:       "http://example.com",
		Timestamp: 1234567890000,
		// No Source, Line, or Column
	}

	if err := logger.Log(msg); err != nil {
		t.Fatalf("failed to log message: %v", err)
	}
	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	// Should not contain source info markers
	if strings.Contains(string(content), ":0") {
		t.Error("log should not contain line/column info when not provided")
	}
}

func TestLog_NoURL(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "browser.log")

	logger, err := New(logPath, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := &natmsg.Message{
		Type:      "console",
		Level:     "log",
		Message:   "Message without URL",
		Timestamp: 1234567890000,
		// No URL
	}

	if err := logger.Log(msg); err != nil {
		t.Fatalf("failed to log message: %v", err)
	}
	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	// Should contain message but no URL brackets
	if !strings.Contains(string(content), "Message without URL") {
		t.Error("log should contain the message")
	}
}

func TestLogPath(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "browser.log")

	logger, err := New(logPath, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer logger.Close()

	if logger.LogPath() != logPath {
		t.Errorf("LogPath() = %q, want %q", logger.LogPath(), logPath)
	}
}
