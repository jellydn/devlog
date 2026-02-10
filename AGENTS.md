# AGENTS.md

> Guidelines for AI agents working on the devlog project

## Build Commands

```bash
# Build CLI binary
go build -o devlog ./cmd/devlog

# Build native messaging host
go build -o devlog-host ./cmd/devlog-host

# Install both binaries locally
go install ./cmd/devlog
go install ./cmd/devlog-host

# Clean build artifacts
rm -f devlog devlog-host
```

## Test Commands

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run a single test
go test -run TestLoad_ValidConfig ./...

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...

# Run integration tests
go test -tags=integration ./internal/tmux/

# Run integration tests with verbose output
go test -v -tags=integration ./internal/tmux/

# Skip integration tests (for fast development)
go test -short -tags=integration ./internal/tmux/
```

### Integration Tests

Integration tests for tmux operations are located in `internal/tmux/integration_test.go` and use the build tag `integration`. These tests:

- Require a real tmux installation
- Create and destroy actual tmux sessions
- Verify log file creation and content
- Test edge cases like special characters and long session names
- Are automatically skipped with the `-short` flag

Run integration tests locally:
```bash
just test-integration      # Run integration tests
just test-integration-v    # Run with verbose output
```

CI automatically runs both unit tests and integration tests.

## Lint Commands

```bash
# Format code
go fmt ./...

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

Standard library first, then third-party. No blank lines between groups. Use full import paths.

```go
import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)
```

### Formatting

Use `gofmt` for formatting. Tabs for indentation. Aim for 100 characters max. No trailing whitespace.

### Types

Use structs with YAML tags (`yaml:"field_name"`). Use pointer receivers for methods that modify the struct. Define custom types for command functions: `type Command func(cfg *config.Config, args []string) error`.

### Naming Conventions

- **Packages**: lowercase, no underscores (`config`, `tmux`)
- **Functions**: CamelCase (`Load`, `ResolveLogsDir`, `cmdUp`)
- **Variables**: camelCase (`configPath`, `logsDir`)
- **Constants**: CamelCase for exported, camelCase for unexported
- **Tests**: `TestFunctionName_Description` (`TestLoad_ValidConfig`)

### Error Handling

Wrap errors with context: `fmt.Errorf("failed to read config: %w", err)`. Error messages start with lowercase. Return errors instead of logging in library code. Use `t.Fatalf` for test setup errors, `t.Errorf` for assertions.

### Comments

Use `// ` for comments (space after slashes). Start with capital letter for exported items. Document all exported types and functions.

### Testing

Use table-driven tests with `tests := []struct{...}`. Use `t.TempDir()` for temporary directories. Test names should describe the scenario clearly.

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
