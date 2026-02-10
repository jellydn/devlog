# 1. Use Go for CLI and Native Host

Date: 2026-02-10

## Status

Accepted

## Context

devlog needs a CLI tool and a native messaging host binary. The CLI manages tmux sessions and log capture. The native host receives browser console logs via Chrome's Native Messaging protocol (stdin/stdout).

Options considered:
- **Go**: Single binary, fast startup, good stdlib for file I/O and process management
- **Node.js (TypeScript)**: Familiar to web devs, but requires runtime installation
- **Python**: Easy scripting, but slow startup and packaging complexity
- **Shell + Node**: Shell for CLI, Node for native host — two languages, harder to maintain

## Decision

Use Go for both the CLI and the native messaging host. Ship as a single binary.

## Consequences

### Positive
- Single binary distribution — no runtime dependencies
- Fast startup time (critical for native messaging host, which is launched per-connection)
- Strong stdlib for file I/O, process exec, and OS-level operations
- Cross-platform compilation (macOS + Linux)

### Negative
- Less familiar to frontend-heavy teams compared to Node.js
- YAML parsing requires an external dependency (`gopkg.in/yaml.v3`)
