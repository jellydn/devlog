// devlog-host is a native messaging host that receives browser console logs
// and writes them to disk. It communicates with browser extensions via stdin/stdout
// using the Native Messaging protocol.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jellydn/devlog/internal/logger"
	"github.com/jellydn/devlog/internal/natmsg"
)

const usage = `devlog-host - Native messaging host for browser console logs

Usage:
  devlog-host <log-file-path> [log-levels...]

Arguments:
  log-file-path   Path to the log file (required)
  log-levels      Space-separated list of log levels to capture
                  (e.g., log warn error). If not specified, all levels are captured.

Examples:
  devlog-host ./logs/browser.log
  devlog-host ./logs/browser.log log warn error

The host reads length-prefixed JSON messages from stdin and writes formatted
logs to the specified file. It runs until stdin is closed.
`

func filterBrowserArgs(args []string) []string {
	var filtered []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "chrome-extension://") || strings.HasPrefix(arg, "moz-extension://") {
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered
}

func main() {
	var logPath string
	var levels []string

	args := filterBrowserArgs(os.Args[1:])

	if len(args) >= 1 {
		logPath = args[0]
		levels = args[1:]
	} else {
		state, err := natmsg.ReadSessionState()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: no arguments provided and no active session found: %v\n", err)
			fmt.Fprint(os.Stderr, usage)
			os.Exit(1)
		}
		logPath = state.LogPath
		levels = state.Levels
	}

	for i, level := range levels {
		levels[i] = strings.ToLower(level)
	}

	log, err := logger.New(logPath, levels)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	host := natmsg.NewHost()

	for {
		msg, err := host.ReadMessage()
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "Error reading message: %v\n", err)
			host.SendAck(false, err.Error())
			continue
		}

		if err := log.Log(msg); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing log: %v\n", err)
			host.SendAck(false, err.Error())
			continue
		}

		if err := host.SendAck(true, ""); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending ack: %v\n", err)
		}
	}
}
