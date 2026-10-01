# Codebase Structure

**Analysis Date:** 2026-07-19 (regenerated)

## Directory Layout
```text
devlog/
├── cmd/
│   ├── devlog/                  # CLI binary entry point
│   │   ├── main.go              # Command dispatcher + usage
│   │   ├── browser_session_adapter.go  # DI adapters for manifest/tmux → browsersession interfaces
│   │   ├── cmd_attach.go        # `devlog attach`
│   │   ├── cmd_down.go          # `devlog down`
│   │   ├── cmd_healthcheck.go   # `devlog healthcheck`
│   │   ├── cmd_init.go          # `devlog init`
│   │   ├── cmd_ls.go            # `devlog ls`
│   │   ├── cmd_open.go          # `devlog open`
│   │   ├── cmd_register.go      # `devlog register`
│   │   ├── cmd_status.go        # `devlog status`
│   │   ├── cmd_up.go            # `devlog up`
│   │   ├── helpers.go           # findConfigFile, ensureFileExists
│   │   ├── healthcheck_test.go  # Tests for healthcheck command
│   │   ├── init_test.go         # Tests for init command
│   │   └── status_test.go       # Tests for status command
│   └── devlog-host/             # Native messaging host binary
│       ├── main.go              # Stdin message loop → logger
│       └── main_test.go         # Host loop tests with stream injection
├── internal/
│   ├── config/                  # YAML config loading & validation
│   │   ├── config.go            # Config types, Load(), Validate()
│   │   └── config_test.go       # Table-driven tests
│   ├── tmux/                    # Tmux session management
│   │   ├── tmux.go              # Runner, SessionConfig, session/window/pane operations
│   │   ├── tmux_test.go         # Unit tests
│   │   └── integration_test.go  # Integration tests requiring tmux (-tags=integration)
│   ├── natmsg/                  # Native messaging wire protocol
│   │   ├── natmsg.go            # Host, Message, wire protocol (length-prefixed JSON)
│   │   └── natmsg_test.go       # Protocol tests
│   ├── manifest/                # Browser manifest registration (extracted from natmsg)
│   │   ├── manifest.go          # ChromeManifest, FirefoxManifest, install/update/repair
│   │   ├── manifest_test.go     # Manifest tests
│   │   ├── validate_host_unix.go    # ValidateHostPath (Unix, build-tagged)
│   │   └── validate_host_windows.go # ValidateHostPath (Windows, build-tagged)
│   ├── browsersession/          # Browser-log capture lifecycle (extracted from helpers.go)
│   │   ├── session.go           # Session, ManifestOps/SessionChecker interfaces, HealthCheck
│   │   ├── browsersession_test.go   # Round-trip, stale recovery, clobber tests
│   │   └── helpers_test.go      # Sanitize, wrapper path, script generation tests
│   ├── logrotate/               # Log directory cleanup (extracted from config)
│   │   ├── logrotate.go         # Policy, Result, Cleanup
│   │   └── logrotate_test.go    # Cleanup tests (max runs, retention days, dry run)
│   ├── fileutil/                # Shared filesystem helpers
│   │   ├── touchfile.go         # TouchFile (MkdirAll + OpenFile + Close)
│   │   └── touchfile_test.go    # TouchFile tests
│   ├── logger/                  # Log file writer with level filtering
│   │   ├── logger.go            # Logger struct, formatted line output
│   │   └── logger_test.go       # Logger tests
│   └── shellescape/             # POSIX shell quoting
│       ├── shellescape.go       # Quote function
│       └── shellescape_test.go  # Shell quoting tests
├── browser-extension/           # Browser extension (Chrome + Firefox share root assets)
│   ├── background.js            # Service worker: native messaging port, message routing
│   ├── content_script.js        # Content script: bridges page_inject ↔ background
│   ├── page_inject.js           # Page-context script: wraps console.* methods
│   ├── popup.html               # Extension popup UI
│   ├── popup.js                 # Popup logic
│   ├── VERSION                  # Extension version (independent of Go binary version)
│   ├── icons/                   # Extension icons (SVG + PNG)
│   ├── chrome/                  # Chrome-specific manifest only; shared assets via symlinks
│   │   ├── manifest.json        # Manifest V3
│   │   ├── background.js → ../background.js
│   │   ├── content_script.js → ../content_script.js
│   │   ├── page_inject.js → ../page_inject.js
│   │   ├── popup.html → ../popup.html
│   │   ├── popup.js → ../popup.js
│   │   └── icons → ../icons
│   ├── firefox/                 # Firefox-specific manifest only; shared assets via symlinks
│   │   ├── manifest.json        # Manifest V2
│   │   ├── background.js → ../background.js
│   │   ├── content_script.js → ../content_script.js
│   │   ├── page_inject.js → ../page_inject.js
│   │   ├── popup.html → ../popup.html
│   │   ├── popup.js → ../popup.js
│   │   └── icons → ../icons
│   └── test/
│       ├── vitest.config.js     # Extension test configuration
│       ├── background.test.js   # Background service worker tests
│       ├── content_script.test.js   # Content script tests
│       ├── page_inject.test.js  # Page inject tests
│       └── mocks/
│           └── chrome.js        # Chrome API mocks
├── doc/                         # Documentation
│   ├── adr/                     # Architecture Decision Records (6 ADRs)
│   ├── PUBLICATION_CHECKLIST.md # Extension store publication guide
│   ├── SCREENSHOTS.md           # Screenshot guidelines
│   └── STORE_SUBMISSION.md      # Store submission details
├── scripts/                     # Build/packaging scripts
│   ├── package-chrome.sh        # Chrome extension packaging
│   ├── package-firefox.sh       # Firefox extension packaging
│   └── validate-screenshots.sh  # Screenshot dimension validator
├── dist/                        # Packaged extension ZIPs (build output)
├── go.mod                       # Go module: github.com/jellydn/devlog (Go 1.25+)
├── go.sum                       # Dependency checksums
├── justfile                     # Task runner (build, test, lint, dev)
├── devlog.yml.example           # Example configuration template
├── .goreleaser.yml              # GoReleaser config for binary releases
├── .planning/                   # Planning and codebase analysis docs
│   └── codebase/                # Auto-generated codemap analysis
├── .github/workflows/           # CI/CD (ci.yml, release.yml)
├── AGENTS.md                    # AI agent guidelines
├── README.md                    # Project documentation
├── PRIVACY.md                   # Privacy policy
└── renovate.json                # Dependency update config
```

## Directory Purposes

**`cmd/devlog/`:**
- Purpose: Main CLI application — one command per file plus shared helpers and DI adapters
- Contains: Command dispatcher (`main.go`), one `cmd_*.go` per subcommand, `helpers.go` (findConfigFile, ensureFileExists), `browser_session_adapter.go` (adapters for ManifestOps/SessionChecker interfaces)
- Key files: `main.go` (dispatcher + usage), `cmd_up.go`/`cmd_down.go`/`cmd_register.go` (primary user-facing commands)

**`cmd/devlog-host/`:**
- Purpose: Standalone binary launched by browsers via native messaging
- Contains: Message read loop that bridges native messaging → logger
- Key files: `main.go`, `main_test.go` (host loop tests with stream injection)

**`internal/config/`:**
- Purpose: YAML configuration parsing, validation, env var interpolation
- Contains: Config structs with YAML tags, Load/Validate pipeline
- Key files: `config.go`, `config_test.go`

**`internal/tmux/`:**
- Purpose: Tmux subprocess management — session creation, window/pane setup, pipe-pane logging, session teardown. Owns log-dir resolution via SessionConfig.
- Contains: `Runner` type wrapping `os/exec` calls to `tmux` binary; `SessionConfig` bundling session params
- Key files: `tmux.go`, `integration_test.go` (requires `-tags=integration`)

**`internal/natmsg/`:**
- Purpose: Native Messaging wire protocol (length-prefixed JSON over stdin/stdout)
- Contains: `Host`, `Message`, `Response` types; `ReadMessage`, `WriteResponse`, `SendAck`
- Key files: `natmsg.go`

**`internal/manifest/`:**
- Purpose: Browser manifest registration — install, update, repair for Chrome/Brave/Firefox/Zen
- Contains: `ChromeManifest`, `FirefoxManifest` structs; install/update/repair functions; OS-specific `ValidateHostPath`
- Key files: `manifest.go`, `validate_host_unix.go`, `validate_host_windows.go`

**`internal/browsersession/`:**
- Purpose: Browser-log capture lifecycle — wrapper creation, clobber protection, health checks
- Contains: `Session` struct with `Start`/`Stop`/`HealthCheck`; `ManifestOps` and `SessionChecker` interfaces for DI; wrapper script generation
- Key files: `session.go`, `browsersession_test.go`

**`internal/logrotate/`:**
- Purpose: Log directory retention cleanup for timestamped run mode
- Contains: `Policy`, `Result`, `Cleanup` function
- Key files: `logrotate.go`, `logrotate_test.go`

**`internal/fileutil/`:**
- Purpose: Shared filesystem utility functions
- Contains: `TouchFile` — ensures a file exists
- Key files: `touchfile.go`, `touchfile_test.go`

**`internal/logger/`:**
- Purpose: Formatted log writing with level-based filtering and thread-safe file access
- Contains: `Logger` struct with mutex, append-only file writes
- Key files: `logger.go`

**`internal/shellescape/`:**
- Purpose: POSIX shell single-quote escaping for command injection prevention
- Contains: `Quote` function
- Key files: `shellescape.go`, `shellescape_test.go`

**`browser-extension/`:**
- Purpose: Browser extension for Chrome (Manifest V3) and Firefox (Manifest V2)
- Contains: Console interception, content script bridging, native messaging client, popup UI
- Key files: `background.js`, `content_script.js`, `page_inject.js`

**`doc/adr/`:**
- Purpose: Architecture Decision Records documenting key design choices
- Contains: 6 ADRs covering Go choice, YAML config, tmux usage, native messaging, append-only logs, timestamped dirs

## Key File Locations

**Entry Points:**
- `cmd/devlog/main.go`: CLI binary — `devlog` command
- `cmd/devlog-host/main.go`: Native messaging host binary — `devlog-host` command
- `browser-extension/background.js`: Browser extension service worker

**Configuration:**
- `devlog.yml.example`: Example configuration template
- `go.mod`: Go module definition (single dependency: `gopkg.in/yaml.v3`)
- `justfile`: Task runner recipes (build, test, lint, dev, CI)
- `.goreleaser.yml`: Release build configuration
- `.github/workflows/ci.yml`: CI pipeline
- `.github/workflows/release.yml`: Release pipeline

**Core Logic:**
- `internal/config/config.go`: Config loading, validation, env interpolation
- `internal/tmux/tmux.go`: Tmux session lifecycle management with SessionConfig
- `internal/natmsg/natmsg.go`: Native Messaging wire protocol
- `internal/manifest/manifest.go`: Browser manifest installation
- `internal/browsersession/session.go`: Browser-log wrapper lifecycle
- `internal/logrotate/logrotate.go`: Log directory retention cleanup
- `internal/logger/logger.go`: Log file writer with level filtering
- `internal/fileutil/touchfile.go`: Shared file creation helper
- `internal/shellescape/shellescape.go`: Shell argument escaping

**Tests:**
- `internal/config/config_test.go`: Config loading/validation tests
- `internal/tmux/tmux_test.go`: Tmux unit tests
- `internal/tmux/integration_test.go`: Tmux integration tests (build tag: `integration`)
- `internal/natmsg/natmsg_test.go`: Protocol encoding/decoding tests
- `internal/manifest/manifest_test.go`: Manifest generation tests
- `internal/browsersession/browsersession_test.go`: Session lifecycle, round-trip, stale recovery tests
- `internal/browsersession/helpers_test.go`: Sanitize, path, script generation tests
- `internal/logrotate/logrotate_test.go`: Cleanup policy tests
- `internal/fileutil/touchfile_test.go`: TouchFile tests
- `internal/logger/logger_test.go`: Logger formatting and filtering tests
- `internal/shellescape/shellescape_test.go`: Shell quoting tests
- `cmd/devlog/healthcheck_test.go`: Healthcheck command tests
- `cmd/devlog/init_test.go`: Init command tests
- `cmd/devlog/status_test.go`: Status command tests
- `cmd/devlog-host/main_test.go`: Host message loop tests with stream injection
- `browser-extension/test/`: Vitest tests for `background.js`, `content_script.js`, `page_inject.js`

## Naming Conventions

**Files:**
- Go source: lowercase, no separators (`config.go`, `session.go`, `tmux.go`)
- Test files: `*_test.go` colocated with source
- Build-tagged files: platform suffix (`validate_host_unix.go`, `validate_host_windows.go`)
- JS files: `snake_case.js` (e.g., `content_script.js`, `page_inject.js`)
- Scripts: `kebab-case.sh` (e.g., `package-chrome.sh`)

**Directories:**
- Go packages: lowercase single-word (e.g., `config`, `tmux`, `manifest`, `browsersession`)
- Commands: kebab-case binary names (e.g., `devlog`, `devlog-host`)
- Browser: `browser-extension/` with `chrome/` and `firefox/` subdirs

**Go Naming:**
- Exported: `CamelCase` (e.g., `NewRunner`, `CreateSession`, `SessionInfo`, `TouchFile`)
- Unexported: `camelCase` (e.g., `cmdUp`, `sendCommandWithLogging`, `discoverHost`)
- Test functions: `TestFunctionName_Description` (e.g., `TestLoad_ValidConfig`, `TestCleanup_MaxRuns`)
- Interfaces: Noun or -er suffix (e.g., `ManifestOps`, `SessionChecker`)

## Where to Add New Code

**New CLI Command:**
- Add handler function in `cmd/devlog/cmd_<name>.go`
- Add to `commands` map and `usage` string in `cmd/devlog/main.go`
- Add tests in `cmd/devlog/<command>_test.go` (use `t.Chdir` for directory-scoped tests)

**New Internal Package:**
- Create `internal/<package>/` directory
- Follow existing patterns: exported types + constructor + methods
- Add `*_test.go` colocated with source
- Add package-level doc comment (`// Package <name> ...`)

**New Browser Extension Feature:**
- Modify shared root files in `browser-extension/` (used by both Chrome and Firefox via symlinks)
- Update `manifest.json` in both `chrome/` and `firefox/` directories only if permissions or resources change
- Packaging (`scripts/package-*.sh`) copies shared files from `browser-extension/` into the archive; do not duplicate assets in `chrome/` or `firefox/`

**New Config Field:**
- Add struct field with YAML tag in `internal/config/config.go`
- Add validation in `Config.Validate()`
- Update `devlog.yml.example`
- Add tests in `internal/config/config_test.go`

## Special Directories

**`dist/`:**
- Purpose: Build output for packaged browser extensions (ZIP files)
- Generated by: `scripts/package-chrome.sh` and `scripts/package-firefox.sh`

**`doc/adr/`:**
- Purpose: Architecture Decision Records documenting rationale for key design choices
- Convention: Numbered `NNNN-title.md` format

**`.planning/`:**
- Purpose: Planning documents and auto-generated codebase analysis
- Contains: `plan.md` (feature plans), `codebase/` (codemap analysis)

**`.github/workflows/`:**
- Purpose: GitHub Actions CI/CD pipelines
- Contains: `ci.yml` (lint + test + multi-platform), `release.yml` (GoReleaser binary release)

---
*Structure analysis: 2026-07-19 (regenerated)*
