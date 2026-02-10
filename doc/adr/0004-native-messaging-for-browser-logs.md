# 4. Use Native Messaging for Browser Log Capture

Date: 2026-02-10

## Status

Accepted

## Context

devlog needs to capture browser console output (log, warn, error, uncaught exceptions) and write it to local files without requiring application code changes.

Options considered:
- **Native Messaging Host**: Browser extension sends logs to a local binary via stdin/stdout. Supported by Chrome and Firefox.
- **WebSocket server**: Extension connects to a local WS server. Requires running a server, firewall considerations.
- **DevTools Protocol (CDP)**: Powerful but Chrome-only, complex setup, fragile
- **Export/download**: Extension batches logs and downloads as a file. Not real-time.

## Decision

Use Chrome's Native Messaging protocol. A Go binary acts as the native host, receiving JSON log events over stdin (4-byte little-endian length-prefixed) and appending formatted lines to the configured log file.

## Consequences

### Positive
- Real-time streaming — logs written within 100ms of the event
- No network access required — communication is via OS-level stdio
- Supported by both Chrome and Firefox
- Native host is a simple append-only file writer with no business logic

### Negative
- Requires installing a browser extension (manual for MVP)
- Requires registering a native host manifest in an OS-specific location
- MVP captures only a single tab at a time
