# Coding Conventions

**Analysis Date:** 2026-02-23

## Naming Patterns

**Files:**
- Go source files: lowercase, no separators (`config.go`, `natmsg.go`, `manifest.go`, `logger.go`)
- Test files: `*_test.go` co-located with source (`config_test.go`, `tmux_test.go`)
- Integration tests: separate file with `integration_` prefix (`integration_test.go`)
- Entry points: always `main.go` under `cmd/<binary>/`

**Functions:**
- Exported: CamelCase (`Load`, `ResolveLogsDir`, `NewRunner`, `SessionExists`, `CheckVersion`)
- Unexported: camelCase (`interpolateEnvVars`, `findConfigFile`, `cmdUp`, `ensurePaneLogFiles`)
- CLI commands: `cmd` prefix with CamelCase command name (`cmdInit`, `cmdUp`, `cmdDown`, `cmdAttach`)
- Constructors: `New` + type name (`NewRunner`, `NewHost`, `NewHostWithStreams`)
- Getters: `Get` prefix for retrieval (`GetLogsDir`, `GetSessionInfo`, `GetChromeNativeMessagingDir`)

**Variables:**
- camelCase for locals (`configPath`, `logsDir`, `sessionName`, `extensionID`)
- Short names in tight scopes (`cfg`, `msg`, `err`, `cmd`, `dir`)
- Receiver names: single letter matching type (`r` for `Runner`, `c` for `Config`, `l` for `Logger`, `h` for `Host`, `t` for `Timestamp`, `m` for `Message`)

**Types:**
- Exported structs: CamelCase nouns (`Config`, `Runner`, `Host`, `Logger`, `Message`, `Response`)
- Info suffix for read-only data types (`SessionInfo`, `WindowInfo`, `PaneInfo`)
- Config suffix for configuration types (`TmuxConfig`, `WindowConfig`, `PaneConfig`, `BrowserConfig`)
- Function types: CamelCase (`Command` in `cmd/devlog/main.go`)

**Constants:**
- Unexported: camelCase (`usage`, `maxLabelLen`)

## Code Style

**Formatting:**
- `gofmt` standard formatting (via `go fmt ./...` in `justfile`)
- Tabs for indentation (Go default)
- No explicit line-length enforcement; generally ~100 chars

**Linting:**
- `go vet ./...` for static analysis (via `justfile`)
- No `.golangci.yml` or `.editorconfig` — relies on `gofmt` + `go vet` only
- Lint target: `just lint` runs `fmt` then `vet`
- CI target: `just ci` runs `lint` then `test`

## Import Organization

**Order:**
1. Standard library imports (grouped together)
2. Third-party imports (separated by blank line)

**Examples from `internal/config/config.go`:**
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

From `internal/logger/logger.go`:
```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jellydn/devlog/internal/natmsg"
)
```

**Path Aliases:**
- None used. All imports use full module paths.

## Error Handling

**Patterns:**
- Wrap errors with context using `fmt.Errorf("context: %w", err)` — consistently throughout
- Error messages start with **lowercase** (`"failed to read config file: %w"`)
- Validation errors prefixed with domain context (`"config: version is required"`, `"config: tmux.session is required"`)
- Return errors from library code; print to stderr in `main()` only
- Check-and-return style (no else after error return):
```go
if err != nil {
    return nil, fmt.Errorf("failed to parse config file: %w", err)
}
```
- Use `os.IsNotExist(err)` for graceful handling of missing files (`internal/config/config.go:167`)
- Ignore errors explicitly with comment when appropriate (`cmd.Run() // Ignore errors - pane might not have a process` in `internal/tmux/tmux.go:200`)

## Logging

**Framework:** Direct `fmt.Fprintf(os.Stderr, ...)` for error output, `fmt.Printf(...)` for user-facing messages

**Patterns:**
- No logging framework; uses `fmt` package directly
- Errors go to `os.Stderr` (`fmt.Fprintf(os.Stderr, "Error: %v\n", err)`)
- Status/progress messages go to `os.Stdout` (`fmt.Printf("Starting devlog session '%s'...\n", ...)`)
- Warning prefix for non-fatal issues (`fmt.Fprintf(os.Stderr, "Warning: ...")`)
- The `internal/logger` package is for browser log capture, not application logging

## Comments

**When to Comment:**
- All exported types and functions get `// TypeName ...` or `// FuncName ...` doc comments
- Package-level doc comments on every package (`// Package natmsg implements ...`, `// Package logger handles ...`)
- Inline comments for non-obvious logic (`// Sanity check: messages shouldn't be larger than 10MB`)
- Comments start with `// ` (space after slashes), capitalized first letter for doc comments
- No commented-out code observed

## Function Design

**Size:** Functions are generally 10–40 lines. Largest (`cmdUp`) is ~50 lines of sequential logic.

**Parameters:**
- Accept interfaces/primitives over concrete types where possible
- Configuration structs passed as `*Config` pointer
- `io.Reader`/`io.Writer` for testable I/O (`NewHostWithStreams`)
- String paths rather than file handles

**Return Values:**
- `(value, error)` tuple pattern consistently used
- Pointer return for structs (`*Config`, `*Message`, `*SessionInfo`)
- `bool` for simple checks (`SessionExists`, `ShouldLog`)
- Named return values not used

## Module Design

**Exports:**
- Each package exports a focused API: types + constructor + methods
- Internal packages under `internal/` prevent external use
- CLI commands are unexported functions in `cmd/devlog/main.go`

**Barrel Files:**
- Not used. Each package has 1–3 files with clear responsibilities.

**Package Structure:**
- `internal/config/` — `config.go` (single file, types + loading + validation + cleanup)
- `internal/tmux/` — `tmux.go` (runner + types), `integration_test.go` (build-tagged)
- `internal/natmsg/` — `natmsg.go` (protocol), `manifest.go` (browser manifest management)
- `internal/logger/` — `logger.go` (file-based log writer)
- `cmd/devlog/` — `main.go` (CLI entry + all command implementations)
- `cmd/devlog-host/` — `main.go` (native messaging host binary)

**Dependencies:**
- Minimal: only `gopkg.in/yaml.v3` as external dependency
- Standard library preferred (`encoding/json`, `os/exec`, `path/filepath`)
- No dependency injection framework; manual wiring via constructors

---
*Convention analysis: 2026-02-23*
