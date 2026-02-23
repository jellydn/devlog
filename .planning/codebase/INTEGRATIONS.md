# External Integrations

**Analysis Date:** 2026-02-23

## APIs & External Services
**tmux:**
- tmux CLI — manages terminal sessions, windows, and panes via `exec.Command("tmux", ...)`
- SDK/Client: Go `os/exec` (standard library)
- Auth: None (local system binary)
- Used in: `internal/tmux/tmux.go`

**Browser Native Messaging Protocol:**
- Chrome/Brave/Firefox Native Messaging — length-prefixed JSON over stdin/stdout
- SDK/Client: Custom implementation in `internal/natmsg/natmsg.go`
- Auth: Extension ID registered in native messaging manifest (`com.devlog.host.json`)
- Used in: `cmd/devlog-host/main.go`, `browser-extension/background.js`

**GitHub Releases API:**
- Used only in `install.sh` to fetch latest release version tag
- Endpoint: `https://api.github.com/repos/jellydn/devlog/releases/latest`
- Auth: None (public API)

## Data Storage
**Databases:**
- None

**File Storage:**
- Local filesystem only
- Log files written to configurable `logs_dir` (default `./logs/`) with timestamped or overwrite modes (`internal/config/config.go`)
- Browser logs written via `internal/logger/logger.go` to configured `browser.file` path
- Native messaging manifests stored in OS-specific directories (`internal/natmsg/manifest.go`):
  - Chrome: `~/Library/Application Support/Google/Chrome/NativeMessagingHosts/` (macOS)
  - Brave: `~/Library/Application Support/BraveSoftware/Brave-Browser/NativeMessagingHosts/` (macOS)
  - Firefox: `~/Library/Application Support/Mozilla/NativeMessagingHosts/` (macOS)
  - Zen: `~/Library/Application Support/zen/NativeMessagingHosts/` (macOS)

**Caching:**
- Wrapper scripts cached in user cache dir (`os.UserCacheDir()/devlog/wrappers/`) — see `cmd/devlog/main.go:656-667`

## Authentication & Identity
**Auth Provider:**
- None — fully local tool, no user authentication

## Monitoring & Observability
**Error Tracking:**
- None

**Logs:**
- Stderr for CLI error output (`fmt.Fprintf(os.Stderr, ...)`)
- No structured logging framework; devlog *produces* logs for the user's projects, not for itself

**Code Coverage:**
- Codecov integration in CI (`secrets.CODECOV_TOKEN`) — `.github/workflows/ci.yml:53-56`

## CI/CD & Deployment
**Hosting:**
- GitHub (source) + GitHub Releases (binary distribution)
- Chrome Web Store / Firefox Add-ons (browser extension packages in `dist/`)

**CI Pipeline:**
- GitHub Actions — `.github/workflows/ci.yml`
  - Runs on: `ubuntu-latest`, multi-platform matrix (`ubuntu-latest`, `macos-latest`, `windows-latest`)
  - Steps: `gofmt` check, `go vet`, `go test -race`, integration tests, build verification
- GitHub Actions — `.github/workflows/release.yml`
  - Triggered by: `v*` tags
  - Uses: `goreleaser/goreleaser-action@v7`

**Dependency Management:**
- Renovate Bot — `renovate.json` (auto-update dependencies)

## Environment Configuration
**Required env vars:**
- None required for normal operation
- `GITHUB_TOKEN` — required for GoReleaser in release CI (`.github/workflows/release.yml`)
- `CODECOV_TOKEN` — optional for coverage upload (`.github/workflows/ci.yml`)

**Secrets location:**
- GitHub Actions secrets (`secrets.GITHUB_TOKEN`, `secrets.CODECOV_TOKEN`)
- No `.env` files or local secret management

## Webhooks & Callbacks
**Incoming:**
- None — no HTTP server component

**Outgoing:**
- None — no outbound HTTP calls from the application itself

## Browser Extension Communication
**Protocol:**
- Native Messaging (stdio-based, length-prefixed JSON)
- Chrome extension → `chrome.runtime.connectNative("com.devlog.host")` → `devlog-host` binary
- Message flow: browser console → content script → background script → native host → log file

**Supported Browsers:**
- Chrome (Manifest V3) — `browser-extension/chrome/manifest.json`
- Brave (Chromium-based, shares Chrome manifest format)
- Firefox (Manifest V2) — `browser-extension/firefox/manifest.json`
- Zen Browser (Firefox-based, shares Firefox native messaging dirs)

---
*Integration audit: 2026-02-23*
