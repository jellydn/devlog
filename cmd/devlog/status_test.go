package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jellydn/devlog/internal/config"
)

func TestResolveStatusLogsDir_PrefersRunningLogsDir(t *testing.T) {
	cfg := &config.Config{
		LogsDir: "./logs",
		RunMode: "timestamped",
	}
	got := resolveStatusLogsDir("/tmp/active-run", cfg)
	if got != "/tmp/active-run" {
		t.Errorf("resolveStatusLogsDir() = %q, want %q", got, "/tmp/active-run")
	}
}

func TestResolveStatusLogsDir_TimestampedUsesLatestRunDir(t *testing.T) {
	base := t.TempDir()
	oldDir := filepath.Join(base, "20260223-210000")
	newDir := filepath.Join(base, "20260223-220000")

	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatalf("failed to create old dir: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := os.MkdirAll(newDir, 0755); err != nil {
		t.Fatalf("failed to create new dir: %v", err)
	}

	cfg := &config.Config{
		LogsDir: base,
		RunMode: "timestamped",
	}
	got := resolveStatusLogsDir("", cfg)
	if got != newDir {
		t.Errorf("resolveStatusLogsDir() = %q, want %q", got, newDir)
	}
}

func TestResolveStatusLogsDir_NonTimestampedUsesBaseLogsDir(t *testing.T) {
	cfg := &config.Config{
		LogsDir: "/tmp/logs",
		RunMode: "overwrite",
	}
	got := resolveStatusLogsDir("", cfg)
	if got != "/tmp/logs" {
		t.Errorf("resolveStatusLogsDir() = %q, want %q", got, "/tmp/logs")
	}
}

func TestEnsureFileExists(t *testing.T) {
	target := filepath.Join(t.TempDir(), "nested", "browser.log")
	if err := ensureFileExists(target); err != nil {
		t.Fatalf("ensureFileExists() failed: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
	if info.IsDir() {
		t.Fatalf("expected file path, got directory: %s", target)
	}
}
