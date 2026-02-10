package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
