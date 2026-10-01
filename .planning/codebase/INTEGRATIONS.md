# External Integrations

**Analysis Date:** 2026-10-01

## APIs & External Services

The CLI and host do not call third-party product APIs. No `net/http` import exists under the repository Go or extension sources.

Local process integrations:

- **tmux.** `internal/tmux/tmux.go` execs the `tmux` binary for sessions, windows, panes, `pipe-pane` (`cat >> <log>`), and `DEVLOG_LOGS_DIR`. `cmd/devlog/cmd_attach.go` runs `tmux attach`. Health check links to `https://github.com/tmux/tmux/wiki/Installing` (`cmd/devlog/cmd_healthcheck.go`).
- **POSIX `sh`.** Pane commands are sent as `sh -lc` (`internal/tmux/tmux.go`).
- **Browser native messaging.** Host name `com.devlog.host`, type `stdio` (`internal/manifest/manifest.go`). Extension connects with `chrome.runtime.connectNative` (`browser-extension/background.js`). Protocol code is `internal/natmsg/natmsg.go`. Browsers with manifest writers: Google Chrome, Brave, Firefox, and Zen (Zen uses the Firefox manifest directory list in `internal/manifest/manifest.go`). `cmd/devlog/cmd_register.go` flags are `--chrome`, `--brave`, and `--firefox` only. `README.md` mentions Edge in an install comment; there is no `--edge` flag.
- **OS file managers.** `devlog open` starts `open` (darwin), `cmd /c start` (windows), or `xdg-open` (other) in `cmd/devlog/cmd_open.go`.

Release and site integrations:

- **GitHub Releases API and asset host.** `install.sh` GETs `https://api.github.com/repos/jellydn/devlog/releases/latest` and downloads `https://github.com/jellydn/devlog/releases/download/${VERSION}/devlog_${OS}_${ARCH}.tar.gz`.
- **GitHub Pages.** `.github/workflows/pages.yml` deploys the repo (including `index.html`) to the `github-pages` environment. README project URL is `https://jellydn.github.io/devlog/`.
- **Google Fonts (site only).** `index.html` loads `https://fonts.googleapis.com` and `https://fonts.gstatic.com` (Inter and JetBrains Mono). The CLI does not.
- **Chrome Web Store.** `.github/workflows/release-extension.yml` runs `npx chrome-webstore-upload-cli upload` (version not pinned). Manual path is documented in `doc/STORE_SUBMISSION.md`.
- **Firefox Add-ons (AMO).** Same workflow runs `npx web-ext sign --source-dir browser-extension/firefox --channel listed` (version not pinned).
- **Codecov.** `codecov/codecov-action@v7` in `.github/workflows/ci.yml`. Badge in `README.md` points at `https://codecov.io/gh/jellydn/devlog`.
- **Go Report Card.** Badge only in `README.md` (`https://goreportcard.com/report/github.com/jellydn/devlog`). No workflow step calls it.
- **Renovate.** `renovate.json` extends `config:recommended`. No runtime call.

## Data Storage

**Databases:** None. `go.mod` has no database driver. A search of `*.go`, `*.js`, `*.yml`, and `*.yaml` finds no `database/sql`, Redis, or other datastore client.

**File Storage:** Local disk only. Server pane output is appended by tmux `pipe-pane` to files under `logs_dir` (`internal/tmux/tmux.go`). Browser logs are append-only files opened with `O_APPEND` in `internal/logger/logger.go`. Run directories are timestamped or overwritten (`internal/config/config.go`, `internal/logrotate/logrotate.go`). Native host manifests are JSON files `com.devlog.host.json` under the browser NativeMessagingHosts directories (`internal/manifest/manifest.go`), mode `0600`. Extension packages are zip files under `dist/` (`scripts/package-chrome.sh`). GitHub Actions stores those zips with `actions/upload-artifact@v4` (`.github/workflows/release-extension.yml`).

**Caching:** None as a cache service. No Redis or memcached client (same search as Databases). The process writes short-lived host wrapper scripts under `os.UserCacheDir()/devlog/wrappers/` (`internal/browsersession/session.go`), falling back to `os.TempDir()`. That directory is a local helper path, not a cache product. The extension manifest declares the `storage` permission (`browser-extension/chrome/manifest.json`, `browser-extension/firefox/manifest.json`), but extension source has no `chrome.storage` call (only the test mock `browser-extension/test/mocks/chrome.js`).

## Authentication & Identity

None for end users. No login, OAuth client, session cookie, or identity provider in `cmd/` or `internal/`.

Local trust boundary: each `com.devlog.host.json` allowlists the extension. Chrome and Brave use `allowed_origins` `chrome-extension://<id>/`. Firefox and Zen use `allowed_extensions`. Default Firefox id is `devlog@devlog.local` (`internal/manifest/manifest.go`, `browser-extension/firefox/manifest.json`). `ValidateHostPath` checks that the host binary exists and, on Unix, is owned by the current user (`internal/manifest/validate_host_unix.go`). Store publish credentials are CI secrets only (see Environment Configuration). `PRIVACY.md` describes this allowlist; it is not a network identity system.

## Monitoring & Observability

**Error Tracking:** None. No Sentry, OpenTelemetry, or Prometheus client in Go, JavaScript, or workflows (search of those trees found no matches). Host read and write failures go to stderr (`cmd/devlog-host/main.go`). `devlog healthcheck` prints local checks and returns an error when a check fails (`cmd/devlog/cmd_healthcheck.go`). Codecov receives coverage profiles, not application errors.

**Logs:** Local files, not a log vendor. Format in `internal/logger/logger.go` is `[TIMESTAMP] [LEVEL] [URL] message`. `devlog up` can delete old timestamped run directories via `max_runs` and `retention_days` (`internal/logrotate/logrotate.go`). CI logs are GitHub Actions job logs. The workflow sets `continue-on-error: true` on the Codecov step (`.github/workflows/ci.yml`).

## CI/CD & Deployment

**Hosting:** No application host. Binaries are GitHub Release assets produced by GoReleaser (`.goreleaser.yml`, `.github/workflows/release.yml` on tags `v*`). The static site is GitHub Pages (`.github/workflows/pages.yml`, paths filter `index.html`, plus `workflow_dispatch`). Extension zips are GitHub Release assets (`softprops/action-gh-release@v2` in `.github/workflows/release-extension.yml`, tags `ext-v*` or `workflow_dispatch`). Optional store publish jobs use `continue-on-error: true`. Users can also `go install github.com/jellydn/devlog/cmd/devlog@latest` and `go install github.com/jellydn/devlog/cmd/devlog-host@latest` (`README.md`).

**CI Pipeline:** `.github/workflows/ci.yml` on push to `main` and `develop`, and on pull requests.

- `actions/checkout@v7`, `actions/setup-go@v7` (`go-version-file: go.mod`), `go mod download`.
- `gofmt` check, `go vet ./...`, `go test -race -coverprofile=coverage.txt -covermode=atomic ./...`, `go test -tags=integration ./internal/tmux/`, `go test -tags=e2e -v ./internal/e2e/`, `go build` of both binaries.
- Extension job: `actions/setup-node@v7` with Node `24`, `npm ci`, `npm test`.
- Matrix build on `ubuntu-latest`, `macos-latest`, `windows-latest`. Integration tests are skipped on Windows.
- Coverage upload: `codecov/codecov-action@v7` with `secrets.CODECOV_TOKEN`.

Release workflow permissions are `contents: write`. Pages workflow permissions are `contents: read`, `pages: write`, `id-token: write`. CI test job permissions are `contents: read`.

## Environment Configuration

**Required env vars:** The running CLI has no required environment variable. `devlog.yml` may reference any `$VAR` or `${VAR}`; unset names stay literal (`internal/config/config.go`). `devlog.yml.example` shows `$PORT` inside a pane command. The tmux session receives `DEVLOG_LOGS_DIR` (`internal/tmux/tmux.go`). Path lookup uses `HOME` (via `os.UserHomeDir`), `XDG_CONFIG_HOME`, and on Windows `APPDATA` (`internal/manifest/manifest.go`). Wrapper scripts use the user cache dir, which honors `XDG_CACHE_HOME` (`internal/browsersession/session.go`). `install.sh` optional variables: `VERSION`, `INSTALL_DIR` (default `~/.local/bin`, or `/usr/local/bin` when that directory is writable).

**Secrets location:** GitHub Actions secrets and one repository variable. Not stored in the repo.

- `secrets.CODECOV_TOKEN` — `.github/workflows/ci.yml`
- `secrets.GITHUB_TOKEN` — `.github/workflows/release.yml`, `.github/workflows/release-extension.yml`
- `secrets.CHROME_CLIENT_ID`, `secrets.CHROME_CLIENT_SECRET`, `secrets.CHROME_REFRESH_TOKEN` — mapped to `CLIENT_ID`, `CLIENT_SECRET`, `REFRESH_TOKEN` for `chrome-webstore-upload-cli`
- `vars.CHROME_EXTENSION_ID` — mapped to `EXTENSION_ID` (repository variable, not a secret)
- `secrets.FIREFOX_JWT_ISSUER`, `secrets.FIREFOX_JWT_SECRET` — mapped to `WEB_EXT_API_KEY` and `WEB_EXT_API_SECRET`

No `.env` file is present. Native messaging manifest files on the user machine are mode `0600` (`internal/manifest/manifest.go`).

## Webhooks & Callbacks

**Incoming:** None. The repository has no HTTP server and no `webhook` string in Go, JavaScript, YAML, Markdown, or shell. GitHub Actions triggers are `push`, `pull_request`, tag push, and `workflow_dispatch`, not an application endpoint.

**Outgoing:** No webhook sender in product code. Outbound calls that do exist:

- `install.sh` — HTTPS GET to the GitHub Releases API and to a release asset URL (via `curl`).
- `codecov/codecov-action@v7` — uploads `coverage.txt` (`.github/workflows/ci.yml`).
- `goreleaser/goreleaser-action@v7` — publishes a GitHub Release using `GITHUB_TOKEN` (`.github/workflows/release.yml`).
- `npx chrome-webstore-upload-cli upload` — Chrome Web Store API (`.github/workflows/release-extension.yml`).
- `npx web-ext sign` — Mozilla Add-ons signing API (same workflow).
- `index.html` — browser requests to Google Fonts when the project site is opened.

Native messaging acks (`type` `ACK` in `browser-extension/background.js`) stay on the local stdio port. They are not network callbacks.

---

*Integration audit: 2026-10-01*
