# Codebase Concerns

**Analysis Date:** 2026-07-19
**Prior audit:** 2026-02-23 (most items from that audit have since been resolved; see _Resolved_ traceability at the bottom)

This document captures concerns that remain valid against the current codebase. Items already verified as fixed are listed in [_Resolved since the 2026-02-23 audit_](#resolved-since-the-2026-02-23-audit) for traceability.

## Tech Debt

**Hardcoded `sleep 0.5` in session teardown:** → [#29](https://github.com/jellydn/devlog/issues/29)
- Issue: `KillSession` sends `sleep 0.5` as a tmux command string to wait for graceful termination rather than using a proper wait mechanism
- Files: `internal/tmux/tmux.go:204`
- Impact: Unreliable timing; the sleep runs inside a pane shell, not in the Go process. Process may not be terminated before `kill-session` runs
- Fix approach: Use `time.Sleep(500 * time.Millisecond)` in the Go process between Ctrl+C rounds
- Status (2026-07-19): Partially addressed — the Go process now calls `time.Sleep(500 * time.Millisecond)` (see `internal/tmux/tmux.go:208`) with a comment explaining why the tmux `sleep` was unreliable. The graceful-shutdown mechanism still relies on `C-c` responsiveness; a configurable grace period would still be an improvement.

## Known Bugs

_None confirmed against the current codebase._

The 2026-02-23 audit listed two known bugs (`os.Chdir` test interference and the dead Firefox UUID guard). Both are resolved — see the traceability section below.

## Security Considerations

**Shell command injection surface via config values:**
- Risk: Pane commands from `devlog.yml` are passed to `sh -lc` via the tested `internal/shellescape` package. Session and window names are passed directly to `tmux` CLI arguments without character validation
- Files: `internal/tmux/tmux.go:170,179`, `internal/shellescape/`, `internal/config/config.go:84-116`
- Current mitigation: `shellescape.Quote` for commands and pipe paths (centralized and unit-tested); the config file is user-controlled so risk is low
- Recommendations: Defense-in-depth — validate session/window names against allowed characters (alphanumeric, dash, underscore) in `config.Validate()` to reject names that break tmux's `session:window` targeting before they reach the CLI

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
- Improvement path: Not a real bottleneck for typical use (2-5 windows). For large configs (10+ windows), could batch via `tmux source-file` with a generated config

**`CleanupOldRuns` reads all directory entries:**
- Problem: Reads entire logs directory listing, gets `Info()` for each entry, then sorts
- Files: `internal/config/config.go:152-230`
- Cause: No indexing; relies on filesystem stat calls
- Improvement path: Not a concern unless hundreds of log run directories accumulate. The `max_runs` setting naturally limits this

## Fragile Areas

**`KillSession` graceful shutdown timing:** → [#29](https://github.com/jellydn/devlog/issues/29)
- Files: `internal/tmux/tmux.go:186-220`
- Why fragile: Relies on sending `C-c` twice with a `time.Sleep` between them in the Go process. If processes don't respond to `SIGINT`, they won't terminate gracefully
- Safe modification: Add a configurable grace period and consider `SIGTERM`/`SIGKILL` escalation
- Test coverage: Covered by integration tests (build-tag gated), not by unit tests

**`writeBrowserHostWrapper` / `restoreBrowserHostWrapper` manifest lifecycle:**
- Files: `cmd/devlog/helpers.go:100-153,225-249`
- Why fragile: On `devlog up`, it overwrites all native messaging manifests to point to a session-specific wrapper script. On `devlog down`, it restores them. If the process crashes between up and down, manifests are left pointing to a wrapper that references a stale log path
- Mitigation now in place: `natmsg.RepairStaleManifestPaths` self-heals stale/missing manifest paths on the next `devlog up`, and `refuseClobberActiveWrapper` guards against overwriting a live session's wrapper
- Test coverage: `cmd/devlog/browser_wrapper_test.go` covers the testable core (`writeBrowserHostWrapperWithHost`, `restoreBrowserHostWrapperWithHost`)
- Remaining improvement: A dedicated startup integrity check command could surface stale state without requiring a full `devlog up`

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
- Problem: Window names with special characters (spaces, colons, periods) can break tmux targeting (`session:window` format). `config.Validate()` checks required fields, run mode, and numeric bounds, but not name character sets
- Blocks: Users with non-alphanumeric window names may get confusing errors
- Files: `internal/config/config.go:83-115`
- Fix approach: Add a name-character allow-list check inside the `Validate()` loop over windows/panes

## Test Coverage Gaps

**`cmd/devlog` command coverage still uneven:** → [#27](https://github.com/jellydn/devlog/issues/27)
- What's tested now: `cmdInit`, `cmdHealthcheck`, `findConfigFile`, `sanitizeSessionForFileName`, `generateShellScript`/`generateBatchScript`, `writeBrowserHostWrapperWithHost`, `restoreBrowserHostWrapperWithHost`, `refuseClobberActiveWrapper`, `resolveStatusLogsDir`, `ensureFileExists`
- What's still not tested directly: `cmdUp`, `cmdDown`, `cmdAttach`, `cmdStatus`, `cmdLs`, `cmdOpen`, `cmdRegister`
- Files: `cmd/devlog/cmd_*.go`
- Risk: The user-facing command surface (`cmdUp`/`cmdDown`/`cmdRegister`) is exercised only via integration tests requiring live tmux
- Priority: **Medium** — the testable helper cores are now covered; the remaining gap is the command orchestration layer

**`tmux.Runner` methods not unit-tested in isolation:**
- What's not tested: `CreateSession`, `KillSession`, `GetSessionInfo`, `GetLogsDir` are only tested via integration tests requiring a live tmux server
- Files: `internal/tmux/tmux.go`, `internal/tmux/tmux_test.go`, `internal/tmux/integration_test.go`
- Risk: Integration tests are skipped with `-short` and require tmux, so CI may not run them. The `tmux_test.go` unit tests only cover `SessionExists` and basic operations
- Priority: **Medium** — consider adding a tmux command executor interface for mockable unit tests

**No `t.Parallel()` usage in any test:**
- What's not tested: Test parallelism correctness
- Files: All `*_test.go` files
- Risk: Tests run sequentially. The previous `os.Chdir` global-state hazard is gone (tests now use `t.Chdir`), so parallel execution is now safe to introduce, but no test opts in
- Priority: **Low** — not a coverage gap per se; safe to enable opportunistically per-package

## Browser Extension

**Firefox extension no longer diverges from shared root assets:**
- Resolved (2026-07-19): `browser-extension/firefox/` now mirrors `browser-extension/chrome/` — `background.js`, `content_script.js`, `popup.html`, `popup.js`, `page_inject.js`, and `icons/` are symlinks to the shared root files. Only `manifest.json` differs (MV2 vs MV3). `scripts/package-firefox.sh` now copies shared files from `browser-extension/` (the canonical source) like `package-chrome.sh`, so the archive never depends on symlink resolution. The root `popup.js` help text is browser-generic (`Run: devlog register`) and `page_inject.js` symlinks were added to both `chrome/` and `firefox/` so unpacked dev loading resolves all manifest-referenced files.
- Verification: Both `package-chrome.sh` and `package-firefox.sh` produce archives containing `manifest.json`, `background.js`, `content_script.js`, `page_inject.js`, `popup.html`, `popup.js`, and all four PNG icons.

## Resolved since the 2026-02-23 audit

| Concern (2026-02-23) | Resolution | Verification |
| --- | --- | --- |
| Monolithic `cmd/devlog/main.go` (862 lines) [#27] | Commands split into `cmd/devlog/cmd_*.go`; `main.go` now 112 lines | `wc -l cmd/devlog/main.go` |
| Duplicated `WindowConfig`/`PaneConfig` across `internal/config` and `internal/tmux` [#28] | `internal/tmux` imports `internal/config`; types defined once in `internal/config/config.go:34-43` | `rg 'type (Window\|Pane)Config' internal/` |
| Duplicated Chrome/Brave manifest install functions [#14] | Extracted `installChromiumManifest(dir, hostPath, extensionID, label)` helper in `internal/natmsg/manifest.go:106`; both `InstallChromeManifest` and `InstallBraveManifest` delegate to it | `go test ./internal/natmsg/` green |
| Hardcoded `sleep 0.5` in `KillSession` [#29] | Replaced with `time.Sleep(500 * time.Millisecond)` in the Go process (`internal/tmux/tmux.go:208`) | `rg 'time.Sleep\|sleep 0\.5' internal/tmux/tmux.go` |
| Tests use `os.Chdir` (process-global, unsafe under parallel) | Converted to `t.Chdir` (Go 1.24+) in `cmd/devlog/init_test.go` and `cmd/devlog/healthcheck_test.go` | `rg 'os\.Chdir' cmd/devlog/*_test.go` returns no matches |
| Firefox manifest UUID guard was a no-op | Dead `strings.HasPrefix` branch removed; `allowedExts := []string{extensionID}` retained with explanatory comment in `internal/natmsg/manifest.go` | `go test ./internal/natmsg/` green |
| `findConfigFile` unbounded upward walk, no tests | Bounded to `maxFindConfigDepth = 20` (`cmd/devlog/helpers.go:18`); tests added in `cmd/devlog/helpers_test.go` covering current-dir, ancestor, and beyond-depth cases | `go test -run TestFindConfigFile ./cmd/devlog/` green |
| Shell quoting scattered/untested | Centralized in `internal/shellescape` package with unit tests; used by `internal/tmux` and `cmd/devlog/helpers.go` | `go test ./internal/shellescape/` green |
| Native messaging manifest writes were world-readable (0644) | All manifest writes now use `0600` (owner-only) in `internal/natmsg/manifest.go:128,174,214,371` | `rg 'WriteFile.*0[0-9]{3}' internal/natmsg/manifest.go` |
| `cmd/devlog-host/main.go` had no tests | `cmd/devlog-host/main_test.go` added; host loop testable via stream injection | `go test ./cmd/devlog-host/` green |
| `writeBrowserHostWrapper`/`restoreBrowserHostWrapper` untested | Split into testable `*WithHost` cores in `cmd/devlog/helpers.go`; covered by `cmd/devlog/browser_wrapper_test.go` | `go test ./cmd/devlog/` green |
| Manifest lifecycle fragility (stale wrapper after crash) | `natmsg.RepairStaleManifestPaths` self-heals stale/missing manifest paths on next `devlog up`; `refuseClobberActiveWrapper` guards live wrappers | `rg 'RepairStaleManifestPaths' internal/natmsg/manifest.go` |
| Firefox extension diverged from shared root assets | Firefox directory now uses symlinks to shared root files; `package-firefox.sh` copies from canonical source; `page_inject.js` symlink added to both browsers | `unzip -l dist/devlog-firefox-*.zip` lists all manifest-referenced files |

---
*Concerns audit: 2026-07-19*
