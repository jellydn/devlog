package fileutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestTouchFile(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"CreatesFileAndParents", testCreatesFileAndParents},
		{"ExistingFileNotTruncated", testExistingFileNotTruncated},
		{"DefaultPermissions", testDefaultPermissions},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func testCreatesFileAndParents(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "a", "b", "test.log")

	if err := TouchFile(path); err != nil {
		t.Fatalf("TouchFile: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func testExistingFileNotTruncated(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "existing.log")

	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := TouchFile(path); err != nil {
		t.Fatalf("TouchFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("data = %q, want %q", string(data), "hello")
	}
}

func testDefaultPermissions(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "perms.log")

	if err := TouchFile(path); err != nil {
		t.Fatalf("TouchFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// On Windows, os.Stat().Mode().Perm() does not reflect Unix permission bits
	// (the OS uses ACLs instead), so skip the exact-mode assertion.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0644 {
		t.Errorf("mode = %o, want 0644", info.Mode().Perm())
	}
}
