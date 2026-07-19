package main

import (
	"os"
	"path/filepath"
)

// maxFindConfigDepth limits how far findConfigFile walks up the directory tree.
// 20 levels is generous for any realistic project nesting while preventing a
// walk all the way to the filesystem root when invoked from an unrelated path.
const maxFindConfigDepth = 20

func findConfigFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for depth := 0; depth < maxFindConfigDepth; depth++ {
		configPath := filepath.Join(dir, "devlog.yml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		// Go up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	return ""
}

func ensureFileExists(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	return f.Close()
}
