// Package fileutil provides small filesystem utility helpers.
package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// TouchFile ensures the file at path exists by creating parent directories
// and the file itself if needed. If the file already exists, it is not truncated.
func TouchFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("touch %q: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("touch %q: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("touch %q: %w", path, err)
	}
	return nil
}
