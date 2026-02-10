// Package logger handles writing browser console logs to files.
package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jellydn/devlog/internal/natmsg"
)

// Logger writes browser console logs to a file with level filtering
type Logger struct {
	file    *os.File
	mu      sync.Mutex
	levels  map[string]bool
	logPath string
}

// New creates a new logger that writes to the specified file.
// If the directory doesn't exist, it will be created.
// levels is a list of log levels to capture (e.g., ["log", "error", "warn"]).
// If empty, all levels are captured.
func New(logPath string, levels []string) (*Logger, error) {
	// Create log directory if needed
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file (create or append)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Build level filter map
	levelMap := make(map[string]bool)
	for _, level := range levels {
		levelMap[strings.ToLower(level)] = true
	}

	return &Logger{
		file:    file,
		levels:  levelMap,
		logPath: logPath,
	}, nil
}

// Close closes the log file
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// ShouldLog returns true if the given level should be logged
func (l *Logger) ShouldLog(level string) bool {
	// If no levels specified, log everything
	if len(l.levels) == 0 {
		return true
	}
	return l.levels[strings.ToLower(level)]
}

// Log writes a message to the log file if it passes the level filter.
// The message is formatted as: [TIMESTAMP] [LEVEL] [URL] message
func (l *Logger) Log(msg *natmsg.Message) error {
	if !l.ShouldLog(msg.Level) {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := formatTimestamp(msg.Timestamp)

	// Build log line
	var logLine strings.Builder
	logLine.WriteString("[")
	logLine.WriteString(timestamp)
	logLine.WriteString("] [")
	logLine.WriteString(strings.ToUpper(msg.Level))
	logLine.WriteString("]")

	if msg.URL != "" {
		logLine.WriteString(" [")
		logLine.WriteString(msg.URL)
		logLine.WriteString("]")
	}

	if msg.Source != "" {
		logLine.WriteString(" ")
		logLine.WriteString(msg.Source)
		loc := formatLoc(msg.Line, msg.Column)
		if loc != "" {
			logLine.WriteString(loc)
		}
	}

	logLine.WriteString(": ")
	logLine.WriteString(msg.Message)
	logLine.WriteString("\n")

	if _, err := l.file.WriteString(logLine.String()); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	return nil
}

func formatTimestamp(v interface{}) string {
	switch t := v.(type) {
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return parsed.Format("2006-01-02 15:04:05.000")
		}
		if parsed, err := time.Parse("2006-01-02T15:04:05.000Z", t); err == nil {
			return parsed.Format("2006-01-02 15:04:05.000")
		}
		return t
	case int, int64, uint, uint64:
		return time.UnixMilli(toInt64(t)).Format("2006-01-02 15:04:05.000")
	case float64:
		return time.UnixMilli(int64(t)).Format("2006-01-02 15:04:05.000")
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return time.UnixMilli(n).Format("2006-01-02 15:04:05.000")
		}
		if n, err := t.Float64(); err == nil {
			return time.UnixMilli(int64(n)).Format("2006-01-02 15:04:05.000")
		}
		return t.String()
	default:
		return fmt.Sprintf("[unparsable timestamp: %v]", v)
	}
}

func toInt64(v interface{}) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case uint:
		return int64(t)
	case uint64:
		return int64(t)
	default:
		return 0
	}
}

func formatLoc(line, column interface{}) string {
	l := toInt(line)
	c := toInt(column)
	if l > 0 {
		if c > 0 {
			return fmt.Sprintf(":%d:%d", l, c)
		}
		return fmt.Sprintf(":%d", l)
	}
	return ""
}

func toInt(v interface{}) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	default:
		return 0
	}
}

// LogPath returns the path to the log file
func (l *Logger) LogPath() string {
	return l.logPath
}
