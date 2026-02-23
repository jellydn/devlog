# Codebase Structure

**Analysis Date:** 2026-02-23

## Directory Layout
```
devlog/
├── cmd/
│   ├── devlog/                  # CLI binary entry point
│   │   ├── main.go              # Command dispatcher + all subcommand implementations
│   │   ├── healthcheck_test.go  # Tests for healthcheck command
│   │   ├── init_test.go         # Tests for init command
│   │   └── status_test.go       # Tests for status command
│   └── devlog-host/             # Native messaging host binary
│       └── main.go              # Stdin message loop → logger
├── internal/
│   ├── config/                  # YAML config loading & validation
│   │   ├── config.go            # Config types, Load(), Validate(), CleanupOldRuns()
│   │   └── config_test.go       # 603 lines of table-driven tests
│   ├── tmux/                    # Tmux session management
│   │   ├── tmux.go              # Runner, session/window/pane operations
│   │   ├── tmux_test.go         # Unit tests (289 lines)
│   │   └── integration_test.go  # Integration tests requiring tmux (629 lines)
│   ├── natmsg/                  # Native messaging protocol
│   │   ├── natmsg.go            # Host, Message, wire protocol (length-prefixed JSON)
│   │   ├── natmsg_test.go       # Protocol tests (367 lines)
│   │   ├── manifest.go          # Browser manifest install/update for Chrome/Brave/Firefox
│   │   └── manifest_test.go     # Manifest tests (227 lines)
│   └── logger/                  # Log file writer with level filtering
│       ├── logger.go            # Logger struct, formatted line output
│       └── logger_test.go       # Logger tests (323 lines)
├── browser-extension/           # Browser extension (Chrome + Firefox)
│   ├── background.js            # Service worker: native messaging port, message routing
│   ├── content_script.js        # Content script: bridges page_inject ↔ background
│   ├── page_inject.js           # Page-context script: wraps console.* methods
│   ├── popup.html               # Extension popup UI
│   ├── popup.js                 # Popup logic
│   ├── icons/                   # Extension icons (SVG + PNG)
│   ├── chrome/                  # Chrome-specific files
│   │   ├── manifest.json        # Manifest V3
│   │   └── page_inject.js       # Chrome-specific page inject
│   └── firefox/                 # Firefox-specific files
│       ├── manifest.json        # Manifest V2
│       ├── background.js        # Firefox background script
│       ├── content_script.js    # Firefox content script
│       ├── page_inject.js       # Firefox page inject
│       ├── popup.html           # Firefox popup
│       ├── popup.js             # Firefox popup logic
│       └── icons/               # Firefox icons
├── doc/                         # Documentation
│   ├── adr/                     # Architecture Decision Records (6 ADRs)
│   ├── PUBLICATION_CHECKLIST.md # Extension store publication guide
│   ├── SCREENSHOTS.md           # Screenshot guidelines
│   └── STORE_SUBMISSION.md      # Store submission details
├── scripts/                     # Build/packaging scripts
│   ├── package-chrome.sh        # Chrome extension packaging
│   ├── package-firefox.sh       # Firefox extension packaging
│   ├── validate-screenshots.sh  # Screenshot dimension validator
│   └── ralph/                   # Ralph autonomous agent config
├── dist/                        # Packaged extension ZIPs (build output)
├── go.mod                       # Go module: github.com/jellydn/devlog (Go 1.25+)
├── go.sum                       # Dependency checksums
├── justfile                     # Task runner (build, test, lint, dev)
├── devlog.yml.example           # Example configuration template
├── .goreleaser.yml              # GoReleaser config for binary releases
├── .github/workflows/           # CI/CD (ci.yml, release.yml)
├── AGENTS.md                    # AI agent guidelines
├── CLAUDE.md                    # Claude-specific instructions
├── README.md                    # Project documentation
├── PRIVACY.md                   # Privacy policy
└── renovate.json                # Dependency update config
```

## Directory Purposes

**`cmd/devlog/`:**
- Purpose: Main CLI application — all user commands in a single file
- Contains: Command dispatcher, 10 subcommand implementations, helper functions
- Key files: `main.go` (862 lines — all CLI logic)

**`cmd/devlog-host/`:**
- Purpose: Standalone binary launched by browsers via native messaging
- Contains: Message read loop that bridges native messaging → logger
- Key files: `main.go` (86 lines)

**`internal/config/`:**
- Purpose: YAML configuration parsing, validation, env var interpolation, log retention cleanup
- Contains: Config structs with YAML tags, Load/Validate pipeline
- Key files: `config.go` (230 lines), `config_test.go` (603 lines)

**`internal/tmux/`:**
- Purpose: Tmux subprocess management — session creation, window/pane setup, pipe-pane logging, session teardown
- Contains: `Runner` type wrapping `os/exec` calls to `tmux` binary
- Key files: `tmux.go` (367 lines), `integration_test.go` (629 lines, requires `-tags=integration`)

**`internal/natmsg/`:**
- Purpose: Native Messaging wire protocol and browser manifest management
- Contains: Length-prefixed JSON encoding/decoding, manifest JSON generation for Chrome/Brave/Firefox/Zen, binary discovery
- Key files: `natmsg.go` (226 lines), `manifest.go` (314 lines)

**`internal/logger/`:**
- Purpose: Formatted log writing with level-based filtering and thread-safe file access
- Contains: `Logger` struct with mutex, append-only file writes
- Key files: `logger.go` (131 lines)

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
- `internal/config/config.go`: Config loading, validation, env interpolation, retention cleanup
- `internal/tmux/tmux.go`: Tmux session lifecycle management
- `internal/natmsg/natmsg.go`: Native Messaging wire protocol
- `internal/natmsg/manifest.go`: Browser manifest installation
- `internal/logger/logger.go`: Log file writer with level filtering

**Tests:**
- `internal/config/config_test.go`: Config loading/validation tests
- `internal/tmux/tmux_test.go`: Tmux unit tests
- `internal/tmux/integration_test.go`: Tmux integration tests (build tag: `integration`)
- `internal/natmsg/natmsg_test.go`: Protocol encoding/decoding tests
- `internal/natmsg/manifest_test.go`: Manifest generation tests
- `internal/logger/logger_test.go`: Logger formatting and filtering tests
- `cmd/devlog/healthcheck_test.go`: Healthcheck command tests
- `cmd/devlog/init_test.go`: Init command tests
- `cmd/devlog/status_test.go`: Status command tests

## Naming Conventions

**Files:**
- Go source: `snake_case.go` (e.g., `config.go`, `natmsg.go`, `integration_test.go`)
- Test files: `*_test.go` colocated with source
- JS files: `snake_case.js` (e.g., `content_script.js`, `page_inject.js`)
- Scripts: `kebab-case.sh` (e.g., `package-chrome.sh`)

**Directories:**
- Go packages: lowercase single-word (e.g., `config`, `tmux`, `natmsg`, `logger`)
- Commands: kebab-case binary names (e.g., `devlog`, `devlog-host`)
- Browser: `browser-extension/` with `chrome/` and `firefox/` subdirs

**Go Naming:**
- Exported: `CamelCase` (e.g., `NewRunner`, `CreateSession`, `SessionInfo`)
- Unexported: `camelCase` (e.g., `cmdUp`, `sendCommandWithLogging`)
- Test functions: `TestFunctionName_Description` (e.g., `TestLoad_ValidConfig`)

## Where to Add New Code

**New CLI Command:**
- Add handler function in `cmd/devlog/main.go`
- Add to `commands` map in `cmd/devlog/main.go`
- Add to `usage` string in `cmd/devlog/main.go`
- Add tests in `cmd/devlog/<command>_test.go`

**New Internal Package:**
- Create `internal/<package>/` directory
- Follow existing patterns: exported types + constructor + methods
- Add `*_test.go` colocated with source

**New Browser Extension Feature:**
- Chrome: modify files in `browser-extension/` (shared) or `browser-extension/chrome/`
- Firefox: modify files in `browser-extension/firefox/`
- Update `manifest.json` in both `chrome/` and `firefox/` directories

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

**`scripts/ralph/`:**
- Purpose: Configuration for the Ralph autonomous agent (PRD, prompts, progress tracking)

**`.github/workflows/`:**
- Purpose: GitHub Actions CI/CD pipelines
- Contains: `ci.yml` (lint + test), `release.yml` (GoReleaser)

---
*Structure analysis: 2026-02-23*
