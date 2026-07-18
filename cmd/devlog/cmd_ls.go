package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jellydn/devlog/internal/config"
)

func cmdLs(cfg *config.Config, args []string) error {
	logsDir := cfg.LogsDir

	entries, err := os.ReadDir(logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("No log runs found (logs directory '%s' does not exist)\n", logsDir)
			return nil
		}
		return fmt.Errorf("failed to read logs directory: %w", err)
	}

	var dirs []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e)
		}
	}

	if cfg.RunMode == "timestamped" {
		if len(dirs) == 0 {
			fmt.Println("No log runs found")
			return nil
		}
		fmt.Printf("Log runs in %s (%d):\n", logsDir, len(dirs))
		for _, d := range dirs {
			info, err := d.Info()
			if err != nil {
				fmt.Printf("  %s\n", d.Name())
				continue
			}
			logFiles := countFiles(filepath.Join(logsDir, d.Name()))
			fmt.Printf("  %s  (%d files, %s)\n", d.Name(), logFiles, info.ModTime().Format("Jan 02 15:04"))
		}
	} else {
		files := countFiles(logsDir)
		fmt.Printf("Logs directory: %s (%d files)\n", logsDir, files)
	}

	return nil
}

func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}
	return count
}
