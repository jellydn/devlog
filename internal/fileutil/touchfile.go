// Package fileutil provides small filesystem utility helpers.
package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DirMode is the permission for log directories created by devlog.
	// Owner-only so console output is not listed by other local users.
	DirMode os.FileMode = 0700
	// FileMode is the permission for log files created by devlog.
	FileMode os.FileMode = 0600
)

// TouchFile ensures the file at path exists. An existing file is not truncated.
func TouchFile(path string) error {
	return PrepareLogFile(path, false)
}

// PrepareLogFile creates parent directories and the file.
// When truncate is true, an existing file is replaced. Otherwise it is left in place.
func PrepareLogFile(path string, truncate bool) error {
	if err := os.MkdirAll(filepath.Dir(path), DirMode); err != nil {
		return fmt.Errorf("touch %q: %w", path, err)
	}
	flags := os.O_CREATE | os.O_WRONLY
	if truncate {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_APPEND
	}
	f, err := os.OpenFile(path, flags, FileMode)
	if err != nil {
		return fmt.Errorf("touch %q: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("touch %q: %w", path, err)
	}
	return nil
}

// SafeJoin joins rel under base and rejects absolute paths and ".." escapes.
// An empty rel returns an empty path and a nil error.
func SafeJoin(base, rel string) (string, error) {
	if rel == "" {
		return "", nil
	}
	if filepath.IsAbs(rel) || strings.Contains(rel, "\x00") {
		return "", fmt.Errorf("log path %q must be a relative path under logs_dir", rel)
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("log path base: %w", err)
	}
	absPath := filepath.Join(absBase, filepath.Clean(rel))
	relBack, err := filepath.Rel(absBase, absPath)
	if err != nil || relBack == ".." || strings.HasPrefix(relBack, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("log path %q escapes logs_dir", rel)
	}
	return absPath, nil
}
