# Technology Stack

**Analysis Date:** 2026-02-23

## Languages
**Primary:**
- Go 1.25.6 - CLI binaries (`cmd/devlog/`, `cmd/devlog-host/`), all internal packages (`internal/`)

**Secondary:**
- JavaScript (ES6+) - Browser extension (`browser-extension/background.js`, `browser-extension/content_script.js`, `browser-extension/page_inject.js`, `browser-extension/popup.js`)
- HTML/CSS - Extension popup UI (`browser-extension/popup.html`)
- Shell (POSIX sh) - Install script (`install.sh`), packaging scripts (`scripts/package-chrome.sh`, `scripts/package-firefox.sh`)

## Runtime
**Environment:**
- Go 1.25.6 (specified in `go.mod`)
- Chrome/Firefox browser runtime for extension (Manifest V3 for Chrome, Manifest V2 for Firefox)

**Package Manager:**
- Go modules (`go.mod` / `go.sum`)
- Lockfile: present (`go.sum`)

## Frameworks
**Core:**
- None — pure Go standard library with minimal dependencies

**Testing:**
- Go built-in `testing` package — unit and integration tests (`*_test.go` files)
- Build tag `integration` for tmux integration tests (`internal/tmux/integration_test.go`)

**Build/Dev:**
- [just](https://github.com/casey/just) — task runner (`justfile`)
- [GoReleaser](https://goreleaser.com/) v2 — cross-platform binary releases (`.goreleaser.yml`)

## Key Dependencies
**Critical:**
- `gopkg.in/yaml.v3` v3.0.1 — YAML config parsing (`internal/config/config.go`)

**Infrastructure:**
- No other external Go dependencies — the project deliberately uses only the standard library

## Configuration
**Environment:**
- YAML config file `devlog.yml` (searched in CWD and parent dirs) — see `devlog.yml.example`
- Supports `$VAR` / `${VAR}` environment variable interpolation in config values (`internal/config/config.go:129-150`)
- `INSTALL_DIR` — override install location (`install.sh`)
- `APPDATA`, `XDG_CONFIG_HOME` — platform-specific native messaging host paths (`internal/natmsg/manifest.go`)

**Build:**
- `go.mod` — module definition and Go version
- `.goreleaser.yml` — release build matrix (linux/darwin × amd64/arm64)
- `justfile` — development task definitions

## Platform Requirements
**Development:**
- Go 1.25+
- tmux (required for session management)
- just (optional, for task runner convenience)
- Chrome or Firefox (for browser extension development)

**Production:**
- Linux (amd64/arm64) or macOS (amd64/arm64) — per `.goreleaser.yml`
- tmux installed and available in `$PATH`
- Browser with extension support (Chrome/Brave/Firefox/Zen) for browser log capture

---
*Stack analysis: 2026-02-23*
