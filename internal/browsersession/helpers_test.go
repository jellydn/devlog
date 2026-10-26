package browsersession

import (
	"fmt"
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
	wantSuffix := fmt.Sprintf("devlog-host-wrapper-demo-%08x%s", sessionHash("demo"), wantExt)
	if !strings.HasSuffix(path, wantSuffix) {
		t.Errorf("browserHostWrapperPath() = %q, want suffix %s", path, wantSuffix)
	}
	other := browserHostWrapperPath("a-b-c")
	sameShape := browserHostWrapperPath("a/b:c")
	if other == sameShape {
		t.Errorf("sanitized names collided: %s", other)
	}
	if !strings.Contains(path, filepath.Join("devlog", "wrappers")) {
		t.Errorf("browserHostWrapperPath() = %q, want wrappers under devlog cache", path)
	}
}

func TestGenerateShellScript(t *testing.T) {
	got := generateShellScript("demo", `/usr/local/bin/devlog-host`, `/tmp/logs/browser.log`, []string{"error", "warn"}, nil, 0)
	want := "#!/bin/sh\n# devlog-session: " + sessionMarker("demo") + "\n" +
		"if ! tmux has-session -t 'demo' 2>/dev/null; then\n  exit 0\nfi\n" +
		"exec '/usr/local/bin/devlog-host' '/tmp/logs/browser.log' 'error' 'warn'\n"
	if got != want {
		t.Errorf("generateShellScript() = %q, want %q", got, want)
	}
}

func TestGenerateShellScript_EscapesSingleQuotes(t *testing.T) {
	got := generateShellScript("demo", `/tmp/o'reilly/host`, `/tmp/log`, nil, nil, 0)
	if !strings.Contains(got, `'/tmp/o'\''reilly/host'`) {
		t.Errorf("generateShellScript() did not escape single quotes: %q", got)
	}
}

func TestGenerateBatchScript(t *testing.T) {
	got := generateBatchScript("demo", `C:\Tools\devlog-host.exe`, `C:\Logs\browser.log`, []string{"error", "warn"}, nil, 0)
	want := "@echo off\r\nrem devlog-session: " + sessionMarker("demo") +
		"\r\ntmux has-session -t \"demo\" >nul 2>&1\r\nif errorlevel 1 exit /b 0\r\n" +
		`"C:\Tools\devlog-host.exe" "C:\Logs\browser.log" "error" "warn"` +
		"\r\n"
	if got != want {
		t.Errorf("generateBatchScript() = %q, want %q", got, want)
	}
}

func TestGenerateBatchScript_EscapesDoubleQuotes(t *testing.T) {
	got := generateBatchScript("demo", `C:\Tools\dev"log-host.exe`, `C:\Logs\a.log`, nil, nil, 0)
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
	if batchQuote(`100%`) != `"100%%"` {
		t.Errorf("batchQuote percent = %q", batchQuote(`100%`))
	}
	if batchQuote(`a&b`) != `"a&b"` {
		t.Errorf("batchQuote ampersand = %q", batchQuote(`a&b`))
	}
	if batchQuote("a\nb") != `"a b"` {
		t.Errorf("batchQuote newline = %q", batchQuote("a\nb"))
	}
}
