# Directory Structure

**Analysis Date:** 2026-02-10

## Layout Overview

```
.
├── cmd/                        # Application entry points
│   ├── devlog/                # Main CLI application
│   │   └── main.go            # CLI entry point with command routing
│   └── devlog-host/           # Native messaging host
│       └── main.go            # Host entry point for browser communication
│
├── internal/                  # Private packages (no external imports)
│   ├── config/                # Configuration management
│   │   ├── config.go          # YAML loading with env interpolation
│   │   └── config_test.go     # Config parsing tests
│   ├── logger/                # Browser console log handling
│   │   ├── logger.go          # Log filtering, formatting, file writing
│   │   └── logger_test.go     # Logger and type conversion tests
│   ├── natmsg/                # Native messaging protocol
│   │   ├── natmsg.go          # Message types, protocol implementation
│   │   ├── manifest.go        # Browser manifest installation
│   │   └── natmsg_test.go     # Protocol tests
│   └── tmux/                  # Tmux session management
│       ├── tmux.go            # Session, pane, window operations
│       └── tmux_test.go       # Tmux operations tests
│
├── browser-extension/         # Browser extension source
│   ├── chrome/                # Chrome-specific files
│   │   ├── manifest.json      # Chrome extension manifest
│   │   └── page_inject.js    # Page console capture (Chrome variant)
│   ├── firefox/               # Firefox-specific files
│   │   ├── manifest.json      # Firefox extension manifest
│   │   ├── background.js      # Firefox background script
│   │   ├── content_script.js  # Firefox content script
│   │   └── page_inject.js    # Page console capture (Firefox variant)
│   ├── page_inject.js        # Shared page injection logic
│   └── content_script.js     # Shared content script logic
│
├── .planning/                 # Planning and documentation
│   └── codebase/             # Generated codebase documentation
│
├── devlog.yml.example         # Example configuration template
├── go.mod                     # Go module definition
├── go.sum                     # Go dependency lock file
├── justfile                   # Build automation commands
├── AGENTS.md                  # Project guidelines for AI agents
├── CLAUDE.md                  # Project memory for Claude Code
└── .goreleaser.yml           # Release configuration
```

## Key Locations

**Configuration:**
- `internal/config/config.go:67` - `Load()` function for YAML parsing
- `internal/config/config.go:25` - `ResolveLogsDir()` for path resolution
- `devlog.yml.example:1` - Template for new configurations

**Tmux Operations:**
- `internal/tmux/tmux.go:31` - `CreateSession()` for tmux setup
- `internal/tmux/tmux.go:141` - `KillSession()` for graceful shutdown
- `internal/tmux/tmux.go:210` - `GetSessionInfo()` for status reporting

**Native Messaging:**
- `internal/natmsg/natmsg.go:56` - `ReadMessage()` protocol parsing
- `internal/natmsg/natmsg.go:93` - `WriteResponse()` protocol writing
- `internal/natmsg/manifest.go:78` - Chrome manifest installation
- `internal/natmsg/manifest.go:105` - Firefox manifest installation

**Logging:**
- `internal/logger/logger.go:29` - `New()` logger constructor
- `internal/logger/logger.go:77` - `Log()` main logging function
- `internal/logger/logger.go:121` - `formatTimestamp()` type conversion

**Browser Extension:**
- `browser-extension/content_script.js:1` - Main bridge script
- `browser-extension/page_inject.js:1` - Page-level console wrapper
- `browser-extension/chrome/manifest.json:1` - Chrome extension config
- `browser-extension/firefox/manifest.json:1` - Firefox extension config

## Naming Conventions

**Packages:**
- Lowercase, single words: `config`, `tmux`, `natmsg`, `logger`
- No underscores or mixed case in package names

**Files:**
- `main.go` for package main entry points
- `{package}_test.go` for test files in same package
- `manifest.json` for browser extension manifests
- `page_inject.js` and `content_script.js` for extension scripts

**Go Code:**
- **Types:** PascalCase - `Runner`, `Config`, `Message`, `Logger`
- **Interfaces:** Usually not used, concrete structs preferred
- **Functions:** CamelCase - `NewRunner`, `CreateSession`, `WriteResponse`
- **Constants:** PascalCase for exported, camelCase for unexported
- **Methods:** PascalCase receivers - `(r *Runner) SessionExists()`

**JavaScript:**
- camelCase for functions and variables
- PascalCase for constructors (rare)
- Constants: UPPER_SNAKE_CASE

## File Organization Patterns

**Command Structure:**
- Each top-level command has a `cmd{Name}` function in `main.go`
- Commands that need config receive `*config.Config`
- Commands that don't need config (init, register) receive `nil`

**Internal Package Structure:**
- Each package has a single main file (e.g., `tmux.go`, `logger.go`)
- Test files alongside implementation
- Types used by multiple packages defined in their respective packages

**Browser Extension Structure:**
- Platform-specific subdirectories (`chrome/`, `firefox/`)
- Shared files at root level for common logic
- Manifest files in platform-specific directories

## Import Conventions

**Standard library first**, then third-party:
```go
import (
    "fmt"
    "os"

    "gopkg.in/yaml.v3"
)
```

**Internal imports use full module path:**
```go
import (
    "github.com/jellydn/devlog/internal/config"
    "github.com/jellydn/devlog/internal/tmux"
)
```

---

*Structure analysis: 2026-02-10*
