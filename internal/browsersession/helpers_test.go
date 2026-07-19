package browsersession

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSanitizeSessionForFileName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", "default"},
		{"my-app", "my-app"},
		{"My App!", "My-App"},
		{"---", "default"},
		{"a/b:c", "a-b-c"},
	}
	for _, tt := range tests {
		got := sanitizeSessionForFileName(tt.in)
		if got != tt.want {
			t.Errorf("sanitizeSessionForFileName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBrowserHostWrapperPath_Extension(t *testing.T) {
	path := browserHostWrapperPath("demo")
	wantExt := ".sh"
	if runtime.GOOS == "windows" {
		wantExt = ".bat"
	}
	if !strings.HasSuffix(path, "devlog-host-wrapper-demo"+wantExt) {
		t.Errorf("browserHostWrapperPath() = %q, want suffix devlog-host-wrapper-demo%s", path, wantExt)
	}
	if !strings.Contains(path, filepath.Join("devlog", "wrappers")) {
		t.Errorf("browserHostWrapperPath() = %q, want wrappers under devlog cache", path)
	}
}

func TestGenerateShellScript(t *testing.T) {
	got := generateShellScript(`/usr/local/bin/devlog-host`, `/tmp/logs/browser.log`, []string{"error", "warn"})
	want := "#!/bin/sh\nexec '/usr/local/bin/devlog-host' '/tmp/logs/browser.log' 'error' 'warn'\n"
	if got != want {
		t.Errorf("generateShellScript() = %q, want %q", got, want)
	}
}

func TestGenerateShellScript_EscapesSingleQuotes(t *testing.T) {
	got := generateShellScript(`/tmp/o'reilly/host`, `/tmp/log`, nil)
	if !strings.Contains(got, `'/tmp/o'\''reilly/host'`) {
		t.Errorf("generateShellScript() did not escape single quotes: %q", got)
	}
}

func TestGenerateBatchScript(t *testing.T) {
	got := generateBatchScript(`C:\Tools\devlog-host.exe`, `C:\Logs\browser.log`, []string{"error", "warn"})
	want := "@echo off\r\n" +
		`"C:\Tools\devlog-host.exe" "C:\Logs\browser.log" "error" "warn"` +
		"\r\n"
	if got != want {
		t.Errorf("generateBatchScript() = %q, want %q", got, want)
	}
}

func TestGenerateBatchScript_EscapesDoubleQuotes(t *testing.T) {
	got := generateBatchScript(`C:\Tools\dev"log-host.exe`, `C:\Logs\a.log`, nil)
	if !strings.Contains(got, `"C:\Tools\dev""log-host.exe"`) {
		t.Errorf("generateBatchScript() did not escape double quotes: %q", got)
	}
}

func TestBatchQuote(t *testing.T) {
	if batchQuote(`C:\a b\x.exe`) != `"C:\a b\x.exe"` {
		t.Errorf("batchQuote spaces = %q", batchQuote(`C:\a b\x.exe`))
	}
	if batchQuote(`say "hi"`) != `"say ""hi"""` {
		t.Errorf("batchQuote quotes = %q", batchQuote(`say "hi"`))
	}
}

func TestFindConfigFile(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, tmpDir string) string // returns the expected config path
		wantEmpty bool
	}{
		{
			name: "in current dir",
			setup: func(t *testing.T, tmpDir string) string {
				t.Chdir(tmpDir)
				return filepath.Join(tmpDir, "devlog.yml")
			},
			wantEmpty: false,
		},
		{
			name: "in ancestor dir",
			setup: func(t *testing.T, tmpDir string) string {
				// Nest three levels below tmpDir; the config two levels up should still resolve.
				nested := filepath.Join(tmpDir, "a", "b", "c")
				if err := os.MkdirAll(nested, 0755); err != nil {
					t.Fatalf("Failed to create nested dirs: %v", err)
				}
				t.Chdir(nested)
				return filepath.Join(tmpDir, "devlog.yml")
			},
			wantEmpty: false,
		},
		{
			name: "beyond bounded depth",
			setup: func(t *testing.T, tmpDir string) string {
				// Build a tree deeper than maxFindConfigDepth so the config at tmpDir is
				// out of reach from the deepest directory. The walk must exhaust its
				// budget before reaching the config and return "".
				current := tmpDir
				for i := 1; i <= maxFindConfigDepth+5; i++ {
					current = filepath.Join(current, fmt.Sprintf("d%02d", i))
				}
				if err := os.MkdirAll(current, 0755); err != nil {
					t.Fatalf("Failed to create deep nesting: %v", err)
				}
				t.Chdir(current)
				return filepath.Join(tmpDir, "devlog.yml")
			},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "devlog.yml")
			if err := os.WriteFile(configPath, []byte("version: \"1\"\n"), 0644); err != nil {
				t.Fatalf("Failed to write devlog.yml: %v", err)
			}

			expectedPath := tt.setup(t, tmpDir)
			got := findConfigFile()

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("findConfigFile() = %q, want empty (config should be beyond maxFindConfigDepth)", got)
				}
				return
			}
			if got != expectedPath {
				t.Errorf("findConfigFile() = %q, want %q", got, expectedPath)
			}
		})
	}
}
