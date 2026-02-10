# 5. Append-Only Log Writes

Date: 2026-02-10

## Status

Accepted

## Context

Multiple writers may produce log output concurrently (tmux panes, browser native host). We need a strategy that avoids data loss and keeps implementation simple.

Options considered:
- **Append-only writes**: Each writer opens its file in append mode. Simple, no coordination needed when files are distinct.
- **File locking (flock)**: Adds safety for shared files but increases complexity
- **Log aggregator (single writer)**: All sources feed into one process that writes. Clean but adds a bottleneck.
- **SQLite / structured store**: Queryable but overkill for MVP

## Decision

Use append-only writes. Each log source writes to its own dedicated file. No file locking is needed because each file has exactly one writer.

## Consequences

### Positive
- Simplest possible implementation — just `os.OpenFile` with `O_APPEND`
- No coordination or locking overhead
- No risk of file corruption from concurrent access (single writer per file)
- Easy to `tail -f` for real-time monitoring

### Negative
- Cannot merge server and browser logs into a single file without a separate aggregation step
- No structured querying (grep/ripgrep is the query tool)
