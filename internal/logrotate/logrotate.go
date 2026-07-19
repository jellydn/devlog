// Package logrotate handles cleanup of old timestamped log directories
// based on configurable retention policies (max runs and retention days).
package logrotate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Policy defines retention rules for timestamped log directories.
type Policy struct {
	MaxRuns       int
	RetentionDays int
}

// Result describes what was cleaned up.
type Result struct {
	// Removed lists directories that were removed (or would be removed in dry-run).
	Removed []string
	// Failed maps directory paths that could not be removed to the error.
	Failed map[string]error
}

// Cleanup removes old log directories from logsDir based on the retention policy.
// In dry-run mode, directories are listed but not removed.
// Callers should only invoke Cleanup for timestamped run mode.
func Cleanup(logsDir string, policy Policy, dryRun bool) (*Result, error) {
	result := &Result{
		Failed: make(map[string]error),
	}

	// Skip if no retention policy is set
	if policy.MaxRuns == 0 && policy.RetentionDays == 0 {
		return result, nil
	}

	entries, err := os.ReadDir(logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil // Nothing to clean up
		}
		return result, fmt.Errorf("failed to read logs directory: %w", err)
	}

	// Collect directories with their info
	type dirInfo struct {
		entry   os.DirEntry
		modTime time.Time
	}
	var dirs []dirInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		dirs = append(dirs, dirInfo{entry: entry, modTime: info.ModTime()})
	}

	// Sort by modification time (newest first)
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].modTime.After(dirs[j].modTime)
	})

	// Determine which directories to remove
	toRemove := make(map[string]bool)

	// Apply max_runs policy
	if policy.MaxRuns > 0 && len(dirs) > policy.MaxRuns {
		for _, dir := range dirs[policy.MaxRuns:] {
			toRemove[dir.entry.Name()] = true
		}
	}

	// Apply retention_days policy
	if policy.RetentionDays > 0 {
		cutoffTime := time.Now().AddDate(0, 0, -policy.RetentionDays)
		for _, dir := range dirs {
			if dir.modTime.Before(cutoffTime) {
				toRemove[dir.entry.Name()] = true
			}
		}
	}

	// Remove directories (or record dry-run candidates)
	for name := range toRemove {
		dirPath := filepath.Join(logsDir, name)
		if dryRun {
			result.Removed = append(result.Removed, dirPath)
			continue
		}
		if err := os.RemoveAll(dirPath); err != nil {
			result.Failed[dirPath] = err
			continue
		}
		result.Removed = append(result.Removed, dirPath)
	}

	return result, nil
}
