package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jellydn/devlog/internal/manifest"
)

func withIsolatedHome(t *testing.T) (home string, cleanup func()) {
	t.Helper()
	home = t.TempDir()
	prevHome := os.Getenv("HOME")
	prevXDG := os.Getenv("XDG_CONFIG_HOME")
	prevCache := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("HOME", home)
	os.Unsetenv("XDG_CONFIG_HOME")
	// Point cache under home so wrapper paths are isolated
	cache := filepath.Join(home, "Library", "Caches")
	if runtime.GOOS != "darwin" {
		cache = filepath.Join(home, ".cache")
		os.Setenv("XDG_CACHE_HOME", cache)
	} else {
		_ = os.MkdirAll(cache, 0700)
	}
	cleanup = func() {
		os.Setenv("HOME", prevHome)
		if prevXDG == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", prevXDG)
		}
		if prevCache == "" {
			os.Unsetenv("XDG_CACHE_HOME")
		} else {
			os.Setenv("XDG_CACHE_HOME", prevCache)
		}
	}
	return home, cleanup
}

func readChromePath(t *testing.T) string {
	t.Helper()
	manifestPath := filepath.Join(manifest.GetChromeNativeMessagingDir(), manifest.ManifestFileName)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p, _ := raw["path"].(string)
	return p
}

func TestBrowserHostWrapper_RoundTrip(t *testing.T) {
	_, cleanup := withIsolatedHome(t)
	defer cleanup()

	tmp := t.TempDir()
	hostPath := filepath.Join(tmp, "devlog-host")
	if err := os.WriteFile(hostPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(tmp, "browser.log")

	if err := manifest.InstallChromeManifest(hostPath, "testid"); err != nil {
		t.Fatalf("install: %v", err)
	}

	if err := writeBrowserHostWrapperWithHost("test-session", logPath, []string{"error", "warn"}, hostPath); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	wrapperPath := browserHostWrapperPath("test-session")
	if _, err := os.Stat(wrapperPath); err != nil {
		t.Fatalf("wrapper missing: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(wrapperPath)
		if info.Mode().Perm() != 0700 {
			t.Errorf("wrapper mode = %o, want 0700", info.Mode().Perm())
		}
	}
	if got := readChromePath(t); got != wrapperPath {
		t.Errorf("manifest path = %q, want wrapper %q", got, wrapperPath)
	}

	restoreBrowserHostWrapperWithHost("test-session", hostPath)

	if _, err := os.Stat(wrapperPath); !os.IsNotExist(err) {
		t.Errorf("wrapper should be deleted after restore")
	}
	if got := readChromePath(t); got != hostPath {
		t.Errorf("manifest path after restore = %q, want host %q", got, hostPath)
	}
}

func TestBrowserHostWrapper_StaleRecovery(t *testing.T) {
	_, cleanup := withIsolatedHome(t)
	defer cleanup()

	tmp := t.TempDir()
	hostPath := filepath.Join(tmp, "devlog-host")
	if err := os.WriteFile(hostPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(tmp, "browser.log")

	if err := manifest.InstallChromeManifest(hostPath, "testid"); err != nil {
		t.Fatalf("install: %v", err)
	}

	// First up
	if err := writeBrowserHostWrapperWithHost("sess-a", logPath, nil, hostPath); err != nil {
		t.Fatalf("first write: %v", err)
	}
	wrapperPath := browserHostWrapperPath("sess-a")

	// Simulate unclean shutdown: delete wrapper, leave manifest pointing at it
	if err := os.Remove(wrapperPath); err != nil {
		t.Fatal(err)
	}
	if got := readChromePath(t); got != wrapperPath {
		t.Fatalf("precondition: manifest should still point at wrapper, got %q", got)
	}

	// Second up should self-heal and succeed
	if err := writeBrowserHostWrapperWithHost("sess-a", logPath, []string{"error"}, hostPath); err != nil {
		t.Fatalf("stale recovery write: %v", err)
	}
	if _, err := os.Stat(wrapperPath); err != nil {
		t.Fatalf("wrapper not recreated: %v", err)
	}
	if got := readChromePath(t); got != wrapperPath {
		t.Errorf("manifest path = %q, want %q", got, wrapperPath)
	}
}

func TestSessionFromWrapperPath(t *testing.T) {
	tests := []struct {
		path string
		want string
		ok   bool
	}{
		{filepath.Join("/tmp", "devlog", "wrappers", "devlog-host-wrapper-myapp.sh"), "myapp", true},
		{filepath.Join("/tmp", "devlog", "wrappers", "devlog-host-wrapper-myapp.bat"), "myapp", true},
		{filepath.Join("/tmp", "devlog-host"), "", false},
		{filepath.Join("/tmp", "devlog", "wrappers", "devlog-host-wrapper-.sh"), "", false},
	}
	for _, tt := range tests {
		got, ok := sessionFromWrapperPath(tt.path)
		if ok != tt.ok || got != tt.want {
			t.Errorf("sessionFromWrapperPath(%q) = (%q, %v), want (%q, %v)", tt.path, got, ok, tt.want, tt.ok)
		}
	}
}

func TestRefuseClobberActiveWrapper_SameSessionAllowed(t *testing.T) {
	_, cleanup := withIsolatedHome(t)
	defer cleanup()

	tmp := t.TempDir()
	hostPath := filepath.Join(tmp, "devlog-host")
	if err := os.WriteFile(hostPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := manifest.InstallChromeManifest(hostPath, "testid"); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(tmp, "b.log")
	if err := writeBrowserHostWrapperWithHost("same", logPath, nil, hostPath); err != nil {
		t.Fatal(err)
	}
	// Rewriting for the same session should be allowed even without a live tmux session
	if err := writeBrowserHostWrapperWithHost("same", logPath, []string{"error"}, hostPath); err != nil {
		t.Fatalf("same session rewrite should be allowed: %v", err)
	}
}

func TestRefuseClobberActiveWrapper_OtherLiveSession(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	_, cleanup := withIsolatedHome(t)
	defer cleanup()

	tmp := t.TempDir()
	hostPath := filepath.Join(tmp, "devlog-host")
	if err := os.WriteFile(hostPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := manifest.InstallChromeManifest(hostPath, "testid"); err != nil {
		t.Fatal(err)
	}

	// Create a live tmux session named other-live
	session := "other-live"
	_ = exec.Command("tmux", "kill-session", "-t", session).Run()
	if err := exec.Command("tmux", "new-session", "-d", "-s", session).Run(); err != nil {
		t.Skipf("could not create tmux session: %v", err)
	}
	defer exec.Command("tmux", "kill-session", "-t", session).Run()

	logPath := filepath.Join(tmp, "b.log")
	if err := writeBrowserHostWrapperWithHost(session, logPath, nil, hostPath); err != nil {
		t.Fatalf("setup other session wrapper: %v", err)
	}

	// Attempt to clobber from a different session while other is live
	err := writeBrowserHostWrapperWithHost("new-session", logPath, nil, hostPath)
	if err == nil {
		t.Fatal("expected clobber refusal when other session is live")
	}
	if !strings.Contains(err.Error(), "already in use") {
		t.Errorf("error = %q, want already in use", err.Error())
	}
}
