package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jellydn/devlog/internal/manifest"
	"github.com/jellydn/devlog/internal/shellescape"
	"github.com/jellydn/devlog/internal/tmux"
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
	hostPath, err := manifest.FindDevlogHostBinary()
	if err != nil {
		return err
	}
	return writeBrowserHostWrapperWithHost(session, browserLogPath, levels, hostPath)
}

// writeBrowserHostWrapperWithHost is the testable core of writeBrowserHostWrapper.
func writeBrowserHostWrapperWithHost(session, browserLogPath string, levels []string, hostPath string) error {
	if err := manifest.ValidateHostPath(hostPath); err != nil {
		return fmt.Errorf("untrusted host binary: %w", err)
	}

	// Self-heal: if a previous unclean shutdown left manifests pointing at a
	// missing wrapper, restore them to the real binary before we rewrite.
	if _, err := manifest.RepairStaleManifestPaths(hostPath); err != nil {
		// Non-fatal when no manifests exist yet.
		if !strings.Contains(err.Error(), "failed to read some manifests") &&
			!strings.Contains(err.Error(), "no native messaging") {
			// continue; UpdateManifestPath will report clearer errors
		}
	}

	absLogPath, err := filepath.Abs(browserLogPath)
	if err != nil {
		return err
	}

	wrapperPath := browserHostWrapperPath(session)
	if err := refuseClobberActiveWrapper(wrapperPath); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0700); err != nil {
		return err
	}

	var script string
	if runtime.GOOS == "windows" {
		script = generateBatchScript(hostPath, absLogPath, levels)
	} else {
		script = generateShellScript(hostPath, absLogPath, levels)
	}
	if err := os.WriteFile(wrapperPath, []byte(script), 0700); err != nil {
		return err
	}

	if err := manifest.UpdateManifestPath(wrapperPath); err != nil {
		return fmt.Errorf("failed to update native messaging manifest: %w", err)
	}

	return nil
}

// refuseClobberActiveWrapper returns an error if any installed manifest currently
// points at a different session's wrapper whose tmux session is still alive.
func refuseClobberActiveWrapper(desiredWrapper string) error {
	desiredWrapper = filepath.Clean(desiredWrapper)
	paths, err := manifest.ReadManifestPaths()
	if err != nil && len(paths) == 0 {
		// No manifests installed yet — nothing to clobber.
		return nil
	}
	for _, current := range paths {
		current = filepath.Clean(current)
		if current == desiredWrapper {
			continue
		}
		otherSession, ok := sessionFromWrapperPath(current)
		if !ok {
			continue
		}
		// Only refuse when the other wrapper still exists AND its session is live.
		if _, statErr := os.Stat(current); statErr != nil {
			continue
		}
		if tmux.NewRunner(otherSession).SessionExists() {
			return fmt.Errorf("browser logging is already in use by session %q; run 'devlog down' in that session first", otherSession)
		}
	}
	return nil
}

// sessionFromWrapperPath extracts the session name from a wrapper path like
// .../devlog-host-wrapper-<session>.sh|.bat
func sessionFromWrapperPath(path string) (string, bool) {
	base := filepath.Base(path)
	const prefix = "devlog-host-wrapper-"
	if !strings.HasPrefix(base, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(base, prefix)
	for _, ext := range []string{".sh", ".bat"} {
		if strings.HasSuffix(name, ext) {
			name = strings.TrimSuffix(name, ext)
			break
		}
	}
	if name == "" {
		return "", false
	}
	return name, true
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
	hostPath, err := manifest.FindDevlogHostBinary()
	if err != nil {
		return
	}
	restoreBrowserHostWrapperWithHost(session, hostPath)
}

// restoreBrowserHostWrapperWithHost restores manifests that still point at this
// session's wrapper (even if the wrapper file was already deleted) and removes
// the wrapper file if present.
func restoreBrowserHostWrapperWithHost(session, hostPath string) {
	wrapperPath := browserHostWrapperPath(session)

	// Restore if our wrapper is referenced, or if any path is missing (stale).
	inUse, err := manifest.IsManifestPathInUse(wrapperPath)
	if err == nil && inUse {
		_ = manifest.UpdateManifestPath(hostPath)
	} else {
		// Also repair any other stale missing paths back to the real binary.
		_, _ = manifest.RepairStaleManifestPaths(hostPath)
	}

	_ = os.Remove(wrapperPath)
}

// batchQuote returns a Windows batch-escaped argument using double quotes.
// Embedded double quotes are escaped by doubling them.
func batchQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
