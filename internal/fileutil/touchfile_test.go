package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTouchFile_CreatesFileAndParents(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "a", "b", "test.log")

	if err := TouchFile(path); err != nil {
		t.Fatalf("TouchFile: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestTouchFile_ExistingFileNotTruncated(t *testing.T) {
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

func TestTouchFile_DefaultPermissions(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "perms.log")

	if err := TouchFile(path); err != nil {
		t.Fatalf("TouchFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("mode = %o, want 0644", info.Mode().Perm())
	}
}
