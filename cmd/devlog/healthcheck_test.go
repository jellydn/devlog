package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdHealthcheck_Basic(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Run healthcheck command
	err = cmdHealthcheck(nil, nil)

	// Healthcheck may fail if tmux or devlog-host is not installed,
	// but it should not panic or crash
	if err != nil && !strings.Contains(err.Error(), "healthcheck failed") {
		t.Errorf("Unexpected error from cmdHealthcheck: %v", err)
	}
}

func TestCmdHealthcheck_WithArgs(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Run healthcheck command with args (should be ignored)
	err = cmdHealthcheck(nil, []string{"--some-arg"})

	// Healthcheck should work with or without args
	if err != nil && !strings.Contains(err.Error(), "healthcheck failed") {
		t.Errorf("Unexpected error from cmdHealthcheck with args: %v", err)
	}
}

func TestHealthcheckCommandInMap(t *testing.T) {
	// Verify that healthcheck command is registered in the commands map
	cmd, ok := commands["healthcheck"]
	if !ok {
		t.Fatal("healthcheck command not found in commands map")
	}

	// Verify it's a valid function
	if cmd == nil {
		t.Fatal("healthcheck command is nil")
	}
}

func TestHealthcheckDoesNotRequireConfig(t *testing.T) {
	// Verify that healthcheck can be called without config
	// by testing in a directory without devlog.yml
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Ensure no devlog.yml exists
	if _, err := os.Stat(filepath.Join(tmpDir, "devlog.yml")); err == nil {
		t.Fatal("devlog.yml should not exist in temp directory")
	}

	// Run healthcheck - should not fail due to missing config
	err = cmdHealthcheck(nil, nil)

	// It may fail healthcheck, but should not fail due to missing config
	if err != nil && !strings.Contains(err.Error(), "healthcheck failed") {
		t.Errorf("cmdHealthcheck should only fail with 'healthcheck failed', got: %v", err)
	}
}
