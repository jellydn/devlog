# Codebase Concerns

**Analysis Date:** 2026-02-23

## Tech Debt

**Monolithic main.go (862 lines):** → [#27](https://github.com/jellydn/devlog/issues/27)
- Issue: All CLI commands, helpers, and business logic live in a single file
- Files: `cmd/devlog/main.go`
- Impact: Hard to navigate, test individual commands, or add new commands cleanly
- Fix approach: Extract each command into its own file (e.g., `cmd_up.go`, `cmd_down.go`) within the same package

**Duplicated config types across packages:** → [#28](https://github.com/jellydn/devlog/issues/28)
- Issue: `WindowConfig` and `PaneConfig` are defined in both `internal/config` and `internal/tmux` with identical structures, requiring manual mapping in `cmdUp`
- Files: `internal/config/config.go:33-43`, `internal/tmux/tmux.go:347-357`, `cmd/devlog/main.go:258-271`
- Impact: Changes to config structure require updating two type definitions and the mapping code
- Fix approach: Have `tmux` package accept `config.WindowConfig`/`config.PaneConfig` directly, or extract shared types to a common package

**Duplicated Chrome/Brave manifest install functions:** → [#14](https://github.com/jellydn/devlog/issues/14)
- Issue: `InstallChromeManifest` and `InstallBraveManifest` are nearly identical—only the directory path differs
- Files: `internal/natmsg/manifest.go:104-155`
- Impact: Bug fixes or format changes must be applied to both functions
- Fix approach: Extract a shared `installChromiumManifest(dir, hostPath, extensionID)` helper

**Hardcoded `sleep 0.5` in session teardown:** → [#29](https://github.com/jellydn/devlog/issues/29)
- Issue: `KillSession` sends `sleep 0.5` as a tmux command string to wait for graceful termination rather than using `time.Sleep` or a proper wait mechanism
- Files: `internal/tmux/tmux.go:204`
- Impact: Unreliable timing; the sleep runs inside a pane shell, not in the Go process. Process may not be terminated before `kill-session` runs
- Fix approach: Use `time.Sleep(500 * time.Millisecond)` in the Go process between Ctrl+C rounds

## Known Bugs

**Tests use `os.Chdir` without `t.Parallel()` safety:**
- Symptoms: Tests that change working directory with `os.Chdir` can interfere with each other if run in parallel
- Files: `cmd/devlog/init_test.go:12-18,67-73,125-131`, `cmd/devlog/healthcheck_test.go:13-19,36-42,72-78,99-105,164-170`
- Trigger: Running tests with `-parallel` or if Go decides to interleave tests
- Workaround: Tests currently use `defer os.Chdir(originalDir)` but this is process-global state. Use `t.Chdir()` (Go 1.24+) or restructure to avoid `os.Chdir`

**Firefox manifest UUID guard is a no-op:**
- Symptoms: The conditional block for UUID-format extension IDs does the same thing as the default case
- Files: `internal/natmsg/manifest.go:162-169`
- Trigger: Passing a UUID-format extension ID like `{abc-def-...}`
- Workaround: None needed functionally, but the dead code is misleading

## Security Considerations

**Shell command injection surface via config values:**
- Risk: Pane commands from `devlog.yml` are passed to `sh -lc` with single-quote escaping. Session names and window names are passed directly to `tmux` CLI arguments without validation
- Files: `internal/tmux/tmux.go:50,122,146,175-176`, `internal/config/config.go:84-116`
- Current mitigation: Single-quote shell escaping for commands (`tmux.go:175`), path quoting for pipe-pane (`tmux.go:165`), `shellQuote` in wrapper script (`main.go:746-751`)
- Recommendations: Validate session/window names against allowed characters (alphanumeric, dash, underscore). The config file is user-controlled so risk is low, but defense-in-depth is good practice

**Native messaging manifest writes with world-readable permissions (0644):**
- Risk: Manifest files contain the path to the host binary; a local attacker could read these to understand the system
- Files: `internal/natmsg/manifest.go:124,150,189`
- Current mitigation: The binary path is not secret; this is standard for native messaging
- Recommendations: Low risk, acceptable as-is

**Environment variable interpolation in config:**
- Risk: Config files can reference any environment variable via `$VAR` / `${VAR}`, which could expose sensitive values in log paths if misconfigured
- Files: `internal/config/config.go:129-150`
- Current mitigation: Only expands variables that are set; unset variables are left as-is
- Recommendations: Acceptable for a local dev tool. Consider documenting which variables are safe to use

## Performance Bottlenecks

**Sequential tmux command execution:**
- Problem: Each window/pane creation issues separate `exec.Command("tmux", ...)` calls synchronously
- Files: `internal/tmux/tmux.go:50-87,120-142`
- Cause: Each pane setup involves 2-3 sequential subprocess calls (new-window/split, pipe-pane, send-keys)
- Improvement path: Not a real bottleneck for typical use (2-5 windows), but for large configs (10+ windows), could batch via `tmux source-file` with a generated config

**`CleanupOldRuns` reads all directory entries:**
- Problem: Reads entire logs directory listing, gets `Info()` for each entry, then sorts
- Files: `internal/config/config.go:152-230`
- Cause: No indexing; relies on filesystem stat calls
- Improvement path: Not a concern unless hundreds of log run directories accumulate. The `max_runs` setting naturally limits this

## Fragile Areas

**`KillSession` graceful shutdown timing:** → [#29](https://github.com/jellydn/devlog/issues/29)
- Files: `internal/tmux/tmux.go:186-220`
- Why fragile: Relies on sending `C-c` twice with a `sleep 0.5` tmux command between them. If processes don't respond to `SIGINT`, they won't terminate gracefully. The sleep happens in a tmux pane, not the Go process, so timing is non-deterministic
- Safe modification: Replace the tmux `sleep` with `time.Sleep` in Go; consider adding a configurable grace period
- Test coverage: Covered by integration tests (build-tag gated), not by unit tests

**`findConfigFile` upward directory walk:**
- Files: `cmd/devlog/main.go:116-138`
- Why fragile: Walks up to filesystem root looking for `devlog.yml`. If run from `/` or a deeply nested path, it traverses many directories. Also depends on `os.Getwd()` which can fail in edge cases (deleted CWD)
- Safe modification: Add a maximum depth limit or stop at home directory
- Test coverage: No direct tests for this function

**`writeBrowserHostWrapper` / `restoreBrowserHostWrapper` manifest lifecycle:**
- Files: `cmd/devlog/main.go:694-741`
- Why fragile: On `devlog up`, it overwrites all native messaging manifests to point to a session-specific wrapper script. On `devlog down`, it restores them. If the process crashes between up and down, manifests are left pointing to a wrapper that references a stale log path
- Safe modification: Add a startup check to verify manifest integrity
- Test coverage: No tests for these functions

## Scaling Limits

**Single tmux session per project:**
- Current capacity: One session per `devlog.yml` config
- Limit: Cannot run multiple instances of the same project simultaneously
- Scaling path: Would need session name suffixing or namespacing

**Native messaging host is single-writer:**
- Current capacity: One browser log file per session
- Limit: All browser tabs matching URL patterns write to the same file. High-volume logging could cause contention (mitigated by mutex in logger)
- Scaling path: Could add per-tab or per-URL log files

**Log files grow unbounded during a session:**
- Current capacity: Limited only by disk space
- Limit: Long-running sessions with verbose logging can produce very large files. No log rotation during a run
- Scaling path: Add file rotation based on size, or integrate with `logrotate`

## Dependencies at Risk

**`gopkg.in/yaml.v3`:**
- Risk: Only external dependency. Well-maintained and stable. No known security issues
- Impact: Core config parsing
- Migration plan: None needed; this is the standard Go YAML library

**System dependency: `tmux`:**
- Risk: External binary dependency not vendored. Different tmux versions may have different behaviors (format strings, pipe-pane behavior)
- Impact: Core functionality completely depends on tmux being installed and compatible
- Migration plan: `CheckVersion()` exists but doesn't enforce a minimum version. Consider adding minimum version validation

## Missing Critical Features

**No `devlog restart` command:**
- Problem: Users must run `devlog down` then `devlog up` to restart, which generates a new timestamped log directory
- Blocks: Quick iteration when config changes

**No log tailing/viewing command:**
- Problem: No built-in way to view logs without `cat`/`tail` on the log files
- Blocks: Quick debugging without external tools

**No Windows support for tmux:** → [#10](https://github.com/jellydn/devlog/issues/10)
- Problem: tmux is not natively available on Windows. The tool assumes Unix-like OS for shell commands (`sh -lc`)
- Blocks: Windows users entirely (WSL would work but is not documented)

**No config validation for window/pane names:**
- Problem: Window names with special characters (spaces, colons, periods) can break tmux targeting (`session:window` format)
- Blocks: Users with non-alphanumeric window names may get confusing errors
- Files: `internal/config/config.go:84-116`

## Test Coverage Gaps

**`cmd/devlog/main.go` — most command functions untested:** → [#27](https://github.com/jellydn/devlog/issues/27)
- What's not tested: `cmdUp`, `cmdDown`, `cmdAttach`, `cmdStatus`, `cmdLs`, `cmdOpen`, `cmdRegister`, `findConfigFile`, `writeBrowserHostWrapper`, `restoreBrowserHostWrapper`, `shellQuote`, `sanitizeSessionForFileName`, `openInFileManager`
- Files: `cmd/devlog/main.go`
- Risk: The largest file (862 lines) with the most user-facing logic has the least test coverage. Only `cmdInit`, `cmdHealthcheck`, `resolveStatusLogsDir`, `ensureFileExists` are tested
- Priority: **High** — core CLI commands are the primary user interface

**`cmd/devlog-host/main.go` — no tests at all:** → [#30](https://github.com/jellydn/devlog/issues/30)
- What's not tested: The entire native messaging host binary entry point
- Files: `cmd/devlog-host/main.go`
- Risk: Message processing loop and error handling are untested (though underlying `natmsg` and `logger` packages are well-tested)
- Priority: **Medium** — the components it uses are tested, but integration behavior is not

**No `t.Parallel()` usage in any test:**
- What's not tested: Test parallelism correctness
- Files: All `*_test.go` files
- Risk: Tests run sequentially, hiding potential race conditions. Several tests mutate global state (`os.Chdir`) which would break under parallel execution
- Priority: **Low** — not a coverage gap per se, but indicates fragility

**`tmux.Runner` methods not unit-tested in isolation:**
- What's not tested: `CreateSession`, `KillSession`, `GetSessionInfo`, `GetLogsDir` are only tested via integration tests requiring a live tmux server
- Files: `internal/tmux/tmux.go`, `internal/tmux/tmux_test.go`, `internal/tmux/integration_test.go`
- Risk: Integration tests are skipped with `-short` and require tmux, so CI may not run them. The `tmux_test.go` unit tests (9 functions) only cover `SessionExists` and basic operations
- Priority: **Medium** — consider adding a tmux command executor interface for mockable unit tests

---
*Concerns audit: 2026-02-23*
