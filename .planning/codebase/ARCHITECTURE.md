# Architecture

**Analysis Date:** 2026-07-19 (regenerated)

## Pattern Overview
**Overall:** Multi-binary CLI tool with browser extension sidecar

**Key Characteristics:**
- Two Go binaries (`devlog` CLI and `devlog-host` native messaging host) communicating indirectly via tmux sessions and file-based logging
- Browser extension (Chrome/Firefox) captures console logs and forwards them to `devlog-host` via the Native Messaging protocol
- YAML-driven configuration with environment variable interpolation
- Append-only log files with timestamped run directories for isolation
- Minimal dependencies — only `gopkg.in/yaml.v3` beyond the Go standard library
- Interface-based dependency injection for testability (`ManifestOps`, `SessionChecker`)

## Layers

**CLI Layer (cmd/devlog):**
- Purpose: User-facing command dispatcher for managing dev logging sessions
- Location: `cmd/devlog/main.go` (dispatcher) + `cmd/devlog/cmd_*.go` (one file per command) + `cmd/devlog/helpers.go` (shared helpers) + `cmd/devlog/browser_session_adapter.go` (DI adapters)
- Contains: Command definitions (`cmdUp`, `cmdDown`, `cmdAttach`, `cmdStatus`, `cmdLs`, `cmdOpen`, `cmdInit`, `cmdRegister`, `cmdHealthcheck`), config discovery (`findConfigFile`), file creation utility (`ensureFileExists`)
- Depends on: `internal/config`, `internal/tmux`, `internal/manifest`, `internal/browsersession`, `internal/logrotate`, `internal/fileutil`, `internal/shellescape`
- Used by: End users via terminal

**Native Messaging Host Layer (cmd/devlog-host):**
- Purpose: Receives browser console log messages via stdin (Native Messaging protocol) and writes them to disk
- Location: `cmd/devlog-host/main.go`
- Contains: Message read loop, log writing orchestration
- Depends on: `internal/natmsg`, `internal/logger`
- Used by: Browser extension (launched by the browser as a native messaging host process)

**Configuration Layer:**
- Purpose: YAML config loading, validation, environment variable interpolation
- Location: `internal/config/config.go`
- Contains: `Config`, `TmuxConfig`, `WindowConfig`, `PaneConfig`, `BrowserConfig` structs; `Load()`, `Validate()` functions
- Depends on: `gopkg.in/yaml.v3`
- Used by: CLI layer

**Tmux Layer:**
- Purpose: Manages tmux sessions, windows, and panes with pipe-pane log capture. Owns log-directory resolution.
- Location: `internal/tmux/tmux.go`
- Contains: `Runner` struct with session lifecycle methods (`CreateSession`, `KillSession`, `SessionExists`, `GetSessionInfo`, `GetLogsDir`); `SessionConfig` (bundles session name, logs dir, run mode, windows)
- Depends on: `internal/config` (for `WindowConfig`/`PaneConfig` types), `internal/fileutil`, `internal/shellescape`, `tmux` binary (external process via `os/exec`)
- Used by: CLI layer

**Native Messaging Protocol Layer:**
- Purpose: Implements the browser Native Messaging wire protocol (length-prefixed JSON over stdin/stdout)
- Location: `internal/natmsg/natmsg.go`
- Contains: `Host` (read/write messages), `Message`/`Response` types, `Timestamp` with flexible unmarshaling
- Depends on: Standard library only
- Used by: `devlog-host` binary

**Manifest Layer:**
- Purpose: Manages browser native messaging host registration — installing, updating, and repairing manifest JSON files
- Location: `internal/manifest/manifest.go`, `internal/manifest/validate_host_*.go`
- Contains: `ChromeManifest`, `FirefoxManifest` structs; install/update/repair functions for Chrome/Brave/Firefox/Zen; OS-specific `ValidateHostPath` (build-tagged)
- Depends on: Standard library only
- Used by: CLI layer (via `browsersession.ManifestOps` interface), `cmd_register.go`

**Browser Session Layer:**
- Purpose: Manages browser-log capture lifecycle — wrapper script creation/destruction, clobber protection, health checks
- Location: `internal/browsersession/session.go`
- Contains: `Session` struct with `Start`/`Stop`/`HealthCheck`; `ManifestOps` and `SessionChecker` interfaces for DI
- Depends on: `internal/shellescape` (for wrapper script generation)
- Used by: CLI layer (`cmd_up.go`, `cmd_down.go`, `cmd_healthcheck.go`)

**Log Rotation Layer:**
- Purpose: Cleans up old timestamped log directories based on retention policy
- Location: `internal/logrotate/logrotate.go`
- Contains: `Policy`, `Result` structs; `Cleanup` function
- Depends on: Standard library only
- Used by: CLI layer (`cmd_up.go`)

**Logger Layer:**
- Purpose: Writes formatted, level-filtered browser log messages to append-only files
- Location: `internal/logger/logger.go`
- Contains: `Logger` struct with mutex-protected file writes, level filtering
- Depends on: `internal/natmsg` (for `Message` type)
- Used by: `devlog-host` binary

**File Utilities Layer:**
- Purpose: Shared filesystem helper functions
- Location: `internal/fileutil/touchfile.go`
- Contains: `TouchFile` — ensures a file exists (MkdirAll + OpenFile + Close)
- Depends on: Standard library only
- Used by: `cmd/devlog/helpers.go`, `internal/tmux/tmux.go`

**Browser Extension Layer:**
- Purpose: Captures browser console logs from web pages and forwards them to the native host
- Location: `browser-extension/`
- Contains: `page_inject.js` (wraps `console.*` methods in page context), `content_script.js` (bridges page → background), `background.js` (manages native messaging port, URL filtering)
- Depends on: Chrome/Firefox extension APIs, Native Messaging API
- Used by: Browsers (Chrome, Firefox, Brave)

## Data Flow

**Server Log Capture (tmux pipe-pane):**
1. User runs `devlog up` → CLI loads `devlog.yml` config
2. CLI constructs `tmux.SessionConfig` and calls `runner.CreateSession()`
3. Runner resolves logs directory (timestamped subdirectory if needed), stores it internally, creates the directory, and ensures pane log files via `fileutil.TouchFile`
4. Each pane runs its configured command with `tmux pipe-pane` capturing stdout to a log file
5. Log files are written to `<logs_dir>/<timestamp>/` (timestamped mode) or `<logs_dir>/` (overwrite mode)
6. `DEVLOG_LOGS_DIR` is set as a tmux session environment variable for external tool access
7. User runs `devlog down` → CLI sends Ctrl+C to all panes, then kills the tmux session; also restores browser manifest path via `browsersession.Session.Stop()`

**Browser Log Capture (native messaging):**
1. `devlog up` → `browsersession.Session.Start()` generates a wrapper script at `~/Library/Caches/devlog/wrappers/devlog-host-wrapper-<session>.sh`
2. The wrapper path is written into the browser's native messaging manifest (`com.devlog.host.json`) via `ManifestOps.UpdateManifestPath`
3. In the browser, `page_inject.js` wraps `console.*` methods and posts messages to `content_script.js` via `window.postMessage`
4. `content_script.js` forwards matching messages to `background.js` via `chrome.runtime.sendMessage`
5. `background.js` checks URL patterns, then sends the log message via `chrome.runtime.connectNative` → native messaging port
6. The browser launches `devlog-host` (via the wrapper), which reads length-prefixed JSON from stdin and writes formatted lines to the log file
7. `devlog down` → `browsersession.Session.Stop()` restores the original binary path in the native messaging manifest

**Log Cleanup:**
1. On `devlog up`, if `run_mode: timestamped`, `logrotate.Cleanup()` is called
2. Old directories exceeding `max_runs` count or older than `retention_days` are removed

**State Management:**
- Session state is stored in tmux itself (session existence, environment variables like `DEVLOG_LOGS_DIR`)
- `tmux.Runner` also caches the resolved `logsDir` in-memory for fast access via `GetLogsDir()`
- No database or persistent state file — the filesystem (log directories) and tmux are the sources of truth
- Browser extension state is in-memory only (connection status, config)

## Key Abstractions

**Command Function Type:**
- Purpose: Uniform interface for all CLI subcommands
- Examples: `cmd/devlog/main.go` — `type Command func(cfg *config.Config, args []string) error`; handlers implemented in `cmd/devlog/cmd_*.go`
- Pattern: Function-as-value command dispatch via `map[string]Command`

**tmux.Runner:**
- Purpose: Encapsulates all tmux subprocess interactions for a named session. Owns log-directory resolution and storage.
- Examples: `internal/tmux/tmux.go`
- Pattern: Constructor injection (`NewRunner(sessionName)`) with method-based API; `SessionConfig` bundles all creation-time parameters

**browsersession.Session:**
- Purpose: Manages browser-log capture lifecycle with interface-based DI for testability
- Examples: `internal/browsersession/session.go`
- Pattern: Constructor injection (`New(manifest ManifestOps, tmux SessionChecker)`) with `Start`/`Stop`/`HealthCheck` API

**natmsg.Host:**
- Purpose: Abstracts Native Messaging wire protocol (stdin/stdout read/write with length prefix)
- Examples: `internal/natmsg/natmsg.go`
- Pattern: Stream-based I/O with `NewHostWithStreams()` for testability

**config.Config:**
- Purpose: Strongly-typed representation of `devlog.yml` with validation
- Examples: `internal/config/config.go`
- Pattern: Load → Interpolate → Unmarshal → Default → Validate pipeline

## Entry Points

**devlog CLI:**
- Location: `cmd/devlog/main.go` (dispatcher) + `cmd/devlog/cmd_*.go` (handlers)
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
- Shell command injection prevention via single-quote escaping (`shellescape.Quote()`)
- Tmux `pipe-pane` paths are escaped
- Native messaging message size capped at 10MB
- Browser extension uses `web_accessible_resources` for page injection (avoids CSP issues)

**Cross-Platform:**
- Native messaging manifest paths handled per-OS (macOS, Linux, Windows)
- File manager opening uses platform-appropriate commands (`open`, `xdg-open`, `cmd /c start`)
- Supports Chrome, Brave, Firefox, and Zen browser

---
*Architecture analysis: 2026-07-19 (regenerated)*
