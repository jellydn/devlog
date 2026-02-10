# 2. YAML-Driven Configuration

Date: 2026-02-10

## Status

Accepted

## Context

devlog needs a way to declare tmux sessions, pane commands, log paths, and browser capture settings. The configuration must be human-readable, version-controllable, and require no imperative scripting.

Options considered:
- **YAML**: Widely used for dev tooling configs, readable, supports nested structures
- **TOML**: Good for flat configs, less natural for deeply nested structures like windows/panes
- **JSON**: Verbose, no comments, poor DX for hand-editing
- **Custom DSL**: Maximum flexibility but learning curve

## Decision

Use a single `devlog.yml` file as the sole configuration source. Support environment variable interpolation (`$VAR` and `${VAR}`) in string values.

## Consequences

### Positive
- Declarative — the config describes what, not how
- Human-readable and easily diffable in version control
- Env var interpolation enables per-environment flexibility without multiple config files
- Familiar format for developers (Docker Compose, GitHub Actions, etc.)

### Negative
- YAML has well-known gotchas (implicit type coercion, indentation sensitivity)
- Env var interpolation adds parsing complexity
