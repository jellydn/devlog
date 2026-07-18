package main

import (
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

func TestShellQuote(t *testing.T) {
	if shellQuote("") != "''" {
		t.Errorf("shellQuote empty = %q", shellQuote(""))
	}
	if shellQuote("plain") != "'plain'" {
		t.Errorf("shellQuote plain = %q", shellQuote("plain"))
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
