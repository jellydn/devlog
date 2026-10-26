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
- `timestamped` (default): Creates `logs/YYYYMMDD-HHMMSS/` per run. A second start in the same second uses `logs/YYYYMMDD-HHMMSS-2/`
- `overwrite`: Writes directly to `logs/`. Existing pane logs are truncated when the run starts, then appended for the rest of the run

## Consequences

### Positive
- Timestamped mode preserves full history — great for debugging regressions
- Overwrite mode keeps things simple for developers who don't need history
- Directory-per-run makes it easy to share or archive a complete debug session
- User chooses the behavior that fits their workflow

### Negative
- Timestamped mode can accumulate disk usage. Optional `max_runs` and `retention_days` delete only directories named like a timestamped run. `max_log_bytes` caps the browser log. Server pane logs from `pipe-pane` are not size-capped
- Two modes mean slightly more code and config surface
