# AGENTS.md

> Guidelines for AI agents working on the devlog project

## Commands

```bash
just devlog-dev     # Build both binaries + symlink to ~/.local/bin
just test-one TestLoad_ValidConfig   # Run a single test by name
just lint           # go fmt + go vet
just ci             # lint + test
go test -tags=integration ./internal/tmux/   # Integration tests (need tmux installed)
go test -short ./...                        # Skip integration tests
just init           # Copy devlog.yml.example -> devlog.yml
```

`just ci` runs `lint` then `test`. CI additionally runs `go test -race`, integration tests, and builds both binaries; it installs `tmux` first.

## Project layout

- `cmd/devlog` — the CLI (`up`, `down`, `attach`, `status`, `ls`, `open`, `register`, `init`, `healthcheck`). Entrypoints are per-command files `cmd_*.go`.
- `cmd/devlog-host` — Native Messaging host binary the browser extension talks to via stdin. Has OS-specific files `validate_host_unix.go` / `validate_host_windows.go` (build-tagged).
- `internal/config` — YAML config loading/validation.
- `internal/tmux` — tmux session management; holds the only integration tests (`-tags=integration`).
- `internal/natmsg` — native messaging wire protocol (length-prefixed JSON over stdin/stdout).
- `internal/manifest` — browser manifest registration for Chrome, Brave, and Firefox.
- `internal/browsersession` — browser-log capture lifecycle (wrapper scripts, clobber protection, health checks).
- `internal/logrotate` — log directory cleanup based on retention policy.
- `internal/logger`, `internal/shellescape` — support packages.
- `browser-extension/` — separate Node package (Chrome + Firefox). Own `npm ci` / `npm test`, not Go.

## Constraints that differ from defaults

- Go 1.25+; only external dependency is `gopkg.in/yaml.v3`. Prefer stdlib.
- Config is `devlog.yml` with `$VAR`/`${VAR}` env interpolation. Required fields: `version`, `project`, `tmux.session`, `windows`, `panes`, `cmd`.
- `devlog up` errors if a session is already running; `down` first.
- Integration tests need `tmux` available on PATH.
- Errors are wrapped with `%w`; messages start lowercase. Library code returns errors, does not log.
- Tests use table-driven style with `TestName_Description` and `t.Run()` subtests.
- Use `gofmt` (tabs); CI fails on unformatted code.
