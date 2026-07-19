# 7. Package Extraction Refactor (Manifest, BrowserSession, Logrotate, Fileutil)

Date: 2026-07-19

## Status

Accepted

## Context

After the initial implementation, several internal packages had grown to mix unrelated responsibilities:

- `internal/natmsg` contained both the wire protocol (length-prefixed JSON over stdin/stdout) and browser manifest registration (install/update/repair for Chrome, Brave, Firefox). These are separate concerns — one is a protocol, the other is OS-level configuration management.
- `internal/config` contained both configuration loading/validation and log directory cleanup logic (`CleanupOldRuns`). Cleanup is an operational concern, not a configuration concern.
- `cmd/devlog/helpers.go` contained 12+ functions for browser-log wrapper lifecycle: script generation, wrapper path management, clobber protection, manifest repair, and health checks. These were orchestrated ad-hoc across `cmd_up.go`, `cmd_down.go`, and `cmd_healthcheck.go`.

Additionally, the `EnsurePaneLogFiles` function in `internal/tmux` and `ensureFileExists` in `cmd/devlog/helpers.go` shared identical MkdirAll+OpenFile+Close logic, duplicated across package boundaries.

Options considered:

- **Keep as-is**: Accept the mixed responsibilities. Simpler in the short term but makes testing harder and creates unclear ownership.
- **Extract into focused packages**: Split natmsg into `internal/natmsg` (protocol) and `internal/manifest` (browser registration). Extract cleanup from config into `internal/logrotate`. Extract wrapper lifecycle from helpers into `internal/browsersession`. Extract shared file utility into `internal/fileutil`.
- **Merge everything into one package**: Opposite direction — put all logic into a single `internal/core` package. Would create a monolith with no clear boundaries.

## Decision

Extract into focused single-responsibility packages following these dependency rules:

1. **`internal/manifest`** (extracted from `internal/natmsg`): Browser manifest registration only. Does not import natmsg. natmsg does not import manifest — no circular dependency.

2. **`internal/browsersession`** (extracted from `cmd/devlog/helpers.go`): Browser-log capture lifecycle with a clean public API (`Session.Start`, `Session.Stop`, `Session.HealthCheck`). Uses interface-based dependency injection (`ManifestOps`, `SessionChecker`) for testability. Does not import `internal/config` or `cmd/devlog`.

3. **`internal/logrotate`** (extracted from `internal/config`): Log directory cleanup based on retention policy (`Policy{MaxRuns, RetentionDays}`). `internal/config` drops the `sort` import. Later, after `tmux.Runner` absorbs `ResolveLogsDir`, config also drops the `time` import.

4. **`internal/fileutil`** (new): Shared `TouchFile` helper eliminating duplication between `cmd/devlog/helpers.go` and `internal/tmux/tmux.go`.

This follows the existing pattern established by `internal/shellescape` — small, focused utility packages with a single clear responsibility.

## Consequences

### Positive

- **Clear ownership**: Each package has one reason to change (Single Responsibility)
- **Testability**: `browsersession.Session` accepts interfaces, enabling fake injection in tests without real filesystem or tmux
- **No circular dependencies**: manifest ↔ natmsg are independent; browsersession imports neither config nor cmd/devlog
- **Reduced duplication**: `fileutil.TouchFile` used by both `cmd/devlog/helpers.go` and `internal/tmux/tmux.go`
- **Smaller files**: `helpers.go` shrunk from ~255 lines to ~40 lines; `HealthCheck` method broken into three focused helpers
- **config is cleaner**: `internal/config` no longer mixes retention logic with config types — it only handles loading, validation, and env interpolation

### Negative

- **More packages**: 4 new internal packages increase the directory count. Navigation overhead is minimal (each package is 1-3 files).
- **Interface boilerplate**: `ManifestOps` and `SessionChecker` interfaces require thin adapter types in `cmd/devlog/browser_session_adapter.go`. This is standard Go DI and worth the testability gain. The final `ManifestOps` interface includes three browser-directory getter methods (`GetChromeNativeMessagingDir`, `GetBraveNativeMessagingDir`, `GetFirefoxNativeMessagingDirs`) not in the original design — these are needed by `HealthCheck` to discover registered browsers.
- **Error message granularity trade-off**: `ensurePaneLogFiles` previously had three distinct error messages (directory creation, file creation, file close). After using `fileutil.TouchFile`, these collapse into one. The underlying OS error is still preserved via `%w` wrapping.
