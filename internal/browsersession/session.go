// Package browsersession manages the browser-log capture lifecycle:
// creating/destroying the native messaging wrapper, guarding against
// clobbering active sessions, and health-checking the setup.
package browsersession

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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
	manifest    ManifestOps
	tmux        SessionChecker
	URLs        []string
	MaxLogBytes int64
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

// HealthCheck verifies the host binary exists and reports stale manifest paths.
// It does not rewrite manifests. devlog up and devlog down do that.
func (s *Session) HealthCheck() (*HealthResult, error) {
	result := &HealthResult{}

	s.discoverHost(result)
	s.checkRegisteredBrowsers(result)
	s.repairAndCountPaths(result)

	return result, nil
}

// discoverHost finds the devlog-host binary and records whether it exists.
func (s *Session) discoverHost(result *HealthResult) {
	hostPath, err := s.manifest.FindDevlogHostBinary()
	if err != nil {
		result.HostFound = false
		return
	}
	result.HostFound = true
	result.HostPath = hostPath
}

// checkRegisteredBrowsers checks which browsers have native messaging manifests
// installed and records them in the result.
func (s *Session) checkRegisteredBrowsers(result *HealthResult) {
	chromeManifestPath := filepath.Join(s.manifest.GetChromeNativeMessagingDir(), "com.devlog.host.json")
	braveManifestPath := filepath.Join(s.manifest.GetBraveNativeMessagingDir(), "com.devlog.host.json")

	if _, err := os.Stat(chromeManifestPath); err == nil {
		result.Registered = append(result.Registered, "Chrome")
	}
	if _, err := os.Stat(braveManifestPath); err == nil {
		result.Registered = append(result.Registered, "Brave")
	}
	for _, dir := range s.manifest.GetFirefoxNativeMessagingDirs() {
		if _, err := os.Stat(filepath.Join(dir, "com.devlog.host.json")); err == nil {
			result.Registered = append(result.Registered, "Firefox")
			break
		}
	}
}

// repairAndCountPaths counts manifest entries. A missing file, or a wrapper
// whose tmux session is gone, is stale. This method does not write.
func (s *Session) repairAndCountPaths(result *HealthResult) {
	paths, pathErr := s.manifest.ReadManifestPaths()
	if pathErr != nil && len(paths) == 0 {
		return
	}
	result.ManifestPaths = len(paths)
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			result.StalePaths++
			continue
		}
		if session, ok := readWrapperSession(p); ok && s.tmux != nil && !s.tmux.SessionExists(session) {
			result.StalePaths++
		}
	}
}

func (s *Session) start(session, browserLogPath string, levels []string, hostPath string) error {
	if err := s.manifest.ValidateHostPath(hostPath); err != nil {
		return fmt.Errorf("untrusted host binary: %w", err)
	}

	// Self-heal: if a previous unclean shutdown left manifests pointing at a
	// missing wrapper, restore them to the real binary before we rewrite.
	_, _ = s.manifest.RepairStaleManifestPaths(hostPath)

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
		script = generateBatchScript(session, hostPath, absLogPath, levels, s.URLs, s.MaxLogBytes)
	} else {
		script = generateShellScript(session, hostPath, absLogPath, levels, s.URLs, s.MaxLogBytes)
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
		otherSession, ok := readWrapperSession(current)
		if !ok {
			otherSession, ok = sessionFromWrapperPath(current)
		}
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
		fmt.Sprintf("devlog-host-wrapper-%s-%08x%s", sanitizeSessionForFileName(session), sessionHash(session), browserHostWrapperExt()),
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

func sessionHash(session string) uint32 {
	sum := fnv.New32a()
	_, _ = sum.Write([]byte(session))
	return sum.Sum32()
}

func sessionMarker(session string) string {
	return hex.EncodeToString([]byte(session))
}

func readWrapperSession(path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for i := 0; i < 8 && scanner.Scan(); i++ {
		line := strings.TrimSpace(scanner.Text())
		var payload string
		switch {
		case strings.HasPrefix(line, "# devlog-session:"):
			payload = strings.TrimSpace(strings.TrimPrefix(line, "# devlog-session:"))
		case strings.HasPrefix(strings.ToLower(line), "rem devlog-session:"):
			payload = strings.TrimSpace(line[len("rem devlog-session:"):])
		default:
			continue
		}
		raw, decErr := hex.DecodeString(payload)
		if decErr != nil || len(raw) == 0 {
			continue
		}
		return string(raw), true
	}
	return "", false
}

func generateShellScript(session, hostPath, absLogPath string, levels, urls []string, maxBytes int64) string {
	args := []string{shellescape.Quote(hostPath), shellescape.Quote(absLogPath)}
	for _, level := range levels {
		args = append(args, shellescape.Quote(level))
	}
	args = append(args, hostFlagArgs(urls, maxBytes, shellescape.Quote)...)
	return fmt.Sprintf("#!/bin/sh\n# devlog-session: %s\nif ! tmux has-session -t %s 2>/dev/null; then\n  exit 0\nfi\nexec %s\n",
		sessionMarker(session), shellescape.Quote(session), strings.Join(args, " "))
}

func generateBatchScript(session, hostPath, absLogPath string, levels, urls []string, maxBytes int64) string {
	args := []string{batchQuote(hostPath), batchQuote(absLogPath)}
	for _, level := range levels {
		args = append(args, batchQuote(level))
	}
	args = append(args, hostFlagArgs(urls, maxBytes, batchQuote)...)
	return "@echo off\r\nrem devlog-session: " + sessionMarker(session) +
		"\r\ntmux has-session -t " + batchQuote(session) + " >nul 2>&1\r\nif errorlevel 1 exit /b 0\r\n" +
		strings.Join(args, " ") + "\r\n"
}

func hostFlagArgs(urls []string, maxBytes int64, quote func(string) string) []string {
	var args []string
	if maxBytes > 0 {
		args = append(args, "--max-bytes", quote(strconv.FormatInt(maxBytes, 10)))
	}
	for _, pattern := range urls {
		args = append(args, "--url", quote(pattern))
	}
	return args
}

// batchQuote returns a Windows batch-escaped argument using double quotes.
// % is still expanded inside quotes. Newlines would start a new command.
func batchQuote(s string) string {
	s = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "%", "%%").Replace(s)
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
