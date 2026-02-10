# System Architecture

**Analysis Date:** 2026-02-10

## High-Level Pattern

**Architecture Type:** CLI Tool with Browser Extension
- Native messaging bridge between browser and local tmux sessions
- Dual-process architecture: main CLI + native messaging host
- Browser extension injects page scripts to capture console logs

## Core Components

```
┌─────────────────────────────────────────────────────────────────┐
│                         User Interface                          │
│  CLI Commands: init, up, down, attach, status, ls, open, register  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Main CLI (devlog)                          │
│  - Config loading (YAML with env interpolation)                 │
│  - Command routing and validation                               │
│  - Tmux session management                                     │
│  - Browser wrapper script generation                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                ┌───────────────┴───────────────┐
                ▼                               ▼
┌──────────────────────┐        ┌──────────────────────────────┐
│   Tmux Integration   │        │   Browser Extension Bridge   │
│                      │        │                              │
│ - Session creation   │        │ - Native messaging protocol │
│ - Pane management    │        │ - Wrapper script management │
│ - Log capture        │        │ - Manifest installation     │
└──────────────────────┘        └──────────────────────────────┘
                                        │
                                        ▼
                              ┌─────────────────────────┐
                              │  Native Messaging Host   │
                              │  (devlog-host)           │
                              │                          │
                              │ - JSON message parsing   │
                              │ - Log level filtering    │
                              │ - File writing           │
                              └─────────────────────────┘
```

## Layers & Data Flow

**Configuration Layer:**
- `internal/config/` - YAML parsing with environment variable interpolation
- Config resolution for logs directory (timestamped/overwrite modes)

**Tmux Abstraction Layer:**
- `internal/tmux/` - Session, window, and pane management
- Log capture via `pipe-pane` with shell command execution
- Environment variable storage for cross-command communication

**Messaging Layer:**
- `internal/natmsg/` - Native messaging protocol implementation
- Length-prefixed JSON messages over stdin/stdout
- Type-safe message structs with `interface{}` for flexible timestamp/line handling

**Logging Layer:**
- `internal/logger/` - Browser console log filtering and formatting
- Thread-safe file writing with mutex protection
- Type conversion utilities for JSON-unmarshaled numeric values

**Browser Extension Layer:**
- Page injection via web_accessible_resources (CSP-safe)
- Console wrapping via postMessage bridge
- Content script → background script → native host messaging chain

## Entry Points

**Main CLI Entry Point:**
- `cmd/devlog/main.go` - CLI application entry
- Command dispatch via `map[string]Command` registry
- Commands that need config vs commands that don't

**Native Messaging Host Entry:**
- `cmd/devlog-host/main.go` - Native messaging host entry
- Reads log path and levels from command line arguments
- Continuous message loop until stdin EOF

**Browser Extension Entry Points:**
- `chrome/page_inject.js` - Injected into page context
- `content_script.js` - Bridge between page and background
- Background script handles native messaging

## Key Abstractions

**Command Pattern:**
```go
type Command func(cfg *config.Config, args []string) error
```
- All commands implement this signature
- Nil config for commands that don't require YAML config

**Tmux Runner:**
```go
type Runner struct {
    sessionName string
}
```
- Encapsulates all tmux operations
- Methods for session creation, pane management, log capture

**Native Messaging Protocol:**
```go
type Message struct {
    Type      string      `json:"type"`
    Level     string      `json:"level"`
    Message   string      `json:"message"`
    URL       string      `json:"url"`
    Timestamp interface{} `json:"timestamp"`  // Flexible for JSON numbers
    Source    string      `json:"source,omitempty"`
    Line      interface{} `json:"line,omitempty"`
    Column    interface{} `json:"column,omitempty"`
}
```

## Data Flow

**Session Startup Flow:**
1. `devlog up` → Load YAML config
2. Resolve logs directory (create if timestamped mode)
3. Create tmux session with windows/panes
4. Set up `pipe-pane` logging for each pane
5. Generate browser host wrapper script with session-specific log path
6. Update native messaging manifest to point to wrapper

**Browser Log Flow:**
1. Page script captures console.{log,error,warn,info}
2. PostMessage to content script with __devlog flag
3. Content script forwards to background script via chrome.runtime.sendMessage
4. Background script sends to native host via native messaging
5. Native host parses JSON, filters by level, writes to log file

## State Management

**Tmux State:**
- Session stored in tmux server process
- `DEVLOG_LOGS_DIR` environment variable for cross-command communication
- Session name used for all tmux commands

**Browser State:**
- Extension stores enabled/disabled state and log levels
- Configuration updates propagate via `CONFIG_UPDATED` messages

**Local State:**
- No persistent state beyond files (YAML config, log files)
- Each command is stateless (reads from tmux/environment)

## Security Boundaries

**Native Messaging:**
- Browser restricts which extensions can communicate with which hosts
- Manifest must explicitly whitelist extension IDs
- Host binary must be in expected location

**Shell Command Execution:**
- All shell-escaped paths prevent command injection
- Absolute paths stored in environment to avoid path resolution issues

---

*Architecture analysis: 2026-02-10*
