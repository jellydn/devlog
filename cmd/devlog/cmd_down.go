package main

import (
	"fmt"

	"github.com/jellydn/devlog/internal/browsersession"
	"github.com/jellydn/devlog/internal/config"
	"github.com/jellydn/devlog/internal/tmux"
)

func cmdDown(cfg *config.Config, args []string) error {
	fmt.Printf("Stopping devlog session '%s'...\n", cfg.Tmux.Session)

	// Create tmux runner
	runner := tmux.NewRunner(cfg.Tmux.Session)
	bs := browsersession.New(manifestAdapter{}, tmuxSessionChecker{})

	// Check if session exists
	if !runner.SessionExists() {
		bs.Stop(cfg.Tmux.Session)
		return fmt.Errorf("tmux session '%s' does not exist", cfg.Tmux.Session)
	}

	// Kill the session
	if err := runner.KillSession(); err != nil {
		return err
	}

	// Restore native messaging manifest to point to the real binary
	bs.Stop(cfg.Tmux.Session)

	fmt.Printf("Stopped tmux session '%s'\n", cfg.Tmux.Session)

	return nil
}
