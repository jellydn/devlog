// Package browsersession manages the browser-log capture lifecycle:
// creating/destroying the native messaging wrapper, guarding against
// clobbering active sessions, and health-checking the setup.
package browsersession

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jellydn/devlog/internal/shellescape"
)

// ManifestOps is the seam to the manifest module (interface-based DI for testability).
type ManifestOps interface {
	FindDevlogHostBinary() (string, error)
	ValidateHostPath(path string) error
	RepairStaleManifestPaths(hostPath string) (int, error)
	UpdateManifestPath(newPath string) error
	ReadManifestPaths() (map[string]string, error)
	IsManifestPathInUse(targetPath string) (bool, error)
	GetChromeNativeMessagingDir() string
	GetBraveNativeMessagingDir() string
	GetFirefoxNativeMessagingDirs() []string
}

// SessionChecker checks if a tmux session is alive (interface-based DI).
type SessionChecker interface {
	SessionExists(name string) bool
}

// Session manages the browser-log wrapper lifecycle for one devlog session.
type Session struct {
	manifest ManifestOps
	tmux     SessionChecker
}

// New creates a Session with the given dependencies.
func New(manifest ManifestOps, tmux SessionChecker) *Session {
	return &Session{manifest: manifest, tmux: tmux}
}

// HealthResult summarizes browser/host healthcheck findings.
type HealthResult struct {
	HostPath      string
	HostFound     bool
	Registered    []string // browser names: "Chrome", "Brave", "Firefox"
	ManifestPaths int
	RepairedPaths int
	StalePaths    int
}

// Start creates the native messaging wrapper script, guards against clobbering
// an active session's wrapper, and updates all installed manifests to point at it.
func (s *Session) Start(sessionName, browserLogPath string, levels []string) error {
	hostPath, err := s.manifest.FindDevlogHostBinary()
	if err != nil {
		return err
	}
	return s.start(sessionName, browserLogPath, levels, hostPath)
}

// Stop restores manifests to point at the real devlog-host binary and removes
// the wrapper script.
func (s *Session) Stop(sessionName string) {
	hostPath, err := s.manifest.FindDevlogHostBinary()
	if err != nil {
		return
	}
	s.stop(sessionName, hostPath)
}

// HealthCheck verifies the host binary exists, manifests are registered,
// and manifest paths point at existing files (repairing stale paths).
func (s *Session) HealthCheck() (*HealthResult, error) {
	result := &HealthResult{}

	hostPath, err := s.manifest.FindDevlogHostBinary()
	if err != nil {
		result.HostFound = false
	} else {
		result.HostFound = true
		result.HostPath = hostPath
	}

	chromeManifestPath := filepath.Join(s.manifest.GetChromeNativeMessagingDir(), "com.devlog.host.json")
	braveManifestPath := filepath.Join(s.manifest.GetBraveNativeMessagingDir(), "com.devlog.host.json")
	firefoxManifestPaths := []string{}
	for _, dir := range s.manifest.GetFirefoxNativeMessagingDirs() {
		firefoxManifestPaths = append(firefoxManifestPaths, filepath.Join(dir, "com.devlog.host.json"))
	}

	if _, err := os.Stat(chromeManifestPath); err == nil {
		result.Registered = append(result.Registered, "Chrome")
	}
	if _, err := os.Stat(braveManifestPath); err == nil {
		result.Registered = append(result.Registered, "Brave")
	}
	for _, path := range firefoxManifestPaths {
		if _, err := os.Stat(path); err == nil {
			result.Registered = append(result.Registered, "Firefox")
			break
		}
	}

	if result.HostFound {
		if repaired, err := s.manifest.RepairStaleManifestPaths(hostPath); err == nil && repaired > 0 {
			result.RepairedPaths = repaired
		}
	}

	paths, pathErr := s.manifest.ReadManifestPaths()
	if pathErr != nil && len(paths) == 0 {
		return result, nil
	}
	result.ManifestPaths = len(paths)
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			result.StalePaths++
		}
	}
	return result, nil
}

func (s *Session) start(session, browserLogPath string, levels []string, hostPath string) error {
	if err := s.manifest.ValidateHostPath(hostPath); err != nil {
		return fmt.Errorf("untrusted host binary: %w", err)
	}

	// Self-heal: if a previous unclean shutdown left manifests pointing at a
	// missing wrapper, restore them to the real binary before we rewrite.
	if _, err := s.manifest.RepairStaleManifestPaths(hostPath); err != nil {
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
	if err := s.refuseClobberActiveWrapper(wrapperPath); err != nil {
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

	if err := s.manifest.UpdateManifestPath(wrapperPath); err != nil {
		return fmt.Errorf("failed to update native messaging manifest: %w", err)
	}

	return nil
}

func (s *Session) stop(session, hostPath string) {
	wrapperPath := browserHostWrapperPath(session)

	// Restore if our wrapper is referenced, or if any path is missing (stale).
	inUse, err := s.manifest.IsManifestPathInUse(wrapperPath)
	if err == nil && inUse {
		_ = s.manifest.UpdateManifestPath(hostPath)
	} else {
		// Also repair any other stale missing paths back to the real binary.
		_, _ = s.manifest.RepairStaleManifestPaths(hostPath)
	}

	_ = os.Remove(wrapperPath)
}

// refuseClobberActiveWrapper returns an error if any installed manifest currently
// points at a different session's wrapper whose tmux session is still alive.
func (s *Session) refuseClobberActiveWrapper(desiredWrapper string) error {
	desiredWrapper = filepath.Clean(desiredWrapper)
	paths, err := s.manifest.ReadManifestPaths()
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
		if s.tmux.SessionExists(otherSession) {
			return fmt.Errorf("browser logging is already in use by session %q; run 'devlog down' in that session first", otherSession)
		}
	}
	return nil
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

// batchQuote returns a Windows batch-escaped argument using double quotes.
// Embedded double quotes are escaped by doubling them.
func batchQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
