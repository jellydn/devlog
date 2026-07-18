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

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// run is the testable entry point for the native messaging host.
// args are command-line arguments after the program name.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) < 1 {
		fmt.Fprint(stderr, usage)
		return fmt.Errorf("log file path is required")
	}

	logPath := args[0]
	levels := append([]string(nil), args[1:]...)

	// Convert levels to lowercase for case-insensitive matching
	for i, level := range levels {
		levels[i] = strings.ToLower(level)
	}

	// Create logger
	log, err := logger.New(logPath, levels)
	if err != nil {
		return fmt.Errorf("Error: failed to create logger: %v\n", err)
	}
	defer log.Close()

	return processMessages(log, natmsg.NewHostWithStreams(stdin, stdout), stderr)
}

// messageLogger is the subset of logger.Logger used by the host loop.
type messageLogger interface {
	Log(msg *natmsg.Message) error
}

// processMessages reads native messages until EOF and writes matching levels to the log.
func processMessages(log messageLogger, host *natmsg.Host, stderr io.Writer) error {
	for {
		msg, err := host.ReadMessage()
		if err != nil {
			if err == io.EOF {
				// Browser closed the connection, exit cleanly
				return nil
			}
			// Log error but continue processing
			fmt.Fprintf(stderr, "Error reading message: %v\n", err)
			host.SendAck(false, err.Error())
			continue
		}

		// Write message to log file
		if err := log.Log(msg); err != nil {
			fmt.Fprintf(stderr, "Error writing log: %v\n", err)
			host.SendAck(false, err.Error())
			continue
		}

		// Send acknowledgment
		if err := host.SendAck(true, ""); err != nil {
			fmt.Fprintf(stderr, "Error sending ack: %v\n", err)
			// Continue even if ack fails
		}
	}
}
