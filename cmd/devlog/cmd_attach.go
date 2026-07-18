package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/jellydn/devlog/internal/config"
	"github.com/jellydn/devlog/internal/tmux"
)

func cmdAttach(cfg *config.Config, args []string) error {
	runner := tmux.NewRunner(cfg.Tmux.Session)

	if !runner.SessionExists() {
		return fmt.Errorf("tmux session '%s' is not running. Run 'devlog up' first", cfg.Tmux.Session)
	}

	cmd := exec.Command("tmux", "attach", "-t", cfg.Tmux.Session)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
