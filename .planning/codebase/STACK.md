# Technology Stack

**Analysis Date:** 2026-10-01

## Languages

**Primary:** Go. Module `github.com/jellydn/devlog` declares `go 1.25.6` in `go.mod`. The CLI is `cmd/devlog` and the native messaging host is `cmd/devlog-host`. ADR `doc/adr/0001-use-go-for-cli-and-native-host.md` records the choice of Go for both binaries.

**Secondary:** JavaScript (no TypeScript) for the browser extension under `browser-extension/`. Chrome uses Manifest V3 (`browser-extension/chrome/manifest.json`, `"manifest_version": 3`). Firefox uses Manifest V2 (`browser-extension/firefox/manifest.json`, `"manifest_version": 2`, `strict_min_version` `109.0`). Shared scripts are `background.js`, `content_script.js`, `page_inject.js`, and `popup.js`. The project site `index.html` is static HTML and CSS. Shell: `justfile`, `install.sh`, `scripts/package-chrome.sh`, `scripts/package-firefox.sh`.

## Runtime

**Environment:** Two runtimes. The Go binaries are compiled programs (no Go runtime is required on the end-user machine). Local operation requires `tmux` on `PATH`; `internal/tmux/tmux.go` calls `tmux` (`has-session`, `new-session`, `pipe-pane`, `send-keys`, `-V`). Pane commands run through `sh -lc` in `internal/tmux/tmux.go`. Browser console capture needs Chrome, Brave, or Firefox (Zen receives the same Firefox host manifest; `internal/manifest/manifest.go`). Extension tests run on Node.js 24 (`node-version: "24"` in `.github/workflows/ci.yml` and `.github/workflows/release-extension.yml`). `browser-extension/package.json` has no `engines` field.

**Package Manager:** Go modules. The only direct requirement is `gopkg.in/yaml.v3 v3.0.1` (`go.mod`). `go.sum` also lists transitive `gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405`. The extension is a separate npm package, `devlog-browser-extension` version `1.0.0`, lockfileVersion 3 (`browser-extension/package-lock.json`). CI installs it with `npm ci`. Task runner is `just` (`justfile`, comment points at https://github.com/casey/just). No `packageManager` field and no pinned `just` version. No Dockerfile and no Makefile (absent from the repository root).

## Frameworks

**Core:** No web framework and no HTTP server. CLI dispatch is a command map in `cmd/devlog/main.go` (`init`, `up`, `down`, `attach`, `status`, `ls`, `open`, `register`, `healthcheck`). Config is YAML via `gopkg.in/yaml.v3` (`internal/config/config.go`). Browser I/O is the Chrome Native Messaging protocol: 4-byte length prefix plus JSON on stdin/stdout (`internal/natmsg/natmsg.go`, `cmd/devlog-host/main.go`). The extension connects with `chrome.runtime.connectNative("com.devlog.host")` (`browser-extension/background.js`).

**Testing:** Go standard `testing` package. `justfile` recipes: `go test ./...`, `go test -race ./...`, `go test -tags=integration ./internal/tmux/`, `go test -cover ./...`. CI also runs `go test -tags=e2e -v ./internal/e2e/` (`.github/workflows/ci.yml`). Extension tests use Vitest with the jsdom environment (`browser-extension/vitest.config.js`, script `"test": "vitest run"` in `browser-extension/package.json`).

**Build/Dev:** `go build` for `./cmd/devlog` and `./cmd/devlog-host` (`justfile` recipes `build` and `devlog-dev`). `gofmt` and `go vet` (`justfile` recipe `lint`; formatting gate in `.github/workflows/ci.yml`). Release builds use GoReleaser config version 2 (`.goreleaser.yml`) via `goreleaser/goreleaser-action@v7` with `version: "~> v2"` (`.github/workflows/release.yml`). Extension zips are built by `scripts/package-chrome.sh` and `scripts/package-firefox.sh`. Version string for the extension is `1.0.0` in `browser-extension/VERSION`.

## Key Dependencies

**Critical:**

- `gopkg.in/yaml.v3 v3.0.1` — only direct Go module (`go.mod`). Parses `devlog.yml`.
- `tmux` — external binary, not a Go module. Required for session and log capture (`internal/tmux/tmux.go`, `cmd/devlog/cmd_healthcheck.go`).
- `vitest` locked at `4.1.10` (`browser-extension/package-lock.json`; range `^4.0.0` in `browser-extension/package.json`). Dev dependency only.
- `jsdom` locked at `29.1.1` (`browser-extension/package-lock.json`; range `^29.0.0` in `browser-extension/package.json`). Dev dependency only. Transitive `node_modules/@asamuzakjp/dom-selector` records engines `^20.19.0 || ^22.12.0 || >=24.0.0`.

**Infrastructure:**

- GitHub Actions runners `ubuntu-latest`, `macos-latest`, and `windows-latest` (`.github/workflows/ci.yml`).
- GitHub Releases and the GitHub Releases API (`install.sh` downloads `devlog_${OS}_${ARCH}.tar.gz`).
- GitHub Pages for `index.html` (`.github/workflows/pages.yml`).
- Codecov upload (`codecov/codecov-action@v7` in `.github/workflows/ci.yml`).
- Chrome Web Store upload CLI and Firefox `web-ext` invoked with unpinned `npx` (`.github/workflows/release-extension.yml`).
- Renovate `config:recommended` (`renovate.json`). No application server, database, or container image.

## Configuration

**Environment:** User config is `devlog.yml` (example `devlog.yml.example`). `internal/config/config.go` interpolates `$VAR` and `${VAR}` with `os.Getenv` and leaves the token unchanged when the variable is empty. Required fields after defaults: `version`, `project`, `tmux.session`, at least one window, and each pane `cmd`. Defaults: `logs_dir` `./logs`, `run_mode` `timestamped`. Allowed `run_mode` values: `timestamped` or `overwrite`. Optional `max_runs` and `retention_days`. There is no dotenv file and no fixed application env-var schema. `install.sh` reads `VERSION` and `INSTALL_DIR`. CI secrets are listed in `INTEGRATIONS.md`.

**Build:** `go.mod` (`go 1.25.6`). `.goreleaser.yml` (`version: 2`) builds both binaries with `-s -w` for `linux` and `darwin`, `amd64` and `arm64`, archive format `tar.gz`, checksum file `checksums.txt`. `justfile` is the local task entry (`just ci` runs `lint` then `test`). Extension packaging reads `browser-extension/VERSION` and rewrites `"version"` in both manifests (`scripts/package-chrome.sh`).

## Platform Requirements

**Development:** Go 1.25.6 as declared in `go.mod` (CI uses `actions/setup-go@v7` with `go-version-file: go.mod`). `tmux` installed (`sudo apt-get install -y tmux` on Linux, `brew install tmux` on macOS in `.github/workflows/ci.yml`). Integration tests skip on Windows (`if: runner.os != 'Windows'`). Node.js 24 and npm for `browser-extension` tests. `just` for the recipes in `justfile`. `gofmt` must be clean. README states `Go 1.25+` for `go install` (`README.md`).

**Production:** Released CLI and host binaries are linux and darwin, amd64 and arm64 only (`.goreleaser.yml`). `install.sh` rejects other `uname` OS and architecture values. Windows is compiled in CI (`go build` on `windows-latest`) but is not a GoReleaser target. End users need `tmux` for server logs and a registered native host for browser logs (`cmd/devlog/cmd_healthcheck.go`). No minimum tmux version is checked; `CheckVersion` only runs `tmux -V`. Firefox extension declares `strict_min_version` `109.0` (`browser-extension/firefox/manifest.json`). Logs are local files under `logs_dir`. No hosted application process.

---

*Stack analysis: 2026-10-01*
