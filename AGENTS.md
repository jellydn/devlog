# AGENTS.md

> Guidelines for AI agents working on the devlog project

## Build Commands

```bash
# Build the binary
go build -o devlog ./cmd/devlog

# Run the CLI
go run ./cmd/devlog [up|down|status|open|help]

# Install locally
go install ./cmd/devlog

# Clean build artifacts
rm -f devlog
```

## Test Commands

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a single test
go test -run TestLoad_ValidConfig ./...

# Run tests for a specific package
go test ./internal/config/...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

## Lint Commands

```bash
# Format code
go fmt ./...

# Vet code for issues
go vet ./...

# If golangci-lint is available
golangci-lint run
```

## Code Style Guidelines

### Imports

- Standard library imports first
- Third-party imports second
- No blank lines between import groups
- Use full import paths (e.g., `gopkg.in/yaml.v3`)

```go
import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)
```

### Formatting

- Use `gofmt` for formatting
- Tabs for indentation
- Line length: aim for 100 characters max
- No trailing whitespace

### Types

- Use structs with YAML tags: `yaml:"field_name"`
- Use pointer receivers for methods that modify the struct
- Define custom types for command functions: `type Command func(cfg *config.Config, args []string) error`

### Naming Conventions

- **Packages**: lowercase, no underscores (e.g., `config`, `tmux`)
- **Functions**: CamelCase (e.g., `Load`, `ResolveLogsDir`, `cmdUp`)
- **Variables**: camelCase (e.g., `configPath`, `logsDir`)
- **Constants**: CamelCase for exported, camelCase for unexported
- **Tests**: `TestFunctionName_Description` (e.g., `TestLoad_ValidConfig`)
- **Test helpers**: use `t.Helper()` if available

### Error Handling

- Wrap errors with context: `fmt.Errorf("failed to read config: %w", err)`
- Error messages start with lowercase
- Return errors instead of logging in library code
- Exit with `os.Exit(1)` in `main()` for fatal errors
- Use `t.Fatalf` for test setup errors, `t.Errorf` for assertions

```go
// Good
return nil, fmt.Errorf("failed to read config file: %w", err)

// Good
if err := cmd(cfg, os.Args[2:]); err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    os.Exit(1)
}
```

### Comments

- Use `// ` for comments (space after slashes)
- Start with capital letter for exported items
- Document all exported types and functions

```go
// Config represents the devlog.yml configuration
type Config struct {
    // ...
}

// Load reads and parses the devlog.yml file
func Load(path string) (*Config, error) {
    // ...
}
```

### Testing

- Use table-driven tests with `tests := []struct{...}`
- Use `t.TempDir()` for temporary directories (auto-cleaned)
- Set environment variables with `os.Setenv()` and defer cleanup
- Test names should describe the scenario clearly

```go
func TestFunction_Scenario(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid input",
            input: "hello",
            want:  "HELLO",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### Project Structure

```
.
├── cmd/
│   └── devlog/          # CLI entry point
│       └── main.go
├── internal/            # Private packages
│   └── config/          # Config loading/validation
│       ├── config.go
│       └── config_test.go
├── go.mod
├── go.sum
└── devlog.yml.example   # Example configuration
```

### Dependencies

- Go 1.25.6+
- Minimal external dependencies (currently only `gopkg.in/yaml.v3`)
- Prefer standard library when possible

### Configuration

- YAML config files named `devlog.yml`
- Support environment variable interpolation (`$VAR` and `${VAR}`)
- Required fields: `version`, `project`, `tmux.session`, `windows`, `panes`, `cmd`
- Defaults: `logs_dir="./logs"`, `run_mode="timestamped"`
