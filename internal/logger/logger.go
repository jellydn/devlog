// Package logger handles writing browser console logs to files.
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

	timestamp := msg.Timestamp.Format("2006-01-02 15:04:05.000")

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

func formatLoc(line, column *int) string {
	if line != nil && *line > 0 {
		if column != nil && *column > 0 {
			return fmt.Sprintf(":%d:%d", *line, *column)
		}
		return fmt.Sprintf(":%d", *line)
	}
	return ""
}

// LogPath returns the path to the log file
func (l *Logger) LogPath() string {
	return l.logPath
}
