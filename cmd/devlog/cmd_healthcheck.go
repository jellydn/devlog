package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jellydn/devlog/internal/config"
	"github.com/jellydn/devlog/internal/natmsg"
	"github.com/jellydn/devlog/internal/tmux"
)

func cmdHealthcheck(cfg *config.Config, args []string) error {
	const maxLabelLen = 22

	fmt.Println("devlog healthcheck")
	fmt.Println("==================")
	fmt.Println()

	allGood := true

	// Check tmux
	fmt.Printf("%-*s ", maxLabelLen, "tmux:")
	version, err := tmux.CheckVersion()
	if err != nil {
		fmt.Println("✗ NOT FOUND")
		fmt.Println("  tmux is required to run devlog.")
		fmt.Println("  Install: https://github.com/tmux/tmux/wiki/Installing")
		allGood = false
	} else {
		fmt.Printf("✓ %s\n", version)
	}

	// Check devlog-host binary
	fmt.Printf("%-*s ", maxLabelLen, "devlog-host binary:")
	hostPath, err := natmsg.FindDevlogHostBinary()
	if err != nil {
		fmt.Println("✗ NOT FOUND")
		fmt.Println("  devlog-host is required for browser logging.")
		fmt.Println("  Install: go install github.com/jellydn/devlog/cmd/devlog-host@latest")
		allGood = false
	} else {
		fmt.Printf("✓ %s\n", hostPath)
	}

	// Check native messaging manifests
	fmt.Printf("%-*s ", maxLabelLen, "Browser extension:")
	chromeManifestPath := filepath.Join(natmsg.GetChromeNativeMessagingDir(), "com.devlog.host.json")
	braveManifestPath := filepath.Join(natmsg.GetBraveNativeMessagingDir(), "com.devlog.host.json")
	firefoxManifestPaths := []string{}
	for _, dir := range natmsg.GetFirefoxNativeMessagingDirs() {
		firefoxManifestPaths = append(firefoxManifestPaths, filepath.Join(dir, "com.devlog.host.json"))
	}

	chromeRegistered := false
	if _, err := os.Stat(chromeManifestPath); err == nil {
		chromeRegistered = true
	}

	braveRegistered := false
	if _, err := os.Stat(braveManifestPath); err == nil {
		braveRegistered = true
	}

	firefoxRegistered := false
	for _, path := range firefoxManifestPaths {
		if _, err := os.Stat(path); err == nil {
			firefoxRegistered = true
			break
		}
	}

	if chromeRegistered || braveRegistered || firefoxRegistered {
		registered := []string{}
		if chromeRegistered {
			registered = append(registered, "Chrome")
		}
		if braveRegistered {
			registered = append(registered, "Brave")
		}
		if firefoxRegistered {
			registered = append(registered, "Firefox")
		}
		fmt.Printf("✓ Registered for %s\n", strings.Join(registered, ", "))
	} else {
		fmt.Println("✗ NOT REGISTERED")
		fmt.Println("  Browser extension is not registered.")
		fmt.Println("  Register: devlog register --chrome --extension-id <id>")
		fmt.Println("            devlog register --brave --extension-id <id>")
		fmt.Println("            devlog register --firefox")
		allGood = false
	}

	fmt.Println()
	if allGood {
		fmt.Println("✓ All checks passed! You're ready to use devlog.")
		return nil
	}

	fmt.Println("⚠ Some checks failed. Please address the issues above.")
	return fmt.Errorf("healthcheck failed")
}
