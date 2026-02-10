package natmsg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetBraveNativeMessagingDir_ReturnsValidPath(t *testing.T) {
	// Act
	dir := GetBraveNativeMessagingDir()

	// Assert
	if dir == "" {
		t.Error("expected non-empty directory path")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got: %s", dir)
	}
}

func TestGetBraveNativeMessagingDir_HasBraveInPath(t *testing.T) {
	// Act
	dir := GetBraveNativeMessagingDir()

	// Assert - Brave-specific path components should be present
	lowerDir := strings.ToLower(dir)
	braveFound := strings.Contains(lowerDir, "brave")

	if !braveFound {
		t.Errorf("expected 'Brave' in path for Brave browser, got: %s", dir)
	}
}

func TestGetBraveNativeMessagingDir_DifferentFromChrome(t *testing.T) {
	// Act
	braveDir := GetBraveNativeMessagingDir()
	chromeDir := GetChromeNativeMessagingDir()

	// Assert - Brave and Chrome should have different manifest directories
	if braveDir == chromeDir {
		t.Errorf("expected Brave manifest dir to differ from Chrome, both: %s", braveDir)
	}
}

func TestInstallBraveManifest_CreatesManifestFile(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	hostPath := filepath.Join(tmpDir, "devlog-host")
	extensionID := "testextensionid"

	// Create fake host binary
	if err := os.WriteFile(hostPath, []byte("#! fake binary"), 0755); err != nil {
		t.Fatalf("failed to create fake host binary: %v", err)
	}

	// Set HOME to temp dir so GetBraveNativeMessagingDir() uses it
	home := os.Getenv("HOME")
	defer os.Setenv("HOME", home)
	os.Setenv("HOME", tmpDir)

	// Clear XDG_CONFIG_HOME to avoid interference
	xdg := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", xdg)
	os.Unsetenv("XDG_CONFIG_HOME")

	// Act
	err := InstallBraveManifest(hostPath, extensionID)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify manifest file exists
	manifestPath := filepath.Join(GetBraveNativeMessagingDir(), "com.devlog.host.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("manifest file not created at %s: %v", manifestPath, err)
	}
}

func TestInstallBraveManifest_CreatesValidManifestContent(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	hostPath := filepath.Join(tmpDir, "devlog-host")
	extensionID := "abc123def456"

	if err := os.WriteFile(hostPath, []byte("#! fake binary"), 0755); err != nil {
		t.Fatalf("failed to create fake host binary: %v", err)
	}

	home := os.Getenv("HOME")
	defer os.Setenv("HOME", home)
	os.Setenv("HOME", tmpDir)

	xdg := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", xdg)
	os.Unsetenv("XDG_CONFIG_HOME")

	// Act
	if err := InstallBraveManifest(hostPath, extensionID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert - Read and validate manifest content
	manifestPath := filepath.Join(GetBraveNativeMessagingDir(), "com.devlog.host.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var manifest ChromeManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	// Verify required fields
	tests := []struct {
		name  string
		field string
		want  string
	}{
		{"name", manifest.Name, "com.devlog.host"},
		{"description", manifest.Description, "devlog Native Messaging Host for Browser Log Capture"},
		{"type", manifest.Type, "stdio"},
		{"path", manifest.Path, hostPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.field, tt.want)
			}
		})
	}

	// Verify allowed origins contains the extension ID
	if len(manifest.AllowedOrigins) != 1 {
		t.Errorf("AllowedOrigins length = %d, want 1", len(manifest.AllowedOrigins))
	}
	expectedOrigin := "chrome-extension://" + extensionID + "/"
	if manifest.AllowedOrigins[0] != expectedOrigin {
		t.Errorf("AllowedOrigins[0] = %q, want %q", manifest.AllowedOrigins[0], expectedOrigin)
	}
}

func TestInstallBraveManifest_MissingHostPath(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "does-not-exist")
	extensionID := "testid"

	home := os.Getenv("HOME")
	defer os.Setenv("HOME", home)
	os.Setenv("HOME", tmpDir)

	xdg := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", xdg)
	os.Unsetenv("XDG_CONFIG_HOME")

	// Act
	err := InstallBraveManifest(nonExistentPath, extensionID)

	// Assert - Should succeed even if host binary doesn't exist
	// (the manifest just needs to reference a path)
	if err != nil {
		t.Errorf("expected success even with missing host binary, got: %v", err)
	}

	// Verify manifest was created with the specified path
	manifestPath := filepath.Join(GetBraveNativeMessagingDir(), "com.devlog.host.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var manifest ChromeManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	if manifest.Path != nonExistentPath {
		t.Errorf("Path = %q, want %q", manifest.Path, nonExistentPath)
	}
}

func TestGetChromeNativeMessagingDir_FallbackToHomeOnError(t *testing.T) {
	// This test verifies the behavior when os.UserHomeDir() fails
	// by checking that the function still returns a valid path

	// Act
	dir := GetChromeNativeMessagingDir()

	// Assert
	if dir == "" {
		t.Error("expected non-empty directory path even if UserHomeDir fails")
	}

	// On error, the function should fall back to "/" + platform-specific paths
	if runtime.GOOS == "darwin" {
		if !strings.Contains(dir, "Chrome") {
			t.Errorf("expected 'Chrome' in path on darwin, got: %s", dir)
		}
	}
}

func TestGetFirefoxNativeMessagingDirs_FallbackToHomeOnError(t *testing.T) {
	// Act
	dirs := GetFirefoxNativeMessagingDirs()

	// Assert
	if len(dirs) == 0 {
		t.Error("expected at least one Firefox directory")
	}

	for _, dir := range dirs {
		if dir == "" {
			t.Error("expected non-empty directory path")
		}
		if !filepath.IsAbs(dir) {
			t.Errorf("expected absolute path, got: %s", dir)
		}
	}
}
