//go:build e2e

// Package e2e contains end-to-end CLI smoke tests that exercise the built
// devlog binary against a real tmux session. These tests require tmux
// installed on PATH and are intended for CI or local pre-release verification.
package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// binary is the path to the built devlog binary, set by TestMain.
var binary string

func TestMain(m *testing.M) {
	// Build the CLI binary to a temp location.
	tmp, err := os.MkdirTemp("", "devlog-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	binary = filepath.Join(tmp, "devlog")
	build := exec.Command("go", "build", "-o", binary, "../../cmd/devlog")
	build.Stderr = os.Stderr
	build.Stdout = os.Stdout
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build devlog binary: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	os.RemoveAll(tmp)
	os.Exit(code)
}

func skipIfNoTmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available in PATH")
	}
}

// runDevlog executes the devlog binary with the given args from dir.
func runDevlog(t *testing.T, dir string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// sessionName returns a unique session name for test isolation.
func sessionName() string {
	return fmt.Sprintf("devlog-e2e-%d", time.Now().UnixNano())
}

// writeConfig writes a minimal devlog.yml for the given session and log dir.
func writeConfig(t *testing.T, dir, session, logsDir string) {
	t.Helper()

	config := fmt.Sprintf(`version: "1.0"
project: e2e-test
logs_dir: %s
run_mode: overwrite
tmux:
  session: %s
  windows:
    - name: test
      panes:
        - cmd: echo "hello-from-e2e"
          log: test.log
        - cmd: echo "pane-two"
          log: pane2.log
`, logsDir, session)

	path := filepath.Join(dir, "devlog.yml")
	if err := os.WriteFile(path, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}

// TestE2E_Lifecycle tests the full devlog lifecycle: up → status → down.
func TestE2E_Lifecycle(t *testing.T) {
	skipIfNoTmux(t)

	session := sessionName()
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	writeConfig(t, tmpDir, session, logsDir)

	// 1. devlog up
	stdout, stderr, err := runDevlog(t, tmpDir, "up")
	if err != nil {
		t.Fatalf("devlog up failed: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
	}
	if !strings.Contains(stdout, "Starting devlog session") {
		t.Errorf("up output should mention starting session, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Created tmux session") {
		t.Errorf("up output should mention created session, got: %s", stdout)
	}

	// Ensure cleanup even if subsequent assertions fail.
	defer func() {
		runDevlog(t, tmpDir, "down")
	}()

	// 2. devlog status
	stdout, stderr, err = runDevlog(t, tmpDir, "status")
	if err != nil {
		t.Fatalf("devlog status failed: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
	}
	if !strings.Contains(stdout, "Status: Running") {
		t.Errorf("status should show Running, got: %s", stdout)
	}

	// 3. devlog down
	stdout, stderr, err = runDevlog(t, tmpDir, "down")
	if err != nil {
		t.Fatalf("devlog down failed: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
	}
	if !strings.Contains(stdout, "Stopped tmux session") {
		t.Errorf("down output should mention stopped session, got: %s", stdout)
	}

	// 4. devlog status after down
	stdout, stderr, err = runDevlog(t, tmpDir, "status")
	if err != nil {
		t.Fatalf("devlog status after down failed: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
	}
	if !strings.Contains(stdout, "Not running") {
		t.Errorf("status after down should show Not running, got: %s", stdout)
	}

	// 5. Verify log file was created
	time.Sleep(100 * time.Millisecond)
	logPath := filepath.Join(logsDir, "test.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(data), "hello-from-e2e") {
		t.Errorf("log should contain 'hello-from-e2e', got: %s", string(data))
	}
}

// TestE2E_UpFailsWhenAlreadyRunning verifies that devlog up errors when a
// session is already active.
func TestE2E_UpFailsWhenAlreadyRunning(t *testing.T) {
	skipIfNoTmux(t)

	session := sessionName()
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	writeConfig(t, tmpDir, session, logsDir)

	// First up should succeed.
	_, stderr, err := runDevlog(t, tmpDir, "up")
	if err != nil {
		t.Fatalf("first devlog up failed: %v\nstderr: %s", err, stderr)
	}

	defer func() {
		runDevlog(t, tmpDir, "down")
	}()

	// Second up should fail.
	_, stderr, err = runDevlog(t, tmpDir, "up")
	if err == nil {
		t.Error("second devlog up should have failed")
	}
	if !strings.Contains(stderr, "already exists") {
		t.Errorf("up error should mention 'already exists', got: %s", stderr)
	}
}

// TestE2E_Healthcheck verifies that devlog healthcheck detects tmux and
// reports results.
func TestE2E_Healthcheck(t *testing.T) {
	skipIfNoTmux(t)

	tmpDir := t.TempDir()

	// Healthcheck may return an error when devlog-host is not installed
	// (which is normal on CI). We only verify that tmux detection works.
	stdout, _, _ := runDevlog(t, tmpDir, "healthcheck")

	// Healthcheck should detect tmux.
	if !strings.Contains(stdout, "tmux:") {
		t.Errorf("healthcheck should check tmux, got: %s", stdout)
	}
}

// TestE2E_Init verifies that devlog init creates a devlog.yml in the current
// directory.
func TestE2E_Init(t *testing.T) {
	tmpDir := t.TempDir()

	stdout, stderr, err := runDevlog(t, tmpDir, "init")
	if err != nil {
		t.Fatalf("devlog init failed: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
	}
	if !strings.Contains(stdout, "Created devlog.yml") {
		t.Errorf("init should mention Created devlog.yml, got: %s", stdout)
	}

	// Verify the file actually exists.
	configPath := filepath.Join(tmpDir, "devlog.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("devlog.yml was not created")
	}

	// Read it back and check it has content.
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read created config: %v", err)
	}
	if len(data) == 0 {
		t.Error("created devlog.yml is empty")
	}
}

// TestE2E_DownWithoutSession verifies that devlog down errors gracefully when
// no session is running.
func TestE2E_DownWithoutSession(t *testing.T) {
	skipIfNoTmux(t)

	session := sessionName()
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	writeConfig(t, tmpDir, session, logsDir)

	_, stderr, err := runDevlog(t, tmpDir, "down")
	if err == nil {
		t.Error("devlog down without session should fail")
	}
	if !strings.Contains(stderr, "does not exist") {
		t.Errorf("down error should mention 'does not exist', got: %s", stderr)
	}
}

// TestE2E_TimestampedMode verifies the timestamped run mode creates a
// time-stamped log directory.
func TestE2E_TimestampedMode(t *testing.T) {
	skipIfNoTmux(t)

	session := sessionName()
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	config := fmt.Sprintf(`version: "1.0"
project: e2e-test
logs_dir: %s
run_mode: timestamped
tmux:
  session: %s
  windows:
    - name: test
      panes:
        - cmd: echo "timestamped"
          log: test.log
`, logsDir, session)

	configPath := filepath.Join(tmpDir, "devlog.yml")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	stdout, stderr, err := runDevlog(t, tmpDir, "up")
	if err != nil {
		t.Fatalf("devlog up failed: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
	}

	defer func() {
		runDevlog(t, tmpDir, "down")
	}()

	// Verify the logs dir has a timestamped subdirectory.
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		t.Fatalf("failed to read logs dir: %v", err)
	}

	found := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Timestamped dirs have format YYYYMMDD-HHMMSS (15 chars).
		if len(entry.Name()) == 15 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a timestamped subdirectory in %s, got: %v", logsDir, entryNames(entries))
	}
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names
}
