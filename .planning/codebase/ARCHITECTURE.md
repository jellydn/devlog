# Architecture

**Analysis Date:** 2026-02-23

## Pattern Overview
**Overall:** Multi-binary CLI tool with browser extension sidecar

**Key Characteristics:**
- Two Go binaries (`devlog` CLI and `devlog-host` native messaging host) communicating indirectly via tmux sessions and file-based logging
- Browser extension (Chrome/Firefox) captures console logs and forwards them to `devlog-host` via the Native Messaging protocol
- YAML-driven configuration with environment variable interpolation
- Append-only log files with timestamped run directories for isolation
- Minimal dependencies — only `gopkg.in/yaml.v3` beyond the Go standard library

## Layers

**CLI Layer (cmd/devlog):**
- Purpose: User-facing command dispatcher for managing dev logging sessions
- Location: `cmd/devlog/main.go`
- Contains: Command definitions (`cmdUp`, `cmdDown`, `cmdAttach`, `cmdStatus`, `cmdLs`, `cmdOpen`, `cmdInit`, `cmdRegister`, `cmdHealthcheck`), config discovery, shell wrapper generation
- Depends on: `internal/config`, `internal/tmux`, `internal/natmsg`
- Used by: End users via terminal

**Native Messaging Host Layer (cmd/devlog-host):**
- Purpose: Receives browser console log messages via stdin (Native Messaging protocol) and writes them to disk
- Location: `cmd/devlog-host/main.go`
- Contains: Message read loop, log writing orchestration
- Depends on: `internal/natmsg`, `internal/logger`
- Used by: Browser extension (launched by the browser as a native messaging host process)

**Configuration Layer:**
- Purpose: YAML config loading, validation, environment variable interpolation, retention policy cleanup
- Location: `internal/config/config.go`
- Contains: `Config`, `TmuxConfig`, `WindowConfig`, `PaneConfig`, `BrowserConfig` structs; `Load()`, `Validate()`, `ResolveLogsDir()`, `CleanupOldRuns()` functions
- Depends on: `gopkg.in/yaml.v3`
- Used by: CLI layer

**Tmux Layer:**
- Purpose: Manages tmux sessions, windows, and panes with pipe-pane log capture
- Location: `internal/tmux/tmux.go`
- Contains: `Runner` struct with session lifecycle methods (`CreateSession`, `KillSession`, `SessionExists`, `GetSessionInfo`); `WindowConfig`, `PaneConfig` (local duplicates for decoupling)
- Depends on: `tmux` binary (external process via `os/exec`)
- Used by: CLI layer

**Native Messaging Protocol Layer:**
- Purpose: Implements the browser Native Messaging wire protocol (length-prefixed JSON over stdin/stdout) and manages browser manifest registration
- Location: `internal/natmsg/natmsg.go`, `internal/natmsg/manifest.go`
- Contains: `Host` (read/write messages), `Message`/`Response` types, `Timestamp` with flexible unmarshaling, manifest install/update functions for Chrome/Brave/Firefox
- Depends on: Standard library only
- Used by: `devlog-host` binary, CLI layer (for `register` and `healthcheck` commands)

**Logger Layer:**
- Purpose: Writes formatted, level-filtered browser log messages to append-only files
- Location: `internal/logger/logger.go`
- Contains: `Logger` struct with mutex-protected file writes, level filtering
- Depends on: `internal/natmsg` (for `Message` type)
- Used by: `devlog-host` binary

**Browser Extension Layer:**
- Purpose: Captures browser console logs from web pages and forwards them to the native host
- Location: `browser-extension/`
- Contains: `page_inject.js` (wraps `console.*` methods in page context), `content_script.js` (bridges page → background), `background.js` (manages native messaging port, URL filtering)
- Depends on: Chrome/Firefox extension APIs, Native Messaging API
- Used by: Browsers (Chrome, Firefox, Brave)

## Data Flow

**Server Log Capture (tmux pipe-pane):**
1. User runs `devlog up` → CLI loads `devlog.yml` config
2. CLI creates tmux session with windows/panes via `tmux.Runner.CreateSession()`
3. Each pane runs its configured command with `tmux pipe-pane` capturing stdout to a log file
4. Log files are written to `<logs_dir>/<timestamp>/` (timestamped mode) or `<logs_dir>/` (overwrite mode)
5. User runs `devlog down` → CLI sends Ctrl+C to all panes, then kills the tmux session

**Browser Log Capture (native messaging):**
1. `devlog up` generates a shell wrapper script at `~/Library/Caches/devlog/wrappers/devlog-host-wrapper-<session>.sh` that calls `devlog-host` with the correct log path and level filters
2. The wrapper path is written into the browser's native messaging manifest (`com.devlog.host.json`)
3. In the browser, `page_inject.js` wraps `console.*` methods and posts messages to `content_script.js` via `window.postMessage`
4. `content_script.js` forwards matching messages to `background.js` via `chrome.runtime.sendMessage`
5. `background.js` checks URL patterns, then sends the log message via `chrome.runtime.connectNative` → native messaging port
6. The browser launches `devlog-host` (via the wrapper), which reads length-prefixed JSON from stdin and writes formatted lines to the log file
7. `devlog down` restores the original binary path in the native messaging manifest

**State Management:**
- Session state is stored in tmux itself (session existence, environment variables like `DEVLOG_LOGS_DIR`)
- No database or persistent state file — the filesystem (log directories) and tmux are the sources of truth
- Browser extension state is in-memory only (connection status, config)

## Key Abstractions

**Command Function Type:**
- Purpose: Uniform interface for all CLI subcommands
- Examples: `cmd/devlog/main.go` — `type Command func(cfg *config.Config, args []string) error`
- Pattern: Function-as-value command dispatch via `map[string]Command`

**tmux.Runner:**
- Purpose: Encapsulates all tmux subprocess interactions for a named session
- Examples: `internal/tmux/tmux.go`
- Pattern: Constructor injection (`NewRunner(sessionName)`) with method-based API

**natmsg.Host:**
- Purpose: Abstracts Native Messaging wire protocol (stdin/stdout read/write with length prefix)
- Examples: `internal/natmsg/natmsg.go`
- Pattern: Stream-based I/O with `NewHostWithStreams()` for testability

**config.Config:**
- Purpose: Strongly-typed representation of `devlog.yml` with validation and derived paths
- Examples: `internal/config/config.go`
- Pattern: Load → Interpolate → Unmarshal → Default → Validate pipeline

## Entry Points

**devlog CLI:**
- Location: `cmd/devlog/main.go`
- Triggers: User invokes `devlog <command>` from terminal
- Responsibilities: Parse args, find and load config, dispatch to command handler

**devlog-host:**
- Location: `cmd/devlog-host/main.go`
- Triggers: Browser launches it via native messaging when extension connects
- Responsibilities: Read native messages from stdin, filter by level, write formatted logs to file

**Browser Extension:**
- Location: `browser-extension/background.js` (entry), `browser-extension/content_script.js`, `browser-extension/page_inject.js`
- Triggers: Browser loads extension on page navigation
- Responsibilities: Intercept `console.*` calls, forward to native messaging host

## Error Handling
**Strategy:** Return errors with context wrapping; log warnings for non-fatal issues

**Patterns:**
- All internal packages return errors with `fmt.Errorf("context: %w", err)` wrapping
- CLI commands print errors to stderr and exit with code 1
- Non-fatal issues (e.g., failed cleanup, failed browser wrapper setup) are logged as warnings and don't abort the command
- `devlog-host` logs errors to stderr but continues processing (resilient message loop)
- Browser extension uses `console.error`/`console.warn` for diagnostics; silently ignores errors for tabs without content scripts

## Cross-Cutting Concerns

**Logging:**
- No structured logging framework — uses `fmt.Printf` for user output and `fmt.Fprintf(os.Stderr, ...)` for errors/warnings
- Browser logs are formatted as `[TIMESTAMP] [LEVEL] [URL] source:line:column: message`

**Configuration:**
- Single `devlog.yml` file with upward directory traversal for discovery
- Environment variable interpolation (`$VAR` / `${VAR}`) before YAML parsing
- Validation at load time with descriptive error messages

**Security:**
- Shell command injection prevention via single-quote escaping (`shellQuote()`)
- Tmux `pipe-pane` paths are escaped
- Native messaging message size capped at 10MB
- Browser extension uses `web_accessible_resources` for page injection (avoids CSP issues)

**Cross-Platform:**
- Native messaging manifest paths handled per-OS (macOS, Linux, Windows)
- File manager opening uses platform-appropriate commands (`open`, `xdg-open`, `cmd /c start`)
- Supports Chrome, Brave, Firefox, and Zen browser

---
*Architecture analysis: 2026-02-23*
