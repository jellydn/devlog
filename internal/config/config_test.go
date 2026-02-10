package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad_ValidConfig(t *testing.T) {
	content := `
version: "1.0"
project: myapp
logs_dir: ./logs
run_mode: timestamped
tmux:
  session: dev
  windows:
    - name: server
      panes:
        - cmd: npm run dev
          log: server.log
        - cmd: npm run worker
          log: worker.log
    - name: editor
      panes:
        - cmd: nvim
          log: editor.log
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "devlog.yml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Version != "1.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1.0")
	}
	if cfg.Project != "myapp" {
		t.Errorf("Project = %q, want %q", cfg.Project, "myapp")
	}
	if cfg.LogsDir != "./logs" {
		t.Errorf("LogsDir = %q, want %q", cfg.LogsDir, "./logs")
	}
	if cfg.RunMode != "timestamped" {
		t.Errorf("RunMode = %q, want %q", cfg.RunMode, "timestamped")
	}
	if cfg.Tmux.Session != "dev" {
		t.Errorf("Tmux.Session = %q, want %q", cfg.Tmux.Session, "dev")
	}
	if len(cfg.Tmux.Windows) != 2 {
		t.Errorf("len(Windows) = %d, want %d", len(cfg.Tmux.Windows), 2)
	}
}

func TestLoad_Defaults(t *testing.T) {
	content := `
version: "1.0"
project: test
tmux:
  session: test
  windows:
    - name: main
      panes:
        - cmd: echo hello
          log: out.log
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "devlog.yml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.LogsDir != "./logs" {
		t.Errorf("LogsDir default = %q, want %q", cfg.LogsDir, "./logs")
	}
	if cfg.RunMode != "timestamped" {
		t.Errorf("RunMode default = %q, want %q", cfg.RunMode, "timestamped")
	}
}

func TestLoad_EnvVarInterpolation(t *testing.T) {
	os.Setenv("TEST_PORT", "3000")
	defer os.Unsetenv("TEST_PORT")

	content := `
version: "1.0"
project: myapp
tmux:
  session: dev
  windows:
    - name: server
      panes:
        - cmd: npm start --port $TEST_PORT
          log: server.log
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "devlog.yml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !strings.Contains(cfg.Tmux.Windows[0].Panes[0].Cmd, "3000") {
		t.Errorf("Env var not interpolated: cmd = %q", cfg.Tmux.Windows[0].Panes[0].Cmd)
	}
}

func TestLoad_EnvVarInterpolationBraces(t *testing.T) {
	os.Setenv("APP_NAME", "myapp")
	defer os.Unsetenv("APP_NAME")

	content := `
version: "1.0"
project: ${APP_NAME}
tmux:
  session: dev
  windows:
    - name: main
      panes:
        - cmd: echo test
          log: out.log
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "devlog.yml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Project != "myapp" {
		t.Errorf("Env var with braces not interpolated: project = %q", cfg.Project)
	}
}

func TestLoad_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "missing version",
			content: `
project: test
tmux:
  session: test
  windows:
    - name: main
      panes:
        - cmd: echo test
`,
			wantErr: "version is required",
		},
		{
			name: "missing project",
			content: `
version: "1.0"
tmux:
  session: test
  windows:
    - name: main
      panes:
        - cmd: echo test
`,
			wantErr: "project is required",
		},
		{
			name: "missing session",
			content: `
version: "1.0"
project: test
tmux:
  windows:
    - name: main
      panes:
        - cmd: echo test
`,
			wantErr: "tmux.session is required",
		},
		{
			name: "no windows",
			content: `
version: "1.0"
project: test
tmux:
  session: test
  windows: []
`,
			wantErr: "tmux.windows must have at least one window",
		},
		{
			name: "no panes",
			content: `
version: "1.0"
project: test
tmux:
  session: test
  windows:
    - name: main
      panes: []
`,
			wantErr: "panes must have at least one pane",
		},
		{
			name: "missing cmd",
			content: `
version: "1.0"
project: test
tmux:
  session: test
  windows:
    - name: main
      panes:
        - log: out.log
`,
			wantErr: "cmd is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "devlog.yml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			_, err := Load(configPath)
			if err == nil {
				t.Errorf("Load() expected error containing %q, got nil", tt.wantErr)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Load() error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestLoad_InvalidRunMode(t *testing.T) {
	content := `
version: "1.0"
project: test
run_mode: invalid
tmux:
  session: test
  windows:
    - name: main
      panes:
        - cmd: echo test
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "devlog.yml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() expected error for invalid run_mode, got nil")
	}
	if !strings.Contains(err.Error(), "run_mode must be 'timestamped' or 'overwrite'") {
		t.Errorf("Load() error = %q, want error about run_mode", err.Error())
	}
}

func TestConfig_ResolveLogsDir_Timestamped(t *testing.T) {
	cfg := &Config{
		LogsDir: "./logs",
		RunMode: "timestamped",
	}

	dir := cfg.ResolveLogsDir()
	if !strings.HasPrefix(dir, "logs/") {
		t.Errorf("ResolveLogsDir() = %q, want prefix 'logs/'", dir)
	}
}

func TestConfig_ResolveLogsDir_Overwrite(t *testing.T) {
	cfg := &Config{
		LogsDir: "./logs",
		RunMode: "overwrite",
	}

	dir := cfg.ResolveLogsDir()
	if dir != "./logs" {
		t.Errorf("ResolveLogsDir() = %q, want %q", dir, "./logs")
	}
}

func TestLoad_RetentionConfig(t *testing.T) {
	content := `
version: "1.0"
project: myapp
max_runs: 10
retention_days: 30
tmux:
  session: dev
  windows:
    - name: server
      panes:
        - cmd: npm run dev
          log: server.log
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "devlog.yml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.MaxRuns != 10 {
		t.Errorf("MaxRuns = %d, want %d", cfg.MaxRuns, 10)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("RetentionDays = %d, want %d", cfg.RetentionDays, 30)
	}
}

func TestValidate_NegativeMaxRuns(t *testing.T) {
	cfg := &Config{
		Version: "1.0",
		Project: "test",
		RunMode: "timestamped",
		MaxRuns: -1,
		Tmux: TmuxConfig{
			Session: "test",
			Windows: []WindowConfig{
				{
					Name: "main",
					Panes: []PaneConfig{
						{Cmd: "echo test"},
					},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for negative max_runs, got nil")
	}
	if !strings.Contains(err.Error(), "max_runs must be non-negative") {
		t.Errorf("Validate() error = %q, want error about max_runs", err.Error())
	}
}

func TestValidate_NegativeRetentionDays(t *testing.T) {
	cfg := &Config{
		Version:       "1.0",
		Project:       "test",
		RunMode:       "timestamped",
		RetentionDays: -1,
		Tmux: TmuxConfig{
			Session: "test",
			Windows: []WindowConfig{
				{
					Name: "main",
					Panes: []PaneConfig{
						{Cmd: "echo test"},
					},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for negative retention_days, got nil")
	}
	if !strings.Contains(err.Error(), "retention_days must be non-negative") {
		t.Errorf("Validate() error = %q, want error about retention_days", err.Error())
	}
}

func TestCleanupOldRuns_MaxRuns(t *testing.T) {
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

	cfg := &Config{
		LogsDir: logsDir,
		RunMode: "timestamped",
		MaxRuns: 3,
	}

	if err := cfg.CleanupOldRuns(false); err != nil {
		t.Fatalf("CleanupOldRuns() failed: %v", err)
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

func TestCleanupOldRuns_RetentionDays(t *testing.T) {
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

	cfg := &Config{
		LogsDir:       logsDir,
		RunMode:       "timestamped",
		RetentionDays: 30,
	}

	if err := cfg.CleanupOldRuns(false); err != nil {
		t.Fatalf("CleanupOldRuns() failed: %v", err)
	}

	// Check that old directory is removed
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Errorf("Old directory still exists: %s", oldDir)
	}

	// Check that recent directory remains
	if _, err := os.Stat(recentDir); err != nil {
		t.Errorf("Recent directory was removed: %s", recentDir)
	}
}

func TestCleanupOldRuns_DryRun(t *testing.T) {
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

	cfg := &Config{
		LogsDir: logsDir,
		RunMode: "timestamped",
		MaxRuns: 3,
	}

	if err := cfg.CleanupOldRuns(true); err != nil {
		t.Fatalf("CleanupOldRuns() failed: %v", err)
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

func TestCleanupOldRuns_NoPolicy(t *testing.T) {
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

	cfg := &Config{
		LogsDir: logsDir,
		RunMode: "timestamped",
		MaxRuns: 0,
		RetentionDays: 0,
	}

	if err := cfg.CleanupOldRuns(false); err != nil {
		t.Fatalf("CleanupOldRuns() failed: %v", err)
	}

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

func TestCleanupOldRuns_OverwriteMode(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	cfg := &Config{
		LogsDir: logsDir,
		RunMode: "overwrite",
		MaxRuns: 3,
	}

	// Should return nil without error for overwrite mode
	if err := cfg.CleanupOldRuns(false); err != nil {
		t.Fatalf("CleanupOldRuns() failed: %v", err)
	}
}
