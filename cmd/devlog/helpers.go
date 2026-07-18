package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jellydn/devlog/internal/natmsg"
)

func findConfigFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
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

func browserHostWrapperPath(session string) string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	return filepath.Join(
		cacheDir,
		"devlog",
		"wrappers",
		fmt.Sprintf("devlog-host-wrapper-%s.sh", sanitizeSessionForFileName(session)),
	)
}

func sanitizeSessionForFileName(session string) string {
	if session == "" {
		return "default"
	}

	var b strings.Builder
	b.Grow(len(session))
	for _, r := range session {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}

	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "default"
	}
	return out
}

func writeBrowserHostWrapper(session string, browserLogPath string, levels []string) error {
	hostPath, err := natmsg.FindDevlogHostBinary()
	if err != nil {
		return err
	}

	absLogPath, err := filepath.Abs(browserLogPath)
	if err != nil {
		return err
	}

	wrapperPath := browserHostWrapperPath(session)
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0755); err != nil {
		return err
	}

	// Build script with proper shell escaping using positional parameters
	// Use "$@" to safely pass arguments without re-parsing by the shell
	var scriptArgs []string
	scriptArgs = append(scriptArgs, shellQuote(hostPath), shellQuote(absLogPath))
	for _, level := range levels {
		scriptArgs = append(scriptArgs, shellQuote(level))
	}
	script := fmt.Sprintf("#!/bin/sh\nexec %s\n", strings.Join(scriptArgs, " "))
	if err := os.WriteFile(wrapperPath, []byte(script), 0755); err != nil {
		return err
	}

	if err := natmsg.UpdateManifestPath(wrapperPath); err != nil {
		return fmt.Errorf("failed to update native messaging manifest: %w", err)
	}

	return nil
}

func restoreBrowserHostWrapper(session string) {
	hostPath, err := natmsg.FindDevlogHostBinary()
	if err != nil {
		return
	}

	wrapperPath := browserHostWrapperPath(session)
	inUse, err := natmsg.IsManifestPathInUse(wrapperPath)
	if err == nil && inUse {
		natmsg.UpdateManifestPath(hostPath)
	}

	os.Remove(wrapperPath)
}

// shellQuote returns a shell-escaped version of the string using single quotes.
// Any single quotes in the input are escaped as '\”' to safely include them.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// openInFileManager opens the given path in the system file manager
