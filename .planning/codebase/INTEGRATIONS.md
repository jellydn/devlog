# External Integrations

**Analysis Date:** 2026-02-10

## APIs & External Services

**Browser APIs:**
- Chrome/Firefox Native Messaging API - Direct communication between browser extension and native host
- Console API - Capturing browser console logs
- Storage API - Extension preferences management
- Extension APIs - Permission system and tab access

**SDK/Client:**
- Custom native messaging protocol implementation
- Browser-specific extension manifests

**Auth:**
- No external authentication required
- Extension ID verification for Chrome

## Data Storage

**Databases:**
- None - All data stored in local files
- Structured logs in plain text format

**File Storage:**
- Local file system for:
  - YAML configuration files
  - Log files (timestamped or named)
  - Tmux session data
  - Native messaging manifests

**Caching:**
- No external caching services
- In-memory state during runtime only

## Authentication & Identity

**Auth Provider:**
- Custom extension-based authentication
- Chrome: Extension ID verification
- Firefox: Signed extension validation

**Implementation:**
- Native messaging manifest whitelisting
- No external identity providers

## Monitoring & Observability

**Error Tracking:**
- No external error tracking services
- Local log file capture from both tmux sessions and browser console

**Logs:**
- Direct log capture from:
  - Tmux panes and windows
  - Browser console (via extension)
- Structured log rotation support
- Timestamp-based log naming

## CI/CD & Deployment

**Hosting:**
- GitHub repository (github.com/jellydn/devlog)
- Releases distributed via GitHub releases

**CI Pipeline:**
- Just build automation (justfile)
- Manual release process with goreleaser
- No external CI services configured

## Environment Configuration

**Required env vars:**
- Optional: XDG_CONFIG_HOME (Linux native messaging paths)
- Optional: APPDATA (Windows native messaging paths)
- Optional: HOME (macOS native messaging paths)

**Secrets location:**
- No secrets required
- Configuration files stored locally

## Webhooks & Callbacks

**Incoming:**
- None - Direct native messaging only

**Outgoing:**
- None - No external API calls

---

*Integration audit: 2026-02-10*
