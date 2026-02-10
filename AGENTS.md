# AGENTS.md

> Guidelines for AI agents working on the devlog project

## Build Commands

```bash
# Build binaries
go build -o devlog ./cmd/devlog
go build -o devlog-host ./cmd/devlog-host
just devlog-dev     # Build both + symlink to ~/.local/bin

# Install locally
go install ./cmd/devlog
go install ./cmd/devlog-host
```

## Test Commands

```bash
go test ./...                          # All tests
go test -v ./...                       # Verbose
just test-one TestLoad_ValidConfig     # Single test
go test -cover ./...                   # Coverage
go test -race ./...                    # Race detection
go test -tags=integration ./internal/tmux/  # Integration tests
go test -short ./...                   # Skip integration tests
just lint                              # Format + vet
just ci                                # Lint + test
```

Integration tests use `-tags=integration` and are skipped with `-short`.

## Code Style Guidelines

### Imports

Standard library first, then third-party. No blank lines between groups.

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)
```

### Formatting

Use `gofmt` for formatting. Tabs for indentation. Aim for 100 characters max. No trailing whitespace.

### Types

Use structs with YAML tags (`yaml:"field_name"`). Use pointer receivers for methods that modify the struct.

```go
type Command func(cfg *config.Config, args []string) error
```

Define command function types for CLI commands.

### Naming Conventions

- **Packages**: lowercase, no underscores (`config`, `tmux`)
- **Functions**: CamelCase (`Load`, `ResolveLogsDir`, `cmdUp`)
- **Variables**: camelCase (`configPath`, `logsDir`)
- **Constants**: CamelCase for exported, camelCase for unexported
- **Tests**: `TestFunctionName_Description` (`TestLoad_ValidConfig`)

### Error Handling

Wrap errors with context: `fmt.Errorf("failed to read config: %w", err)`. Error messages start with lowercase.

Prefix validation errors with context: `fmt.Errorf("config: version is required")`.

Return errors instead of logging in library code. Use `t.Fatalf` for test setup errors, `t.Errorf` for assertions.

### Comments

Use `// ` for comments (space after slashes). Start with capital letter for exported items. Document all exported types and functions.

### Testing

Use table-driven tests with `tests := []struct{...}`. Use `t.TempDir()` for temporary directories. Use subtests with `t.Run()` for clarity. Test names should describe the scenario clearly.

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
│   │   ├── tmux_test.go
│   │   └── integration_test.go
│   ├── natmsg/           # Native messaging protocol
│   │   ├── natmsg.go
│   │   ├── natmsg_test.go
│   │   └── manifest.go
│   └── logger/           # Logging utilities
│       ├── logger.go
│       └── logger_test.go
├── browser-extension/    # Chrome/Firefox extension
├── go.mod
├── go.sum
├── justfile
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
