package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdInit_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	err = cmdInit(nil, nil)
	if err != nil {
		t.Fatalf("cmdInit() failed: %v", err)
	}

	// Check that devlog.yml was created
	configPath := filepath.Join(tmpDir, "devlog.yml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("devlog.yml was not created: %v", err)
	}

	// Read the content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read devlog.yml: %v", err)
	}

	// Check for required fields
	contentStr := string(content)
	requiredFields := []string{
		"version:",
		"project:",
		"logs_dir:",
		"run_mode:",
		"tmux:",
		"session:",
		"windows:",
		"panes:",
		"cmd:",
		"log:",
		"browser:",
		"urls:",
		"file:",
		"levels:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(contentStr, field) {
			t.Errorf("Generated config missing required field: %s", field)
		}
	}
}

func TestCmdInit_FileAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Create an existing devlog.yml
	configPath := filepath.Join(tmpDir, "devlog.yml")
	if err := os.WriteFile(configPath, []byte("existing"), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Try to init again
	err = cmdInit(nil, nil)
	if err == nil {
		t.Fatal("cmdInit() should have failed with existing file")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("cmdInit() error = %q, want error containing 'already exists'", err.Error())
	}
}

func TestCmdInit_MonorepoDetection(t *testing.T) {
	tests := []struct {
		name              string
		createDir         string
		wantMultiplePanes bool
	}{
		{
			name:              "packages directory",
			createDir:         "packages",
			wantMultiplePanes: true,
		},
		{
			name:              "apps directory",
			createDir:         "apps",
			wantMultiplePanes: true,
		},
		{
			name:              "services directory",
			createDir:         "services",
			wantMultiplePanes: true,
		},
		{
			name:              "no monorepo indicator",
			createDir:         "",
			wantMultiplePanes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer os.Chdir(originalDir)

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("Failed to change directory: %v", err)
			}

			// Create monorepo indicator directory if specified
			if tt.createDir != "" {
				if err := os.Mkdir(tt.createDir, 0755); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
			}

			err = cmdInit(nil, nil)
			if err != nil {
				t.Fatalf("cmdInit() failed: %v", err)
			}

			// Read the generated config
			content, err := os.ReadFile("devlog.yml")
			if err != nil {
				t.Fatalf("Failed to read devlog.yml: %v", err)
			}

			contentStr := string(content)

			// Check for monorepo-specific content (multiple panes)
			paneCount := strings.Count(contentStr, "- cmd:")
			if tt.wantMultiplePanes {
				if paneCount < 2 {
					t.Errorf("Monorepo template should have multiple panes, got %d", paneCount)
				}
				// Check for typical monorepo commands
				if !strings.Contains(contentStr, "pnpm --filter") {
					t.Error("Monorepo template should contain pnpm --filter command")
				}
			} else {
				if paneCount != 1 {
					t.Errorf("Simple template should have 1 pane, got %d", paneCount)
				}
			}
		})
	}
}

func TestGenerateTemplate_Simple(t *testing.T) {
	template := generateTemplate("test-project", false)

	// Check that it contains required fields
	required := []string{
		"version:",
		"project: test-project",
		"logs_dir:",
		"run_mode:",
		"tmux:",
		"session: test-project",
		"windows:",
		"panes:",
		"cmd:",
		"log:",
		"browser:",
	}

	for _, field := range required {
		if !strings.Contains(template, field) {
			t.Errorf("Template missing required field: %s", field)
		}
	}

	// Check it's a simple template (only one pane)
	if strings.Count(template, "- cmd:") != 1 {
		t.Error("Simple template should have exactly 1 pane")
	}
}

func TestGenerateTemplate_Monorepo(t *testing.T) {
	template := generateTemplate("test-monorepo", true)

	// Check that it contains required fields
	required := []string{
		"version:",
		"project: test-monorepo",
		"logs_dir:",
		"run_mode:",
		"tmux:",
		"session: test-monorepo",
		"windows:",
		"panes:",
		"cmd:",
		"log:",
		"browser:",
	}

	for _, field := range required {
		if !strings.Contains(template, field) {
			t.Errorf("Template missing required field: %s", field)
		}
	}

	// Check it's a monorepo template (multiple panes)
	if strings.Count(template, "- cmd:") < 2 {
		t.Error("Monorepo template should have multiple panes")
	}

	// Check for monorepo-specific content
	if !strings.Contains(template, "pnpm --filter") {
		t.Error("Monorepo template should contain pnpm --filter command")
	}
}
