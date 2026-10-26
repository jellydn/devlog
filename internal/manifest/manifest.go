// Package manifest manages browser native messaging host registration:
// installing, updating, and repairing manifest JSON files for Chrome,
// Brave, Firefox, and Zen browsers.
package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// userHomeDir is replaced in tests that need a missing home directory.
var userHomeDir = os.UserHomeDir

var chromeExtensionIDPattern = regexp.MustCompile(`^[a-p]{32}$`)
var firefoxExtensionIDPattern = regexp.MustCompile(`^(\{[0-9a-fA-F-]{36}\}|[A-Za-z0-9._+-]+@[A-Za-z0-9.-]+)$`)

func userHome() (string, error) {
	home, err := userHomeDir()
	if err != nil || home == "" || home == string(filepath.Separator) {
		return "", fmt.Errorf("user home directory is not available")
	}
	return home, nil
}

// ValidateChromeExtensionID checks the unpacked-extension id form Chrome uses.
func ValidateChromeExtensionID(id string) error {
	if !chromeExtensionIDPattern.MatchString(id) {
		return fmt.Errorf("chrome extension id %q must be 32 characters in the range a-p", id)
	}
	return nil
}

// ValidateFirefoxExtensionID checks an email-like or brace-UUID extension id.
func ValidateFirefoxExtensionID(id string) error {
	if !firefoxExtensionIDPattern.MatchString(id) {
		return fmt.Errorf("firefox extension id %q must be an email-like id or a brace UUID", id)
	}
	return nil
}

type ChromeManifest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	AllowedOrigins []string `json:"allowed_origins"`
}

type FirefoxManifest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Path              string   `json:"path"`
	Type              string   `json:"type"`
	AllowedExtensions []string `json:"allowed_extensions"`
}

func GetChromeNativeMessagingDir() string {
	switch runtime.GOOS {
	case "windows":
		// APPDATA does not depend on the home directory.
		return filepath.Join(os.Getenv("APPDATA"), "Google", "Chrome", "NativeMessagingHosts")
	case "darwin":
		home, err := userHome()
		if err != nil {
			return ""
		}
		return filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "NativeMessagingHosts")
	default:
		home, err := userHome()
		if err != nil {
			return ""
		}
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(home, ".config")
		}
		return filepath.Join(xdgConfig, "google-chrome", "NativeMessagingHosts")
	}
}

func GetBraveNativeMessagingDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "BraveSoftware", "Brave-Browser", "NativeMessagingHosts")
	case "darwin":
		home, err := userHome()
		if err != nil {
			return ""
		}
		return filepath.Join(home, "Library", "Application Support", "BraveSoftware", "Brave-Browser", "NativeMessagingHosts")
	default:
		home, err := userHome()
		if err != nil {
			return ""
		}
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(home, ".config")
		}
		return filepath.Join(xdgConfig, "BraveSoftware", "Brave-Browser", "NativeMessagingHosts")
	}
}

func GetFirefoxNativeMessagingDir() string {
	dirs := getFirefoxDirs()
	if len(dirs) == 0 {
		return ""
	}
	return dirs[0]
}

func GetFirefoxNativeMessagingDirs() []string {
	return getFirefoxDirs()
}

func getFirefoxDirs() []string {
	home, homeErr := userHome()
	switch runtime.GOOS {
	case "darwin":
		if homeErr != nil {
			return nil
		}
		return []string{
			filepath.Join(home, "Library", "Application Support", "Mozilla", "NativeMessagingHosts"),
			filepath.Join(home, "Library", "Application Support", "zen", "NativeMessagingHosts"),
		}
	case "windows":
		appdata := os.Getenv("APPDATA")
		return []string{
			filepath.Join(appdata, "Mozilla", "NativeMessagingHosts"),
			filepath.Join(appdata, "zen", "NativeMessagingHosts"),
		}
	default:
		if homeErr != nil {
			return nil
		}
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(home, ".config")
		}
		return []string{
			filepath.Join(xdgConfig, "mozilla", "native-messaging-hosts"),
			filepath.Join(xdgConfig, "zen", "native-messaging-hosts"),
		}
	}
}

// installChromiumManifest writes a Chromium-style native messaging manifest (used by
// Chrome and Brave) into dir. The label is used for error messages only.
func installChromiumManifest(dir, hostPath, extensionID, label string) error {
	if extensionID == "" {
		return fmt.Errorf("%s extension ID is required", label)
	}
	if err := ValidateHostPath(hostPath); err != nil {
		return err
	}
	if err := ValidateChromeExtensionID(extensionID); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create %s native messaging directory: %w", label, err)
	}

	manifest := ChromeManifest{
		Name:           "com.devlog.host",
		Description:    "devlog Native Messaging Host for Browser Log Capture",
		Path:           hostPath,
		Type:           "stdio",
		AllowedOrigins: []string{fmt.Sprintf("chrome-extension://%s/", extensionID)},
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal %s manifest: %w", label, err)
	}

	manifestPath := filepath.Join(dir, ManifestFileName)
	if err := os.WriteFile(manifestPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write %s manifest: %w", label, err)
	}

	return nil
}

func InstallChromeManifest(hostPath string, extensionID string) error {
	return installChromiumManifest(GetChromeNativeMessagingDir(), hostPath, extensionID, "Chrome")
}

func InstallBraveManifest(hostPath string, extensionID string) error {
	return installChromiumManifest(GetBraveNativeMessagingDir(), hostPath, extensionID, "Brave")
}

func InstallFirefoxManifest(hostPath string) error {
	return InstallFirefoxManifestWithID(hostPath, "devlog@devlog.local")
}

func InstallFirefoxManifestWithID(hostPath string, extensionID string) error {
	if extensionID == "" {
		return fmt.Errorf("Firefox extension ID is required")
	}
	if err := ValidateFirefoxExtensionID(extensionID); err != nil {
		return err
	}
	if err := ValidateHostPath(hostPath); err != nil {
		return err
	}
	// A single-element slice covers both standard add-on IDs (e.g. "devlog@devlog.local")
	// and UUID-format IDs (e.g. "{xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx}") that Firefox
	// generates for unpacked temporary extensions.
	allowedExts := []string{extensionID}

	manifest := FirefoxManifest{
		Name:              "com.devlog.host",
		Description:       "devlog Native Messaging Host for Browser Log Capture",
		Path:              hostPath,
		Type:              "stdio",
		AllowedExtensions: allowedExts,
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Firefox manifest: %w", err)
	}

	for _, dir := range getFirefoxDirs() {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create native messaging directory %s: %w", dir, err)
		}
		manifestPath := filepath.Join(dir, "com.devlog.host.json")
		if err := os.WriteFile(manifestPath, data, 0600); err != nil {
			return fmt.Errorf("failed to write manifest to %s: %w", dir, err)
		}
	}

	return nil
}

func UpdateManifestPath(newPath string) error {
	if err := ValidateHostPath(newPath); err != nil {
		return err
	}
	dirs := ManifestDirs()
	manifestFile := "com.devlog.host.json"
	updated := false
	var updateErrors []string

	for _, dir := range dirs {
		manifestPath := filepath.Join(dir, manifestFile)
		if err := rewriteManifestPath(manifestPath, newPath); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				updateErrors = append(updateErrors, fmt.Sprintf("%s: %v", manifestPath, err))
			}
			continue
		}
		updated = true
	}

	if len(updateErrors) > 0 {
		if updated {
			// Some manifests were updated, but others failed
			return fmt.Errorf("partially updated: some manifests succeeded but others failed: %s", strings.Join(updateErrors, "; "))
		}
		// All manifests that were found failed to update
		return fmt.Errorf("failed to update native messaging manifests: %s", strings.Join(updateErrors, "; "))
	}

	if !updated {
		return fmt.Errorf("no native messaging manifests found to update (run 'devlog register' first)")
	}
	return nil
}

func IsManifestPathInUse(targetPath string) (bool, error) {
	dirs := ManifestDirs()
	manifestFile := ManifestFileName
	targetPath = filepath.Clean(targetPath)
	foundManifest := false
	var queryErrors []string

	for _, dir := range dirs {
		manifestPath := filepath.Join(dir, manifestFile)
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				queryErrors = append(queryErrors, fmt.Sprintf("%s: %v", manifestPath, err))
			}
			continue
		}
		foundManifest = true

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			queryErrors = append(queryErrors, fmt.Sprintf("%s: invalid JSON: %v", manifestPath, err))
			continue
		}

		path, ok := raw["path"].(string)
		if !ok {
			continue
		}

		if filepath.Clean(path) == targetPath {
			return true, nil
		}
	}

	if len(queryErrors) > 0 {
		return false, fmt.Errorf("failed to inspect one or more native messaging manifests: %s", strings.Join(queryErrors, "; "))
	}

	if !foundManifest {
		return false, fmt.Errorf("no native messaging manifests found (run 'devlog register' first)")
	}

	return false, nil
}

func FindDevlogHostBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	dir := filepath.Dir(exe)

	names := []string{"devlog-host"}
	if runtime.GOOS == "windows" {
		names = append(names, "devlog-host.exe")
	}

	for _, name := range names {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("devlog-host binary not found in %s (searched for: %s)", dir, strings.Join(names, ", "))
}

// ManifestFileName is the native messaging host registration filename.
const ManifestFileName = "com.devlog.host.json"

// ManifestDirs returns all known native messaging host directories for supported browsers.
func ManifestDirs() []string {
	return append([]string{GetChromeNativeMessagingDir(), GetBraveNativeMessagingDir()}, getFirefoxDirs()...)
}

// ReadManifestPaths returns the path field from each installed com.devlog.host.json.
// Missing manifests are skipped. Returns an error only if a present manifest cannot be read/parsed.
func ReadManifestPaths() (map[string]string, error) {
	paths := make(map[string]string)
	var errs []string
	for _, dir := range ManifestDirs() {
		manifestPath := filepath.Join(dir, ManifestFileName)
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			errs = append(errs, fmt.Sprintf("%s: %v", manifestPath, err))
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			errs = append(errs, fmt.Sprintf("%s: invalid JSON: %v", manifestPath, err))
			continue
		}
		p, ok := raw["path"].(string)
		if !ok || p == "" {
			errs = append(errs, fmt.Sprintf("%s: missing path field", manifestPath))
			continue
		}
		paths[manifestPath] = p
	}
	if len(errs) > 0 {
		return paths, fmt.Errorf("failed to read some manifests: %s", strings.Join(errs, "; "))
	}
	return paths, nil
}

// rewriteManifestPath reads a manifest JSON from manifestPath, updates its
// "path" field to newPath, and writes it back. ReadFile errors (including
// os.ErrNotExist) are returned unwrapped so callers can use errors.Is.
func rewriteManifestPath(manifestPath, newPath string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	raw["path"] = newPath
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}
	if err := os.WriteFile(manifestPath, out, 0600); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}
	return nil
}

// RepairStaleManifestPaths rewrites any installed manifest whose path does not exist
// back to hostPath (typically the real devlog-host binary).
func RepairStaleManifestPaths(hostPath string) (repaired int, err error) {
	if err := ValidateHostPath(hostPath); err != nil {
		return 0, err
	}
	paths, readErr := ReadManifestPaths()
	for manifestPath, current := range paths {
		if _, statErr := os.Stat(current); statErr == nil {
			continue
		}
		if err := rewriteManifestPath(manifestPath, hostPath); err != nil {
			continue
		}
		repaired++
	}
	if readErr != nil && repaired == 0 {
		return repaired, readErr
	}
	return repaired, nil
}
