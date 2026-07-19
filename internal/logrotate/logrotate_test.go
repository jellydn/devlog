package logrotate

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanup_MaxRuns(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatalf("Failed to create logs dir: %v", err)
	}

	// Create 5 log directories with different timestamps
	dirs := []string{
		"20240101-120000",
		"20240102-120000",
		"20240103-120000",
		"20240104-120000",
		"20240105-120000",
	}
	for _, dir := range dirs {
		dirPath := filepath.Join(logsDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create test dir %s: %v", dir, err)
		}
	}

	result, err := Cleanup(logsDir, Policy{MaxRuns: 3}, false)
	if err != nil {
		t.Fatalf("Cleanup() failed: %v", err)
	}
	if len(result.Removed) != 2 {
		t.Errorf("Removed = %v, want 2", result.Removed)
	}
	if len(result.Failed) != 0 {
		t.Errorf("Failed = %v, want empty", result.Failed)
	}

	// Check that only 3 directories remain
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		t.Fatalf("Failed to read logs dir: %v", err)
	}

	var remainingDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			remainingDirs = append(remainingDirs, entry.Name())
		}
	}

	if len(remainingDirs) != 3 {
		t.Errorf("After cleanup, got %d directories, want 3. Remaining: %v", len(remainingDirs), remainingDirs)
	}
}

func TestCleanup_RetentionDays(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatalf("Failed to create logs dir: %v", err)
	}

	// Create directories and set modification times
	oldDir := filepath.Join(logsDir, "20240101-120000")
	recentDir := filepath.Join(logsDir, "20240201-120000")

	for _, dir := range []string{oldDir, recentDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create test dir: %v", err)
		}
	}

	// Set old directory to be 40 days old
	oldTime := time.Now().AddDate(0, 0, -40)
	if err := os.Chtimes(oldDir, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set old time: %v", err)
	}

	result, err := Cleanup(logsDir, Policy{RetentionDays: 30}, false)
	if err != nil {
		t.Fatalf("Cleanup() failed: %v", err)
	}
	_ = result

	// Check that old directory is removed
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Errorf("Old directory still exists: %s", oldDir)
	}

	// Check that recent directory remains
	if _, err := os.Stat(recentDir); err != nil {
		t.Errorf("Recent directory was removed: %s", recentDir)
	}
}

func TestCleanup_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatalf("Failed to create logs dir: %v", err)
	}

	// Create 5 log directories
	for i := 1; i <= 5; i++ {
		dir := filepath.Join(logsDir, fmt.Sprintf("2024010%d-120000", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create test dir: %v", err)
		}
	}

	result, err := Cleanup(logsDir, Policy{MaxRuns: 3}, true)
	if err != nil {
		t.Fatalf("Cleanup() failed: %v", err)
	}
	if len(result.Removed) != 2 {
		t.Errorf("DryRun Removed = %v, want 2 candidates", result.Removed)
	}

	// Check that all 5 directories still exist (dry run)
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		t.Fatalf("Failed to read logs dir: %v", err)
	}

	var remainingDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			remainingDirs = append(remainingDirs, entry.Name())
		}
	}

	if len(remainingDirs) != 5 {
		t.Errorf("After dry run cleanup, got %d directories, want 5. Remaining: %v", len(remainingDirs), remainingDirs)
	}
}

func TestCleanup_NoPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatalf("Failed to create logs dir: %v", err)
	}

	// Create some directories
	for i := 1; i <= 5; i++ {
		dir := filepath.Join(logsDir, fmt.Sprintf("2024010%d-120000", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create test dir: %v", err)
		}
	}

	result, err := Cleanup(logsDir, Policy{MaxRuns: 0, RetentionDays: 0}, false)
	if err != nil {
		t.Fatalf("Cleanup() failed: %v", err)
	}
	_ = result

	// Check that all directories still exist (no policy)
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		t.Fatalf("Failed to read logs dir: %v", err)
	}

	var remainingDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			remainingDirs = append(remainingDirs, entry.Name())
		}
	}

	if len(remainingDirs) != 5 {
		t.Errorf("With no policy, got %d directories, want 5", len(remainingDirs))
	}
}

func TestCleanup_MissingLogsDir(t *testing.T) {
	result, err := Cleanup(filepath.Join(t.TempDir(), "missing"), Policy{MaxRuns: 3}, false)
	if err != nil {
		t.Fatalf("Cleanup() failed: %v", err)
	}
	if len(result.Removed) != 0 {
		t.Errorf("Removed = %v, want empty", result.Removed)
	}
}
