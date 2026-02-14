<p align="center">
  <img src="logo.svg" alt="devlog logo" width="120" height="120">
</p>

# devlog

[![CI](https://github.com/jellydn/devlog/actions/workflows/ci.yml/badge.svg)](https://github.com/jellydn/devlog/actions/workflows/ci.yml)
[![Release](https://github.com/jellydn/devlog/actions/workflows/release.yml/badge.svg)](https://github.com/jellydn/devlog/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/jellydn/devlog/branch/main/graph/badge.svg)](https://codecov.io/gh/jellydn/devlog)
[![Go Report Card](https://goreportcard.com/badge/github.com/jellydn/devlog)](https://goreportcard.com/report/github.com/jellydn/devlog)

Zero-code log capture for local development. Captures server logs via tmux and browser console logs via a native messaging host — all controlled by a single YAML config.

## Features

- **tmux log capture** — automatically creates sessions, runs dev commands, and captures stdout/stderr to log files
- **Browser console capture** — Chrome + Firefox extension streams `console.*`, uncaught errors, and unhandled rejections to disk
- **YAML config** — one `devlog.yml` declares everything: commands, panes, log paths, browser filters
- **Environment variable interpolation** — use `$PORT` or `${API_URL}` in your config
- **Zero code changes** — no SDKs, no wrappers, no instrumentation

## Install

### Quick Install (curl)

```sh
curl -fsSL https://raw.githubusercontent.com/jellydn/devlog/main/install.sh | sh
```

Or specify a version and install directory:

```sh
VERSION=v0.1.0 INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/jellydn/devlog/main/install.sh | sh
```

### Go Install

```sh
go install github.com/jellydn/devlog/cmd/devlog@latest
go install github.com/jellydn/devlog/cmd/devlog-host@latest
```

### Browser Extension

> **Coming Soon**: The extension will be available on Chrome Web Store and Firefox Add-ons for easy one-click installation.

For now, load the extension manually:

#### Chrome

1. Open Chrome and navigate to `chrome://extensions/`
2. Enable "Developer mode" (toggle in top right)
3. Click "Load unpacked"
4. Select the `browser-extension/chrome` directory
5. The extension icon should appear in your toolbar

#### Firefox

1. Open Firefox and navigate to `about:debugging`
2. Click "This Firefox" (or "This Nightly" for Developer Edition)
3. Click "Load Temporary Add-on"
4. Select the `browser-extension/firefox` directory
5. The extension icon should appear in your toolbar

#### Brave

1. Open Brave and navigate to `brave://extensions/`
2. Enable "Developer mode" (toggle in top right)
3. Click "Load unpacked"
4. Select the `browser-extension/chrome` directory (Brave uses Chrome extensions)
5. The extension icon should appear in your toolbar

## Quick Start

Create a `devlog.yml` in your project root:

```yaml
version: 1
project: my-app
logs_dir: ./logs
run_mode: timestamped # timestamped | overwrite

# Optional: Automatic log cleanup (for timestamped mode only)
# max_runs: 10        # Keep only the 10 most recent log runs
# retention_days: 30  # OR keep logs from the last 30 days

tmux:
  session: my-app
  windows:
    - name: dev
      panes:
        - name: web
          cmd: npm run dev
          log: server/web.log
        - name: api
          cmd: pnpm --filter api dev
          log: server/api.log

browser:
  native_host: true
  file: browser/console.log
  levels: [error, warn, info, log]
  urls:
    - "http://localhost:*/*"
```

Then:

```sh
devlog healthcheck  # Check system requirements
devlog up
```

## CLI

| Command            | Description                          |
| ------------------ | ------------------------------------ |
| `devlog init`      | Create devlog.yml template           |
| `devlog healthcheck` | Check system requirements          |
| `devlog up`        | Start tmux session + browser logging |
| `devlog down`      | Stop session, flush logs             |
| `devlog attach`    | Attach to the running tmux session   |
| `devlog status`    | Show session state + log paths       |
| `devlog ls`        | List log runs                        |
| `devlog open`      | Open logs directory in file manager  |
| `devlog register`  | Register native messaging host (Chrome, Brave, Firefox) |

`devlog up` will error if a session is already running. Use `devlog down` first.

## Log Output

### Directory Structure

```
logs/
  2026-02-10_17-23-11/
    server/web.log
    server/api.log
    browser/console.log
```

With `run_mode: overwrite`, logs write directly to `logs/` without a timestamp subdirectory.

### Log Cleanup

When using `run_mode: timestamped`, you can configure automatic cleanup of old log directories:

- **`max_runs`**: Keep only the N most recent log runs (e.g., `max_runs: 10`)
- **`retention_days`**: Remove logs older than N days (e.g., `retention_days: 30`)

Both options can be used together. Directories are removed if they exceed `max_runs` OR are older than `retention_days`. Cleanup runs automatically when `devlog up` starts a new session.

### Server Logs

Raw stdout/stderr from each pane:

```
[2026-02-10T17:23:11] npm run dev
[INFO] Server listening on :3000
```

### Browser Logs

Timestamped and level-tagged:

```
[2026-02-10T17:24:02][ERROR] Uncaught TypeError: foo is not a function
[2026-02-10T17:24:05][WARN] Deprecation warning: ...
[2026-02-10T17:24:10][LOG] User clicked submit
```

## Architecture

```
┌─────────────┐      stdout/stderr      ┌──────────────┐
│ tmux panes  │ ──────────────────────▶ │ log files    │
│ (dev cmds)  │                         │ server/*.log │
└─────────────┘                         └──────────────┘
                                                ▲
                        Native Messaging (stdin)│
                                                │
┌───────────────┐  console logs  ┌──────────────┴───────── ┐
│ Browser Tab   │ ─────────────▶ │ Extension + Native Host│
└───────────────┘                │ → browser/*.log        │
                                 └────────────────────────┘
```

- **CLI** (`devlog`): Go binary. Manages tmux sessions, registers native host manifest.
- **Browser Extension**: Manifest V3 (Chrome) + Firefox. Captures console output from matching URLs.
- **Native Host**: Go binary invoked by the extension via Native Messaging protocol. Writes log events to disk.

## Requirements

- Go 1.25+
- tmux
- Chrome, Brave, and/or Firefox

**Verify your setup:**

```sh
devlog healthcheck
```

The healthcheck command verifies:
- tmux is installed and accessible
- devlog-host binary is available
- Browser extension is registered (Chrome, Brave, or Firefox)

If any checks fail, the command provides instructions on how to fix them.

## Contributing

Contributions are welcome! Please see:

- [Browser Extension Store Submission Guide](doc/STORE_SUBMISSION.md) - For publishing to Chrome Web Store and Firefox Add-ons
- [Screenshots Guide](doc/SCREENSHOTS.md) - For creating store submission screenshots

## License

See [LICENSE](LICENSE).
