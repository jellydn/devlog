package main

import (
	"fmt"
	"strings"

	"github.com/jellydn/devlog/internal/browsersession"
	"github.com/jellydn/devlog/internal/config"
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

	bs := browsersession.New(manifestAdapter{}, tmuxSessionChecker{})
	result, err := bs.HealthCheck()
	if err != nil {
		return fmt.Errorf("browser healthcheck failed: %w", err)
	}

	// Check devlog-host binary
	fmt.Printf("%-*s ", maxLabelLen, "devlog-host binary:")
	if !result.HostFound {
		fmt.Println("✗ NOT FOUND")
		fmt.Println("  devlog-host is required for browser logging.")
		fmt.Println("  Install: go install github.com/jellydn/devlog/cmd/devlog-host@latest")
		allGood = false
	} else {
		fmt.Printf("✓ %s\n", result.HostPath)
	}

	// Check native messaging manifests
	fmt.Printf("%-*s ", maxLabelLen, "Browser extension:")
	if len(result.Registered) > 0 {
		fmt.Printf("✓ Registered for %s\n", strings.Join(result.Registered, ", "))
	} else {
		fmt.Println("✗ NOT REGISTERED")
		fmt.Println("  Browser extension is not registered.")
		fmt.Println("  Register: devlog register --chrome --extension-id <id>")
		fmt.Println("            devlog register --brave --extension-id <id>")
		fmt.Println("            devlog register --firefox")
		allGood = false
	}

	// Check that manifest path targets exist on disk (self-heal when possible)
	fmt.Printf("%-*s ", maxLabelLen, "Manifest host path:")
	if result.HostFound && result.RepairedPaths > 0 {
		fmt.Printf("✓ repaired %d stale path(s)\n", result.RepairedPaths)
	}
	if result.ManifestPaths == 0 {
		// Match prior behavior: when no paths and repair already printed a line,
		// still report "none installed" on its own line.
		fmt.Println("○ none installed")
	} else if result.StalePaths > 0 {
		fmt.Printf("✗ %d path(s) missing on disk\n", result.StalePaths)
		fmt.Println("  Run: devlog up  (or re-register) to repair")
		allGood = false
	} else {
		fmt.Printf("✓ %d path(s) exist\n", result.ManifestPaths)
	}

	fmt.Println()
	if allGood {
		fmt.Println("✓ All checks passed! You're ready to use devlog.")
		return nil
	}

	fmt.Println("⚠ Some checks failed. Please address the issues above.")
	return fmt.Errorf("healthcheck failed")
}
