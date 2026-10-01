# Coding Conventions

**Analysis Date:** 2026-07-19 (regenerated)

## Naming Patterns

**Files:**
- Go source files: lowercase, no separators (`config.go`, `natmsg.go`, `session.go`, `logger.go`)
- Test files: `*_test.go` co-located with source (`config_test.go`, `tmux_test.go`)
- Integration tests: separate file with `integration_` prefix (`integration_test.go`)
- Entry points: always `main.go` under `cmd/<binary>/`

**Functions:**
- Exported: CamelCase (`Load`, `NewRunner`, `SessionExists`, `CheckVersion`, `TouchFile`)
- Unexported: camelCase (`interpolateEnvVars`, `findConfigFile`, `cmdUp`, `ensurePaneLogFiles`)
- CLI commands: `cmd` prefix with CamelCase command name (`cmdInit`, `cmdUp`, `cmdDown`, `cmdAttach`)
- Constructors: `New` + type name (`NewRunner`, `NewHost`, `NewHostWithStreams`, `New` for `browsersession.Session`)
- Getters: `Get` prefix for retrieval (`GetLogsDir`, `GetSessionInfo`, `GetChromeNativeMessagingDir`)

**Variables:**
- camelCase for locals (`configPath`, `logsDir`, `sessionName`, `extensionID`)
- Short names in tight scopes (`cfg`, `msg`, `err`, `cmd`, `dir`)
- Receiver names: single letter matching type (`r` for `Runner`, `c` for `Config`, `s` for `Session`, `h` for `Host`, `m` for `Message`)

**Types:**
- Exported structs: CamelCase nouns (`Config`, `Runner`, `Host`, `Logger`, `Message`, `Response`, `Session`)
- Info suffix for read-only data types (`SessionInfo`, `WindowInfo`, `PaneInfo`, `HealthResult`)
- Config suffix for configuration types (`TmuxConfig`, `WindowConfig`, `PaneConfig`, `BrowserConfig`, `SessionConfig`)
- Function types: CamelCase (`Command` in `cmd/devlog/main.go`)

**Constants:**
- Unexported: camelCase (`usage`, `maxLabelLen`, `maxFindConfigDepth`)

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
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)
```

From `internal/tmux/tmux.go`:
```go
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jellydn/devlog/internal/config"
	"github.com/jellydn/devlog/internal/fileutil"
	"github.com/jellydn/devlog/internal/shellescape"
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
- Use `os.IsNotExist(err)` for graceful handling of missing files
- Ignore errors explicitly with comment when appropriate (`cmd.Run() // Ignore errors - pane might not have a process`)

## Logging

**Framework:** Direct `fmt.Fprintf(os.Stderr, ...)` for error output, `fmt.Printf(...)` for user-facing messages

**Patterns:**
- No logging framework; uses `fmt` package directly
- Errors go to `os.Stderr` (`fmt.Fprintf(os.Stderr, "Error: %v\n", err)`)
- Status/progress messages go to `os.Stdout` (`fmt.Printf("Starting devlog session '%s'...\n", ...)`)
- Warning prefix for non-fatal issues (`fmt.Fprintf(os.Stderr, "Warning: ...\")`)
- The `internal/logger` package is for browser log capture, not application logging

## Comments

**When to Comment:**
- All exported types and functions get `// TypeName ...` or `// FuncName ...` doc comments
- Package-level doc comments on every package (`// Package natmsg implements ...`, `// Package browsersession manages ...`)
- Inline comments for non-obvious logic (`// Sanity check: messages shouldn't be larger than 10MB`)
- Comments start with `// ` (space after slashes), capitalized first letter for doc comments
- No commented-out code observed

## Function Design

**Size:** Functions are generally 10–40 lines. Largest is ~50 lines of sequential logic. Methods are broken down into focused private helpers when exceeding ~20 lines (e.g., `HealthCheck` delegates to `discoverHost`, `checkRegisteredBrowsers`, `repairAndCountPaths`).

**Parameters:**
- Accept interfaces over concrete types where possible (e.g., `ManifestOps`, `SessionChecker`)
- Configuration structs passed as `*Config` pointer or by value (`SessionConfig`, `Policy`)
- `io.Reader`/`io.Writer` for testable I/O (`NewHostWithStreams`)
- String paths rather than file handles

**Return Values:**
- `(value, error)` tuple pattern consistently used
- Pointer return for structs (`*Config`, `*Message`, `*SessionInfo`, `*HealthResult`)
- `bool` for simple checks (`SessionExists`, `ShouldLog`)
- Named return values not used

## Module Design

**Exports:**
- Each package exports a focused API: types + constructor + methods
- Internal packages under `internal/` prevent external use
- CLI commands are unexported functions in `cmd/devlog/main.go`
- Interface-based DI for testability (`ManifestOps`, `SessionChecker` in `internal/browsersession`)

**Barrel Files:**
- Not used. Each package has 1–3 files with clear responsibilities.

**Package Structure:**
- `internal/config/` — `config.go` (types + loading + validation)
- `internal/tmux/` — `tmux.go` (runner + types + session management), `integration_test.go` (build-tagged)
- `internal/natmsg/` — `natmsg.go` (wire protocol: Host, Message, read/write)
- `internal/manifest/` — `manifest.go` (browser manifest install/update/repair), `validate_host_*.go` (OS-specific)
- `internal/browsersession/` — `session.go` (Session, ManifestOps/SessionChecker interfaces, health check)
- `internal/logrotate/` — `logrotate.go` (Policy, Result, Cleanup)
- `internal/logger/` — `logger.go` (file-based log writer with level filtering)
- `internal/fileutil/` — `touchfile.go` (shared TouchFile helper)
- `internal/shellescape/` — `shellescape.go` (POSIX shell quoting)
- `cmd/devlog/` — `main.go` (CLI entry + command dispatch) + `cmd_*.go` (one per command) + `browser_session_adapter.go` (DI adapters)
- `cmd/devlog-host/` — `main.go` (native messaging host binary)

**Dependencies:**
- Minimal: only `gopkg.in/yaml.v3` as external dependency
- Standard library preferred (`encoding/json`, `os/exec`, `path/filepath`)
- No dependency injection framework; manual wiring via constructors and adapters

---
*Convention analysis: 2026-07-19 (regenerated)*
