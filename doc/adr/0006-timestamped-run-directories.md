# 6. Timestamped Run Directories

Date: 2026-02-10

## Status

Accepted

## Context

Developers run `devlog up` multiple times per day. We need a strategy for organizing log output across runs so that previous logs are not lost and each run is easy to find.

Options considered:
- **Timestamped subdirectories**: Each run creates `logs/<timestamp>/`. Previous runs are preserved.
- **Overwrite mode**: Always write to `logs/`. Simple but destructive.
- **Rotating logs**: Numbered suffixes (`.1`, `.2`). Familiar but harder to correlate across files.
- **Git-based**: Commit logs per run. Overkill.

## Decision

Support both modes via `run_mode` in the YAML config:
- `timestamped` (default): Creates `logs/YYYY-MM-DD_HH-MM-SS/` per run
- `overwrite`: Writes directly to `logs/`, replacing previous output

## Consequences

### Positive
- Timestamped mode preserves full history — great for debugging regressions
- Overwrite mode keeps things simple for developers who don't need history
- Directory-per-run makes it easy to share or archive a complete debug session
- User chooses the behavior that fits their workflow

### Negative
- Timestamped mode can accumulate disk usage over time (no automatic cleanup in MVP)
- Two modes mean slightly more code and config surface
