package main

import (
	"fmt"

	"github.com/jellydn/devlog/internal/config"
	"github.com/jellydn/devlog/internal/manifest"
)

func cmdRegister(cfg *config.Config, args []string) error {
	hostPath, err := manifest.FindDevlogHostBinary()
	if err != nil {
		return fmt.Errorf("failed to find devlog-host binary: %w", err)
	}

	installChrome := false
	installBrave := false
	installFirefox := false
	extensionID := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--chrome":
			installChrome = true
		case "--brave":
			installBrave = true
		case "--firefox":
			installFirefox = true
		case "--extension-id":
			if i+1 < len(args) {
				extensionID = args[i+1]
				i++
			} else {
				return fmt.Errorf("--extension-id requires a value")
			}
		case "--help", "-h":
			fmt.Print(`Usage: devlog register [options]

Register native messaging host for browser logging.

Options:
  --chrome         Register for Google Chrome
  --brave          Register for Brave Browser
  --firefox        Register for Mozilla Firefox
  --extension-id   Chrome/Brave extension ID (required for --chrome or --brave)
  --help, -h       Show this help message

Examples:
  devlog register --chrome --extension-id abcdefghijklmnopqrstuvwxyz123456
  devlog register --brave --extension-id abcdefghijklmnopqrstuvwxyz123456
  devlog register --firefox
  devlog register --chrome --brave --extension-id abcdefghijklmnopqrstuvwxyz123456
`)
			return nil
		default:
			return fmt.Errorf("unknown argument: %s (use --help for usage)", arg)
		}
	}

	if !installChrome && !installBrave && !installFirefox {
		if extensionID != "" {
			installChrome = true
		}
		installFirefox = true
	}

	if (installChrome || installBrave) && extensionID == "" {
		return fmt.Errorf("--extension-id is required when registering for Chrome or Brave")
	}

	fmt.Printf("devlog-host binary: %s\n", hostPath)

	if installChrome {
		fmt.Printf("Registering for Chrome...\n")
		if err := manifest.InstallChromeManifest(hostPath, extensionID); err != nil {
			return fmt.Errorf("failed to register Chrome manifest: %w", err)
		}
		dir := manifest.GetChromeNativeMessagingDir()
		fmt.Printf("  Installed to: %s\n", dir)
	}

	if installBrave {
		fmt.Printf("Registering for Brave...\n")
		if err := manifest.InstallBraveManifest(hostPath, extensionID); err != nil {
			return fmt.Errorf("failed to register Brave manifest: %w", err)
		}
		dir := manifest.GetBraveNativeMessagingDir()
		fmt.Printf("  Installed to: %s\n", dir)
	}

	if installFirefox {
		fmt.Printf("Registering for Firefox...\n")
		// Use extension ID if provided, otherwise use default
		firefoxExtID := extensionID
		if firefoxExtID == "" {
			firefoxExtID = "devlog@devlog.local"
		}
		if err := manifest.InstallFirefoxManifestWithID(hostPath, firefoxExtID); err != nil {
			return fmt.Errorf("failed to register Firefox manifest: %w", err)
		}
		dirs := manifest.GetFirefoxNativeMessagingDirs()
		fmt.Printf("  Installed to:\n")
		for _, dir := range dirs {
			fmt.Printf("    - %s\n", dir)
		}
		if extensionID != "" {
			fmt.Printf("  Extension ID: %s\n", extensionID)
		}
	}

	fmt.Println("Registration complete!")
	return nil
}
