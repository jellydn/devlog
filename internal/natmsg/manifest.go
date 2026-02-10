package natmsg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

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
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Google", "Chrome", "NativeMessagingHosts")
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Google", "Chrome", "NativeMessagingHosts")
	default:
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(os.Getenv("HOME"), ".config")
		}
		return filepath.Join(xdgConfig, "google-chrome", "NativeMessagingHosts")
	}
}

func GetFirefoxNativeMessagingDir() string {
	return getFirefoxDirs()[0]
}

func getFirefoxDirs() []string {
	home := os.Getenv("HOME")
	switch runtime.GOOS {
	case "darwin":
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

func InstallChromeManifest(hostPath string, extensionID string) error {
	dir := GetChromeNativeMessagingDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create Chrome native messaging directory: %w", err)
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
		return fmt.Errorf("failed to marshal Chrome manifest: %w", err)
	}

	manifestPath := filepath.Join(dir, "com.devlog.host.json")
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write Chrome manifest: %w", err)
	}

	return nil
}

func InstallFirefoxManifest(hostPath string) error {
	manifest := FirefoxManifest{
		Name:              "com.devlog.host",
		Description:       "devlog Native Messaging Host for Browser Log Capture",
		Path:              hostPath,
		Type:              "stdio",
		AllowedExtensions: []string{"devlog@devlog.local"},
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
		if err := os.WriteFile(manifestPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write manifest to %s: %w", dir, err)
		}
	}

	return nil
}

func UpdateManifestPath(newPath string) error {
	dirs := append([]string{GetChromeNativeMessagingDir()}, getFirefoxDirs()...)
	manifestFile := "com.devlog.host.json"
	updated := false

	for _, dir := range dirs {
		manifestPath := filepath.Join(dir, manifestFile)
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}

		raw["path"] = newPath
		out, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			continue
		}

		if err := os.WriteFile(manifestPath, out, 0644); err != nil {
			continue
		}
		updated = true
	}

	if !updated {
		return fmt.Errorf("no native messaging manifests found to update (run 'devlog register' first)")
	}
	return nil
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
