package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the devlog.yml configuration
type Config struct {
	Version       string        `yaml:"version"`
	Project       string        `yaml:"project"`
	LogsDir       string        `yaml:"logs_dir"`
	RunMode       string        `yaml:"run_mode"`
	MaxRuns       int           `yaml:"max_runs"`
	RetentionDays int           `yaml:"retention_days"`
	Tmux          TmuxConfig    `yaml:"tmux"`
	Browser       BrowserConfig `yaml:"browser"`
}

// TmuxConfig represents tmux session configuration
type TmuxConfig struct {
	Session string         `yaml:"session"`
	Windows []WindowConfig `yaml:"windows"`
}

// WindowConfig represents a tmux window
type WindowConfig struct {
	Name  string       `yaml:"name"`
	Panes []PaneConfig `yaml:"panes"`
}

// PaneConfig represents a tmux pane
type PaneConfig struct {
	Cmd string `yaml:"cmd"`
	Log string `yaml:"log"`
}

// BrowserConfig represents browser log capture configuration
type BrowserConfig struct {
	URLs   []string `yaml:"urls"`
	File   string   `yaml:"file"`
	Levels []string `yaml:"levels"`
}

// Load reads and parses the devlog.yml file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Interpolate environment variables
	interpolated := interpolateEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(interpolated), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	if cfg.LogsDir == "" {
		cfg.LogsDir = "./logs"
	}
	if cfg.RunMode == "" {
		cfg.RunMode = "timestamped"
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks that all required fields are present and valid
func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("config: version is required")
	}
	if c.Project == "" {
		return fmt.Errorf("config: project is required")
	}
	if c.Tmux.Session == "" {
		return fmt.Errorf("config: tmux.session is required")
	}
	if len(c.Tmux.Windows) == 0 {
		return fmt.Errorf("config: tmux.windows must have at least one window")
	}
	for i, window := range c.Tmux.Windows {
		if len(window.Panes) == 0 {
			return fmt.Errorf("config: tmux.windows[%d].panes must have at least one pane", i)
		}
		for j, pane := range window.Panes {
			if pane.Cmd == "" {
				return fmt.Errorf("config: tmux.windows[%d].panes[%d].cmd is required", i, j)
			}
		}
	}
	if c.RunMode != "timestamped" && c.RunMode != "overwrite" {
		return fmt.Errorf("config: run_mode must be 'timestamped' or 'overwrite', got '%s'", c.RunMode)
	}
	if c.MaxRuns < 0 {
		return fmt.Errorf("config: max_runs must be non-negative, got %d", c.MaxRuns)
	}
	if c.RetentionDays < 0 {
		return fmt.Errorf("config: retention_days must be non-negative, got %d", c.RetentionDays)
	}
	return nil
}

// ResolveLogsDir returns the actual logs directory based on run_mode
func (c *Config) ResolveLogsDir() string {
	if c.RunMode == "timestamped" {
		timestamp := time.Now().Format("20060102-150405")
		return filepath.Join(c.LogsDir, timestamp)
	}
	return c.LogsDir
}

// envVarRegex matches $VAR or ${VAR} patterns
var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

// interpolateEnvVars replaces environment variable placeholders with their values
func interpolateEnvVars(input string) string {
	return envVarRegex.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name
		var varName string
		if strings.HasPrefix(match, "${") {
			varName = match[2 : len(match)-1]
		} else {
			varName = match[1:]
		}

		// Get environment variable value
		value := os.Getenv(varName)
		if value == "" {
			// Return original if not set (could also return empty string)
			return match
		}
		return value
	})
}

// CleanupOldRuns removes old log directories based on retention policy
func (c *Config) CleanupOldRuns(dryRun bool) error {
	// Only cleanup for timestamped mode
	if c.RunMode != "timestamped" {
		return nil
	}

	// Skip if no retention policy is set
	if c.MaxRuns == 0 && c.RetentionDays == 0 {
		return nil
	}

	entries, err := os.ReadDir(c.LogsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to clean up
		}
		return fmt.Errorf("failed to read logs directory: %w", err)
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
	for i := 0; i < len(dirs); i++ {
		for j := i + 1; j < len(dirs); j++ {
			if dirs[i].modTime.Before(dirs[j].modTime) {
				dirs[i], dirs[j] = dirs[j], dirs[i]
			}
		}
	}

	// Determine which directories to remove
	var toRemove []string

	// Apply max_runs policy
	if c.MaxRuns > 0 && len(dirs) > c.MaxRuns {
		for _, dir := range dirs[c.MaxRuns:] {
			toRemove = append(toRemove, dir.entry.Name())
		}
	}

	// Apply retention_days policy
	if c.RetentionDays > 0 {
		cutoffTime := time.Now().AddDate(0, 0, -c.RetentionDays)
		for _, dir := range dirs {
			if dir.modTime.Before(cutoffTime) {
				// Check if already in toRemove list
				alreadyAdded := false
				for _, name := range toRemove {
					if name == dir.entry.Name() {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					toRemove = append(toRemove, dir.entry.Name())
				}
			}
		}
	}

	// Remove directories
	for _, name := range toRemove {
		dirPath := filepath.Join(c.LogsDir, name)
		if dryRun {
			fmt.Printf("[DRY RUN] Would remove: %s\n", dirPath)
		} else {
			if err := os.RemoveAll(dirPath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove %s: %v\n", dirPath, err)
			} else {
				fmt.Printf("Removed old log directory: %s\n", dirPath)
			}
		}
	}

	return nil
}
