package main

import (
	"fmt"
	"os"

	"devlog/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: devlog <command> [args...]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "up":
		fmt.Println("devlog up: Starting tmux session...")
	case "down":
		fmt.Println("devlog down: Stopping tmux session...")
	case "status":
		fmt.Println("devlog status: Checking status...")
	case "open":
		fmt.Println("devlog open: Opening log directory...")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}

	// Try to load config (for testing US-001)
	cfg, err := config.Load("devlog.yml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Config loaded: project=%s, session=%s\n", cfg.Project, cfg.Tmux.Session)
}
