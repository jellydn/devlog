# devlog

Zero-code log capture for local development. Captures server logs via tmux and browser console logs via a native messaging host — all controlled by a single YAML config.

## Features

- **tmux log capture** — automatically creates sessions, runs dev commands, and captures stdout/stderr to log files
- **Browser console capture** — Chrome + Firefox extension streams `console.*`, uncaught errors, and unhandled rejections to disk
- **YAML config** — one `devlog.yml` declares everything: commands, panes, log paths, browser filters
- **Environment variable interpolation** — use `$PORT` or `${API_URL}` in your config
- **Zero code changes** — no SDKs, no wrappers, no instrumentation

## Install

```sh
go install devlog/cmd/devlog@latest
```

### Browser Extension (manual)

1. Chrome: Load unpacked from `extension/chrome/`
2. Firefox: Load temporary add-on from `extension/firefox/`

## Quick Start

Create a `devlog.yml` in your project root:

```yaml
version: 1
project: my-app
logs_dir: ./logs
run_mode: timestamped   # timestamped | overwrite

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
devlog up
```

## CLI

| Command         | Description                          |
|-----------------|--------------------------------------|
| `devlog up`     | Start tmux session + browser logging |
| `devlog down`   | Stop session, flush logs             |
| `devlog status` | Show session state + log paths       |
| `devlog open`   | Open logs directory in file manager  |

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
│ tmux panes  │ ──────────────────────▶ │ log files     │
│ (dev cmds)  │                         │ server/*.log  │
└─────────────┘                         └──────────────┘
                                                ▲
                        Native Messaging (stdin) │
                                                │
┌───────────────┐  console logs  ┌──────────────┴────────┐
│ Browser Tab   │ ─────────────▶ │ Extension + Native Host│
└───────────────┘                │ → browser/*.log        │
                                 └───────────────────────┘
```

- **CLI** (`devlog`): Go binary. Manages tmux sessions, registers native host manifest.
- **Browser Extension**: Manifest V3 (Chrome) + Firefox. Captures console output from matching URLs.
- **Native Host**: Go binary invoked by the extension via Native Messaging protocol. Writes log events to disk.

## Requirements

- Go 1.25+
- tmux
- Chrome and/or Firefox

## License

See [LICENSE](LICENSE).
