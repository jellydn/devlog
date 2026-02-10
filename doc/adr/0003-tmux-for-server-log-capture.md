# 3. Use tmux for Server Log Capture

Date: 2026-02-10

## Status

Accepted

## Context

devlog needs to start multiple dev server processes, capture their stdout/stderr, and write output to log files — all without modifying application code.

Options considered:
- **tmux pipe-pane**: Captures pane output natively, zero-code, well-supported
- **Process wrapper (e.g., `tee`)**: Requires wrapping each command, fragile with signals
- **Custom process manager**: Maximum control but reinvents the wheel
- **Docker log drivers**: Heavyweight, not suitable for local dev workflows

## Decision

Use tmux as the process manager. Create sessions/windows/panes via `tmux` CLI commands (`new-session`, `send-keys`, `pipe-pane`). Capture output using `pipe-pane` to write stdout/stderr to configured log files.

## Consequences

### Positive
- Zero-code capture — no wrappers or modifications to dev commands
- Developers already using tmux get a familiar experience
- `pipe-pane` captures exactly what appears in the terminal, including color codes if desired
- Built-in session management (attach, detach, kill)

### Negative
- Requires tmux to be installed
- Not usable by developers who don't use terminal multiplexers
- `pipe-pane` captures raw terminal output (may include ANSI escape codes)
