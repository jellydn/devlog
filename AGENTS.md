# AGENTS.md

> Guidelines for AI agents working on the devlog project

## Build Commands

```bash
# Build CLI binary
go build -o devlog ./cmd/devlog

# Build native messaging host
go build -o devlog-host ./cmd/devlog-host

# Build both binaries and symlink to ~/.local/bin for easy testing
just devlog-dev

# Install both binaries locally
go install ./cmd/devlog
go install ./cmd/devlog-host

# Verify code compiles without building
just check

# Clean build artifacts
rm -f devlog devlog-host
```

## Test Commands

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run a single test by name
go test -run TestLoad_ValidConfig ./...
just test-one TestLoad_ValidConfig

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...
```

## Lint Commands

```bash
# Format code
go fmt ./...

# Check formatting without modifying files
gofmt -l .

# Vet code for issues
go vet ./...

# Full CI pipeline
just lint
just ci

# If golangci-lint is available
golangci-lint run
```

## Code Style Guidelines

### Imports

Standard library first, then third-party. No blank lines between groups.

```go
import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)
```

### Formatting, Types, Naming

Use `gofmt`. Tabs for indentation. 100 chars max. No trailing whitespace. Use structs with YAML tags. Pointer receivers for methods modifying structs.

- **Packages**: lowercase, no underscores (`config`, `tmux`)
- **Functions**: CamelCase (`Load`, `ResolveLogsDir`, `cmdUp`)
- **Variables**: camelCase (`configPath`, `logsDir`)
- **Tests**: `TestFunctionName_Description` (`TestLoad_ValidConfig`)

### Error Handling

Wrap errors: `fmt.Errorf("failed to read config: %w", err)`. Error messages start lowercase. Return errors in library code. Use `t.Fatalf` for test setup, `t.Errorf` for assertions.

### Comments

Use `// ` (space after slashes). Capitalize exported items. Document all exported types/functions.

### Testing

Table-driven tests with `tests := []struct{...}`. Use `t.Run(tt.name, func(t *testing.T) {...})`. Use `t.TempDir()`.

## Project Structure

```
.
├── cmd/
│   ├── devlog/           # CLI entry point
│   │   └── main.go
│   └── devlog-host/      # Native messaging host for browser integration
│       └── main.go
├── internal/
│   ├── config/           # Config loading/validation
│   │   ├── config.go
│   │   └── config_test.go
│   ├── tmux/             # Tmux session management
│   │   ├── tmux.go
│   │   └── tmux_test.go
│   ├── natmsg/           # Native messaging protocol
│   │   ├── natmsg.go
│   │   └── natmsg_test.go
│   └── logger/           # Logging utilities
│       ├── logger.go
│       └── logger_test.go
├── browser-extension/    # Shared extension source files
│   ├── background.js     # Native messaging communication
│   ├── content_script.js # Bridges page and background script
│   ├── page_inject.js    # Console capture logic
│   ├── popup.js/html     # Extension popup UI
├── chrome/               # Chrome extension (copies of shared files)
├── firefox/              # Firefox extension (copies of shared files)
├── go.mod
├── go.sum
└── devlog.yml.example    # Example configuration
```

## Dependencies

- Go 1.25+
- Minimal external dependencies (only `gopkg.in/yaml.v3`)
- Prefer standard library when possible

## Configuration

- YAML config files named `devlog.yml`
- Support environment variable interpolation (`$VAR` and `${VAR}`)
- Required fields: `version`, `project`, `tmux.session`, `windows`, `panes`, `cmd`

## Initialize Project

```bash
# Create devlog.yml from the example template
just init
```

## Browser Extension Maintenance

The browser extension uses shared source files in `browser-extension/` that are copied to both `chrome/` and `firefox/` directories. After editing shared files, sync them:

```bash
just sync-extensions
```
