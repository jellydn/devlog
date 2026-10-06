package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jellydn/devlog/internal/fileutil"
	"gopkg.in/yaml.v3"
)

// DefaultMaxLogBytes is the per-file cap used when max_log_bytes is omitted.
// Set max_log_bytes to -1 in devlog.yml to disable the cap.
const DefaultMaxLogBytes = 256 << 20

// Config represents the devlog.yml configuration
type Config struct {
	Version       string `yaml:"version"`
	Project       string `yaml:"project"`
	LogsDir       string `yaml:"logs_dir"`
	RunMode       string `yaml:"run_mode"`
	MaxRuns       int    `yaml:"max_runs"`
	RetentionDays int    `yaml:"retention_days"`
	// MaxLogBytes caps each browser log file. 0 becomes DefaultMaxLogBytes.
	// A negative value means no cap.
	MaxLogBytes int           `yaml:"max_log_bytes"`
	Tmux        TmuxConfig    `yaml:"tmux"`
	Browser     BrowserConfig `yaml:"browser"`
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

	// Interpolate after the YAML tree exists so an environment value
	// cannot insert new keys. Placeholders stay inside their scalar.
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	interpolateNode(&doc)

	var cfg Config
	if err := doc.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	if cfg.LogsDir == "" {
		cfg.LogsDir = "./logs"
	}
	if cfg.RunMode == "" {
		cfg.RunMode = "timestamped"
	}
	if cfg.MaxLogBytes == 0 {
		cfg.MaxLogBytes = DefaultMaxLogBytes
	} else if cfg.MaxLogBytes < 0 {
		cfg.MaxLogBytes = 0
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
	if strings.ContainsAny(c.Tmux.Session, "\r\n\x00") {
		return fmt.Errorf("config: tmux.session must not contain a newline")
	}
	if len(c.Tmux.Windows) == 0 {
		return fmt.Errorf("config: tmux.windows must have at least one window")
	}
	names := make(map[string]struct{}, len(c.Tmux.Windows))
	for i, window := range c.Tmux.Windows {
		if window.Name == "" {
			return fmt.Errorf("config: tmux.windows[%d].name is required", i)
		}
		if strings.ContainsAny(window.Name, ":.\r\n") {
			return fmt.Errorf("config: tmux.windows[%d].name %q must not contain ':', '.', or a newline", i, window.Name)
		}
		if _, dup := names[window.Name]; dup {
			return fmt.Errorf("config: duplicate window name %q", window.Name)
		}
		names[window.Name] = struct{}{}
		if len(window.Panes) == 0 {
			return fmt.Errorf("config: tmux.windows[%d].panes must have at least one pane", i)
		}
		for j, pane := range window.Panes {
			if pane.Cmd == "" {
				return fmt.Errorf("config: tmux.windows[%d].panes[%d].cmd is required", i, j)
			}
			if pane.Log != "" {
				if _, err := fileutil.SafeJoin(c.LogsDir, pane.Log); err != nil {
					return fmt.Errorf("config: tmux.windows[%d].panes[%d].log: %w", i, j, err)
				}
			}
		}
	}
	if c.Browser.File != "" {
		if _, err := fileutil.SafeJoin(c.LogsDir, c.Browser.File); err != nil {
			return fmt.Errorf("config: browser.file: %w", err)
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

// envVarRegex matches $VAR or ${VAR} patterns
var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

// interpolateNode replaces $VAR and ${VAR} inside YAML scalars.
// The tree shape is fixed before substitution, so a value cannot add keys.
func interpolateNode(n *yaml.Node) {
	if n == nil {
		return
	}
	if n.Kind == yaml.ScalarNode && strings.Contains(n.Value, "$") {
		original := n.Value
		n.Value = interpolateEnvVars(n.Value)
		if n.Value != original && entireScalarIsEnv(original) && isDecimal(n.Value) {
			n.Tag = "!!int"
		}
	}
	for _, child := range n.Content {
		interpolateNode(child)
	}
}

func entireScalarIsEnv(s string) bool {
	return envVarRegex.FindString(s) == s
}

func isDecimal(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

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
