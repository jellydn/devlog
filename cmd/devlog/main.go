package main

import (
	"fmt"
	"os"

	"github.com/jellydn/devlog/internal/config"
)

const usage = `Usage: devlog <command> [args...]

Commands:
  init        Create a devlog.yml template in current directory
  up          Start tmux session and browser logging
  down        Stop tmux session and flush logs
  attach      Attach to the running tmux session
  status      Show session state and log paths
  ls          List log runs
  open        Open logs directory in file manager
  register    Register native messaging host for browser logging
  healthcheck Check system requirements (tmux, browser extension)
  help        Show this help message

Examples:
  devlog init
  devlog healthcheck
  devlog up
  devlog attach
  devlog status
  devlog ls
  devlog down
  devlog register --chrome --extension-id abcdefghijklmnop
`

type Command func(cfg *config.Config, args []string) error

var commands = map[string]Command{
	"init":        cmdInit,
	"up":          cmdUp,
	"down":        cmdDown,
	"attach":      cmdAttach,
	"status":      cmdStatus,
	"ls":          cmdLs,
	"open":        cmdOpen,
	"help":        cmdHelp,
	"register":    cmdRegister,
	"healthcheck": cmdHealthcheck,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	command := os.Args[1]

	// Help command doesn't need config
	if command == "help" || command == "--help" || command == "-h" {
		cmdHelp(nil, nil)
		os.Exit(0)
	}

	// Commands that don't need config
	if command == "init" || command == "register" || command == "healthcheck" {
		runCommandWithoutConfig(command)
		return
	}

	// Find config file
	configPath := findConfigFile()
	if configPath == "" {
		fmt.Fprintln(os.Stderr, "Error: devlog.yml not found in current directory or parent directories")
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Execute command
	cmd, ok := commands[command]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n%s", command, usage)
		os.Exit(1)
	}

	if err := cmd(cfg, os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCommandWithoutConfig(command string) {
	cmd, ok := commands[command]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n%s", command, usage)
		os.Exit(1)
	}
	if err := cmd(nil, os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cmdHelp(cfg *config.Config, args []string) error {
	fmt.Print(usage)
	return nil
}
