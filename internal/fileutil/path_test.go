package fileutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPrepareLogFile_Truncate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.log")
	if err := os.WriteFile(path, []byte("old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := PrepareLogFile(path, true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Errorf("data = %q, want empty", data)
	}
}

func TestSafeJoin(t *testing.T) {
	base := t.TempDir()
	tests := []struct {
		name    string
		rel     string
		wantErr bool
	}{
		{name: "Relative", rel: "browser.log"},
		{name: "Nested", rel: filepath.Join("server", "api.log")},
		{name: "Empty", rel: ""},
		{name: "Parent", rel: "../secret", wantErr: true},
		{name: "Absolute", rel: filepath.Join(base, "x"), wantErr: true},
		{name: "DotDotInside", rel: "server/../../etc/passwd", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeJoin(base, tt.rel)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("SafeJoin(%q) = %q, want error", tt.rel, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("SafeJoin(%q): %v", tt.rel, err)
			}
			if tt.rel == "" {
				if got != "" {
					t.Errorf("got %q, want empty", got)
				}
				return
			}
			if !strings.HasPrefix(got, base) {
				t.Errorf("got %q, want prefix %q", got, base)
			}
		})
	}
	if runtime.GOOS == "windows" {
		t.Skip("absolute-path fixture above uses the temp dir on every OS")
	}
}
