package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jellydn/devlog/internal/natmsg"
	"github.com/jellydn/devlog/internal/shellescape"
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

func browserHostWrapperExt() string {
	if runtime.GOOS == "windows" {
		return ".bat"
	}
	return ".sh"
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
		fmt.Sprintf("devlog-host-wrapper-%s%s", sanitizeSessionForFileName(session), browserHostWrapperExt()),
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

	var script string
	if runtime.GOOS == "windows" {
		script = generateBatchScript(hostPath, absLogPath, levels)
	} else {
		script = generateShellScript(hostPath, absLogPath, levels)
	}
	if err := os.WriteFile(wrapperPath, []byte(script), 0755); err != nil {
		return err
	}

	if err := natmsg.UpdateManifestPath(wrapperPath); err != nil {
		return fmt.Errorf("failed to update native messaging manifest: %w", err)
	}

	return nil
}

func generateShellScript(hostPath, absLogPath string, levels []string) string {
	// Build script with proper shell escaping. exec replaces the shell with the host.
	var scriptArgs []string
	scriptArgs = append(scriptArgs, shellescape.Quote(hostPath), shellescape.Quote(absLogPath))
	for _, level := range levels {
		scriptArgs = append(scriptArgs, shellescape.Quote(level))
	}
	return fmt.Sprintf("#!/bin/sh\nexec %s\n", strings.Join(scriptArgs, " "))
}

func generateBatchScript(hostPath, absLogPath string, levels []string) string {
	// Native messaging on Windows can invoke a .bat host wrapper.
	var scriptArgs []string
	scriptArgs = append(scriptArgs, batchQuote(hostPath), batchQuote(absLogPath))
	for _, level := range levels {
		scriptArgs = append(scriptArgs, batchQuote(level))
	}
	return "@echo off\r\n" + strings.Join(scriptArgs, " ") + "\r\n"
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

// batchQuote returns a Windows batch-escaped argument using double quotes.
// Embedded double quotes are escaped by doubling them.
func batchQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
