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

	// Kill the session, then always restore the manifest. A failed kill must
	// not leave the browser pointed at this session's wrapper.
	err := runner.KillSession()
	bs.Stop(cfg.Tmux.Session)
	if err != nil {
		return err
	}

	fmt.Printf("Stopped tmux session '%s'\n", cfg.Tmux.Session)

	return nil
}
