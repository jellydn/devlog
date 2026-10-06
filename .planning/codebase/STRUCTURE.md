# Codebase Structure

**Analysis Date:** 2026-10-01

## Directory Layout

Tracked layout of this repository. `browser-extension/chrome/` and `browser-extension/firefox/` symlink shared scripts and icons at the parent. `manifest.json` in each of those directories is a real file. Generated and local-only paths are noted after the tree. They are not source.

```
.
├── .github/workflows/
│   ├── ci.yml
│   ├── pages.yml
│   ├── release-extension.yml
│   └── release.yml
├── .goreleaser.yml
├── .planning/codebase/          # analysis notes (see Special Directories)
├── AGENTS.md
├── CLAUDE.md
├── LICENSE
├── PRIVACY.md
├── README.md
├── browser-extension/
│   ├── README.md
│   ├── VERSION                  # 1.0.0, read by package scripts
│   ├── background.js            # canonical background script
│   ├── content_script.js
│   ├── page_inject.js
│   ├── popup.html
│   ├── popup.js
│   ├── package.json             # private Vitest package
│   ├── package-lock.json
│   ├── vitest.config.js
│   ├── icons/
│   │   ├── icon.svg
│   │   ├── icon16.png
│   │   ├── icon32.png
│   │   ├── icon48.png
│   │   └── icon128.png
│   ├── chrome/                  # unpacked Chrome/Brave extension
│   │   ├── manifest.json        # Manifest V3
│   │   ├── background.js -> ../background.js
│   │   ├── content_script.js -> ../content_script.js
│   │   ├── page_inject.js -> ../page_inject.js
│   │   ├── popup.html -> ../popup.html
│   │   ├── popup.js -> ../popup.js
│   │   └── icons -> ../icons
│   ├── firefox/                 # unpacked Firefox extension
│   │   ├── manifest.json        # Manifest V2
│   │   └── (same symlinks as chrome/)
│   └── test/
│       ├── background.test.js
│       ├── content_script.test.js
│       ├── page_inject.test.js
│       └── mocks/chrome.js
├── cmd/
│   ├── devlog/                  # CLI binary
│   │   ├── main.go
│   │   ├── helpers.go
│   │   ├── browser_session_adapter.go
│   │   ├── cmd_attach.go
│   │   ├── cmd_down.go
│   │   ├── cmd_healthcheck.go
│   │   ├── cmd_init.go
│   │   ├── cmd_ls.go
│   │   ├── cmd_open.go
│   │   ├── cmd_register.go
│   │   ├── cmd_status.go
│   │   ├── cmd_up.go
│   │   ├── healthcheck_test.go
│   │   ├── init_test.go
│   │   └── status_test.go
│   └── devlog-host/             # native messaging host binary
│       ├── main.go
│       └── main_test.go
├── doc/
│   ├── adr/
│   │   ├── 0001-use-go-for-cli-and-native-host.md
│   │   ├── 0002-yaml-driven-configuration.md
│   │   ├── 0003-tmux-for-server-log-capture.md
│   │   ├── 0004-native-messaging-for-browser-logs.md
│   │   ├── 0005-append-only-log-writes.md
│   │   ├── 0006-timestamped-run-directories.md
│   │   └── 0007-package-extraction-refactor.md
│   ├── PUBLICATION_CHECKLIST.md
│   ├── SCREENSHOTS.md
│   └── STORE_SUBMISSION.md
├── internal/
│   ├── browsersession/
│   │   ├── session.go
│   │   ├── browsersession_test.go
│   │   └── helpers_test.go
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   ├── e2e/
│   │   └── cli_test.go          # //go:build e2e
│   ├── fileutil/
│   │   ├── touchfile.go
│   │   └── touchfile_test.go
│   ├── logger/
│   │   ├── logger.go
│   │   └── logger_test.go
│   ├── logrotate/
│   │   ├── logrotate.go
│   │   └── logrotate_test.go
│   ├── manifest/
│   │   ├── manifest.go
│   │   ├── manifest_test.go
│   │   ├── validate_host_unix.go
│   │   └── validate_host_windows.go
│   ├── natmsg/
│   │   ├── natmsg.go
│   │   └── natmsg_test.go
│   ├── shellescape/
│   │   ├── shellescape.go
│   │   └── shellescape_test.go
│   └── tmux/
│       ├── tmux.go
│       ├── tmux_test.go
│       └── integration_test.go  # //go:build integration
├── scripts/
│   ├── package-chrome.sh
│   ├── package-firefox.sh
│   ├── validate-screenshots.sh
│   └── ralph/
│       ├── prd.json
│       ├── progress.txt
│       ├── prompt-opencode.md
│       └── ralph.sh
├── devlog.yml.example
├── go.mod
├── go.sum
├── index.html
├── install.sh
├── justfile
├── logo.svg
└── renovate.json
```

Not in the source tree: `/devlog` and `/devlog-host` (local build outputs, listed in `.gitignore`), `dist/` (zip output of `scripts/package-chrome.sh` and `scripts/package-firefox.sh`), `browser-extension/node_modules/`, and `coverage.txt`. `just devlog-dev` writes the two binaries at the repo root and symlinks `~/.local/bin`.

## Directory Purposes

- `cmd/devlog/`: the `devlog` CLI. `main.go` dispatches. `cmd_*.go` holds one user command each. This package is `package main` and is not imported by `internal/`.
- `cmd/devlog-host/`: the `devlog-host` binary. Browser native messaging starts this process. It is not a subcommand of `devlog`.
- `internal/config/`: YAML load, `$VAR` / `${VAR}` interpolation, defaults, and `Validate`.
- `internal/tmux/`: tmux session create, kill, inspect, and version check. The only integration-test package (`-tags=integration`).
- `internal/browsersession/`: wrapper script lifecycle and browser health summary. Depends on callers for manifest and tmux operations.
- `internal/manifest/`: Chrome, Brave, Firefox, and Zen native-messaging JSON install, rewrite, and host-path checks. OS-specific files use build tags.
- `internal/natmsg/`: length-prefixed JSON on stdin/stdout.
- `internal/logger/`: append-only browser log file writer used only by `devlog-host`.
- `internal/logrotate/`: deletion of old timestamped run directories.
- `internal/fileutil/`: `TouchFile`, shared by the CLI and tmux.
- `internal/shellescape/`: POSIX single-quote helper used by tmux and the browser wrapper.
- `internal/e2e/`: builds `cmd/devlog` and drives a real tmux session. Build tag `e2e`. Not part of `go test ./...` unless the tag is set. `.github/workflows/ci.yml` runs it explicitly.
- `browser-extension/`: Chrome and Firefox extension sources and their Vitest package. Not part of the Go module.
- `browser-extension/chrome/` and `browser-extension/firefox/`: directories you load unpacked (`README.md`). Shared behavior is edited in the parent `*.js` files. Only `manifest.json` differs.
- `browser-extension/test/`: Vitest tests and a small `chrome` mock. `vitest.config.js` sets `environment: "jsdom"` and includes `test/**/*.test.js`.
- `doc/adr/`: accepted decisions (Go, YAML, tmux, native messaging, append-only files, timestamped runs, package split). `doc/PUBLICATION_CHECKLIST.md`, `doc/SCREENSHOTS.md`, and `doc/STORE_SUBMISSION.md` are store-publishing notes, not runtime code.
- `scripts/`: shell packaging for the extension zips and screenshot checks. `scripts/ralph/` holds an agent-loop PRD and prompt. It is not used by `devlog up`.
- `.github/workflows/`: `ci.yml` (Go fmt, vet, race tests, tmux integration, e2e, both binaries, extension `npm test`), `release.yml` (GoReleaser on `v*` tags), `release-extension.yml`, and `pages.yml` (deploys when `index.html` changes).
- `.planning/codebase/`: written architecture and structure notes for this repo. On 2026-10-01 the working tree contains the files written here. `git status` shows previously tracked `CONCERNS.md`, `CONVENTIONS.md`, `INTEGRATIONS.md`, `STACK.md`, and `TESTING.md` as deleted.

## Key File Locations

**Entry Points:**

- `cmd/devlog/main.go` — CLI `main`, usage text, and the `commands` map.
- `cmd/devlog/cmd_up.go` — `devlog up` (tmux create, retention, browser wrapper).
- `cmd/devlog/cmd_down.go` — `devlog down`.
- `cmd/devlog/cmd_register.go` — `devlog register`.
- `cmd/devlog/cmd_healthcheck.go` — `devlog healthcheck`.
- `cmd/devlog/cmd_init.go` — writes a new `devlog.yml`.
- `cmd/devlog/cmd_status.go`, `cmd/devlog/cmd_ls.go`, `cmd/devlog/cmd_open.go`, `cmd/devlog/cmd_attach.go` — inspect, list, open, and attach.
- `cmd/devlog-host/main.go` — native host `main` and `processMessages`.
- `browser-extension/background.js` — `connectNative("com.devlog.host")`.
- `browser-extension/content_script.js` — injects the page hook and forwards logs.
- `browser-extension/page_inject.js` — wraps `console.*`.
- `browser-extension/chrome/manifest.json` and `browser-extension/firefox/manifest.json` — browser entry manifests.
- `install.sh` — install a released binary.
- `justfile` — local build, test, lint, and package recipes.
- `index.html` — GitHub Pages site only.

**Configuration:**

- `devlog.yml.example` — sample `version`, `project`, `logs_dir`, `run_mode`, `max_runs`, `retention_days`, tmux windows, and browser block.
- `internal/config/config.go` — loader and validation. Runtime file name searched is `devlog.yml` (`cmd/devlog/helpers.go`).
- `go.mod` — module `github.com/jellydn/devlog`, Go 1.25.6, `gopkg.in/yaml.v3`.
- `.goreleaser.yml` — two binaries, one `tar.gz` archive, linux and darwin, amd64 and arm64.
- `browser-extension/chrome/manifest.json` — MV3 permissions `nativeMessaging`, `storage`, `activeTab`, and `host_permissions` `<all_urls>`.
- `browser-extension/firefox/manifest.json` — MV2, gecko id `devlog@devlog.local`.
- `browser-extension/VERSION` — version string stamped by `scripts/package-chrome.sh` and `scripts/package-firefox.sh`.
- `renovate.json` — dependency update bot config. Not read by the CLI.
- `.github/workflows/ci.yml` — CI command list (`gofmt`, `go vet`, `go test -race`, integration tag, e2e tag, both `go build`s, `npm test`).

**Core Logic:**

- `internal/tmux/tmux.go` — session create, `pipe-pane`, `DEVLOG_LOGS_DIR`, kill, and status parse.
- `internal/browsersession/session.go` — wrapper generate, clobber guard, start, stop, health.
- `internal/manifest/manifest.go` — manifest install and path rewrite. `FindDevlogHostBinary` looks next to the current executable, not on `PATH`.
- `internal/manifest/validate_host_unix.go` and `validate_host_windows.go` — host path checks.
- `internal/natmsg/natmsg.go` — native-endian length prefix and JSON `Message`.
- `internal/logger/logger.go` — level filter and append format.
- `internal/logrotate/logrotate.go` — retention deletion.
- `internal/shellescape/shellescape.go` and `internal/fileutil/touchfile.go` — quoting and file create.
- `cmd/devlog/browser_session_adapter.go` — the only production wiring of manifest functions into `browsersession.ManifestOps`.

**Testing:**

- Unit tests sit beside the code: `internal/*/*_test.go`, `cmd/devlog/healthcheck_test.go`, `init_test.go`, `status_test.go`, `cmd/devlog-host/main_test.go`.
- `internal/tmux/tmux_test.go` is the unit file. `internal/tmux/integration_test.go` needs `tmux` on `PATH` and `-tags=integration`. `go test -short` skips those tests (`testing.Short` inside the file).
- `internal/e2e/cli_test.go` builds `../../cmd/devlog` in `TestMain` and runs it against a real session. Tag `e2e`.
- `browser-extension/test/*.test.js` with `browser-extension/test/mocks/chrome.js`. Run with `npm test` in `browser-extension/` (`vitest run`). CI job `test-extension` in `.github/workflows/ci.yml` does `npm ci` then `npm test` on Node 24.
- `justfile` recipes: `test`, `test-one`, `test-integration`, `test-race`, `lint` (`gofmt` plus `go vet`), `ci`.

## Naming Conventions

**Files:**

- CLI commands are `cmd_<name>.go` under `cmd/devlog/` (`cmd_up.go`, `cmd_register.go`). The command function is `cmd` plus the exported-style suffix already used in the file: `cmdUp`, `cmdDown`, `cmdRegister`.
- Tests are `*_test.go` next to the package. Test functions follow `TestName_Description` (`TestLoad_ValidConfig` is the example in `justfile`; `TestResolveStatusLogsDir_PrefersRunningLogsDir` is in `cmd/devlog/status_test.go`).
- OS-specific Go files end in `_unix.go` or `_windows.go` and start with a `//go:build` line (`internal/manifest/validate_host_unix.go`, `validate_host_windows.go`).
- Build-tagged tests use the tag as the file's first build constraint (`integration`, `e2e`), not a file-name suffix.
- Extension scripts use snake names: `background.js`, `content_script.js`, `page_inject.js`, `popup.js`. Tests mirror that stem: `background.test.js`.
- ADRs are `NNNN-kebab-title.md` in `doc/adr/`.
- Go source uses tabs via `gofmt`. CI in `.github/workflows/ci.yml` fails when `gofmt -l .` is non-empty.

**Directories:**

- One Go package per directory. The package name matches the directory: `config`, `tmux`, `natmsg`, `browsersession`, `logrotate`, `fileutil`, `shellescape`, `manifest`, `logger`, `e2e`.
- Binaries are directories under `cmd/`: `cmd/devlog` and `cmd/devlog-host`. Both are `package main`.
- `internal/` is the library root. `cmd/` does not export a library. `browser-extension/` is a separate Node package (`"private": true` in `browser-extension/package.json`).
- Lowercase, single-word directory names for Go packages. The hyphenated names are the host binary (`devlog-host`), the extension folder (`browser-extension`), and workflow files.

## Where to Add New Code

**New Feature:**

- A new CLI verb goes in a new `cmd/devlog/cmd_<name>.go` file with a `cmd<Name>(cfg *config.Config, args []string) error` function, then a line in the `commands` map in `cmd/devlog/main.go`. If the verb must run without `devlog.yml`, add it to the `init` / `register` / `healthcheck` branch in `main` that calls `runCommandWithoutConfig`.
- Session or pane behavior goes in `internal/tmux/tmux.go` and is called from the command file. Keep window and pane struct types in `internal/config/config.go` so the dependency stays config → tmux, which is the direction noted on `SessionConfig` in `internal/tmux/tmux.go`.
- Browser-host lifecycle (wrapper, clobber check, manifest path swap) goes in `internal/browsersession/session.go`. Add methods to `ManifestOps` there and implement them on `manifestAdapter` in `cmd/devlog/browser_session_adapter.go`. Do not import `internal/manifest` from `browsersession`.
- Wire-protocol fields go on `natmsg.Message` in `internal/natmsg/natmsg.go`, then through `logger.Log`, then the object posted by `browser-extension/background.js`.
- Extension behavior is edited in the canonical files `browser-extension/background.js`, `content_script.js`, `page_inject.js`, and `popup.js`. Do not edit the symlinks under `chrome/` and `firefox/`. Change permissions or the background entry in the matching `manifest.json` only.
- A new YAML field goes on a struct in `internal/config/config.go`, with a check in `Validate` when it is required, plus a matching key in `devlog.yml.example` and in `generateTemplate` in `cmd/devlog/cmd_init.go`.

**New Component/Module:**

- Add `internal/<name>/<name>.go` with a package comment, matching the one-purpose packages from `doc/adr/0007-package-extraction-refactor.md` (`manifest`, `browsersession`, `logrotate`, `fileutil`). Put `*_test.go` in the same directory.
- Keep the new package free of imports from `cmd/`. If the CLI must bind it to another package, add a small adapter next to `cmd/devlog/browser_session_adapter.go` rather than an import cycle.
- A new browser target needs a manifest directory next to `browser-extension/chrome/` and `firefox/`, a directory helper in `internal/manifest/manifest.go` (see `GetBraveNativeMessagingDir` and `getFirefoxDirs`, which already include Zen), and a flag branch in `cmd/devlog/cmd_register.go`.
- Do not add the feature to `scripts/ralph/`. That tree is a task-runner scratch pad (`prd.json`, `ralph.sh`), not a library.

**Utilities:**

- Shell quoting belongs in `internal/shellescape/shellescape.go` (`Quote`). Call it before interpolating any path or command into `sh -lc` or the wrapper script.
- Create-if-missing file helpers belong in `internal/fileutil/touchfile.go` (`TouchFile`). `cmd/devlog/helpers.go` already wraps it as `ensureFileExists`.
- Retention rules belong in `internal/logrotate/logrotate.go`, not in `internal/config`.
- String or JSON frame helpers for the browser pipe belong in `internal/natmsg/natmsg.go`, not in the host `main` loop.

## Special Directories

- `.github/workflows/` — CI and release. `ci.yml` installs tmux before integration tests. `release.yml` runs GoReleaser. `pages.yml` publishes the repo root when `index.html` changes. `release-extension.yml` packages the browser extension.
- `doc/adr/` — decision records. They are not loaded at runtime. Where an ADR disagrees with code, the code wins (timestamp directory format in `internal/tmux/tmux.go` versus `doc/adr/0006-timestamped-run-directories.md`; two binaries in `.goreleaser.yml` versus `doc/adr/0001-use-go-for-cli-and-native-host.md`).
- `scripts/` — release packaging for the extension and screenshot validation. `package-chrome.sh` copies canonical JS into a temp dir and zips `dist/`. It does not change how `devlog up` runs.
- `scripts/ralph/` — PRD and prompt files for an external agent loop. Not referenced by `cmd/` or `internal/`.
- `browser-extension/chrome/` and `browser-extension/firefox/` — loadable extension roots. Shared files are symlinks. Packaging scripts copy the canonical parent files so the zip does not depend on symlink resolution (`scripts/package-firefox.sh`).
- `internal/e2e/` and the integration file `internal/tmux/integration_test.go` — tests that need a built binary or a live tmux. They are excluded from default `go test` by build tags.
- `dist/` — created at package time and gitignored. Not a source directory.
- `.planning/codebase/` — analysis documents. Empty of the older notes in this working tree except the files added for this analysis (`git status` lists those older notes as deleted).

---

*Structure analysis: 2026-10-01*
