# Architecture

**Analysis Date:** 2026-10-01

## Pattern Overview

**Overall:** Two local Go programs plus a browser extension. `devlog` (`cmd/devlog/main.go`) is a short-lived CLI that reads `devlog.yml` and drives tmux. `devlog-host` (`cmd/devlog-host/main.go`) is a long-lived native-messaging process the browser starts. It reads length-prefixed JSON on stdin and appends lines to one log file. There is no HTTP server, database, or shared in-process runtime between the two binaries. They meet only through files: log directories, a wrapper script, and the browser native-messaging manifest `com.devlog.host.json`.

**Key Characteristics:**

- Declarative session layout. `internal/config/config.go` loads one YAML file. `internal/tmux/tmux.go` turns windows and panes into `tmux` CLI calls.
- External process supervisor. tmux owns dev-server processes. Pane output is captured with `pipe-pane` and `cat >>`, not by wrapping the user command in `tee`.
- Separate browser path. `internal/browsersession/session.go` points the installed manifest at a per-session wrapper. The wrapper `exec`s `devlog-host` with the log path and level list.
- One-way library dependencies. `internal/tmux` imports `internal/config` for window and pane types (`SessionConfig` comment in `internal/tmux/tmux.go`). `internal/browsersession` does not import `internal/manifest` or `internal/config`. The CLI adapts those packages in `cmd/devlog/browser_session_adapter.go`. `internal/natmsg` and `internal/manifest` do not import each other (`doc/adr/0007-package-extraction-refactor.md`).
- Stdlib plus one module dependency. `go.mod` requires Go 1.25.6 and `gopkg.in/yaml.v3`.
- Two release binaries. `.goreleaser.yml` builds `./cmd/devlog` and `./cmd/devlog-host` for linux and darwin (amd64, arm64) into one archive. `doc/adr/0001-use-go-for-cli-and-native-host.md` says "single binary"; the tree ships two mains.

## Layers

### CLI command layer

**Purpose:** Parse argv, load config when required, and call one command function. Print human-readable status to stdout and errors to stderr. Exit non-zero on failure.

**Location:** `cmd/devlog/`

**Contains:** `main.go` (usage text, `commands` map, `main`, `runCommandWithoutConfig`). One file per command: `cmd_init.go`, `cmd_up.go`, `cmd_down.go`, `cmd_attach.go`, `cmd_status.go`, `cmd_ls.go`, `cmd_open.go`, `cmd_register.go`, `cmd_healthcheck.go`. `helpers.go` finds `devlog.yml` and touches files. `browser_session_adapter.go` adapts `internal/manifest` and `internal/tmux` to browser-session interfaces.

**Depends on:** `internal/config`, `internal/tmux`, `internal/browsersession`, `internal/logrotate`, `internal/manifest`, `internal/fileutil`.

**Used by:** The `devlog` process started by the user, `just run`, `go install ./cmd/devlog`, and `internal/e2e/cli_test.go` (builds this package).

### Native messaging host

**Purpose:** Stay attached to the browser on stdin/stdout. Filter console events by level and append formatted lines until stdin hits EOF.

**Location:** `cmd/devlog-host/main.go`

**Contains:** `main`, testable `run`, and `processMessages`. `messageLogger` is the small interface the loop needs from `internal/logger`.

**Depends on:** `internal/logger`, `internal/natmsg`.

**Used by:** The browser, after it launches the path stored in `com.devlog.host.json`. During `devlog up` that path is the wrapper from `internal/browsersession/session.go`, which `exec`s this binary. `internal/manifest/manifest.go` `FindDevlogHostBinary` looks for `devlog-host` beside the running `devlog` executable.

### Configuration

**Purpose:** Read `devlog.yml`, substitute `$VAR` and `${VAR}`, apply defaults, and reject incomplete session layouts.

**Location:** `internal/config/config.go`

**Contains:** `Config`, `TmuxConfig`, `WindowConfig`, `PaneConfig`, `BrowserConfig`, `Load`, `Validate`, `interpolateEnvVars`.

**Depends on:** `gopkg.in/yaml.v3` and the standard library. It does not import other `internal` packages. Retention cleanup lives in `internal/logrotate`, not here (`doc/adr/0007-package-extraction-refactor.md`).

**Used by:** `cmd/devlog/main.go` (`config.Load`). `internal/tmux/tmux.go` uses `config.WindowConfig` inside `SessionConfig`. Commands read fields. They do not reload the file.

### tmux session orchestration

**Purpose:** Create, inspect, and kill a tmux session whose panes run the configured commands and append output to log files.

**Location:** `internal/tmux/tmux.go`

**Contains:** `Runner`, `SessionConfig`, `CreateSession`, `KillSession`, `GetLogsDir`, `GetSessionInfo`, `CheckVersion`, and the unexported pane and window helpers. `SessionInfo`, `WindowInfo`, and `PaneInfo` are the read model for `devlog status`.

**Depends on:** `internal/config` (window and pane structs only), `internal/fileutil` (`TouchFile` for pane logs), `internal/shellescape` (quote log paths and pane commands). It shells out to the `tmux` binary.

**Used by:** `cmd/devlog/cmd_up.go`, `cmd_down.go`, `cmd_attach.go`, `cmd_status.go`, `cmd_open.go`, `cmd_healthcheck.go` (`CheckVersion`), and `tmuxSessionChecker` in `cmd/devlog/browser_session_adapter.go`.

### Browser-log capture lifecycle

**Purpose:** For one tmux session, install a native-host wrapper, refuse to clobber another live session's wrapper, and restore the real binary on stop. Also report host and manifest health.

**Location:** `internal/browsersession/session.go`

**Contains:** `Session`, `ManifestOps`, `SessionChecker`, `Start`, `Stop`, `HealthCheck`, wrapper path helpers, and shell or batch script generators.

**Depends on:** `internal/shellescape` for the Unix wrapper. Manifest and tmux behavior arrive only through the two interfaces. Callers in `cmd/devlog` pass `manifestAdapter` and `tmuxSessionChecker`.

**Used by:** `cmd/devlog/cmd_up.go` (`Start`), `cmd_down.go` (`Stop`), `cmd_healthcheck.go` (`HealthCheck`).

### Native messaging manifest registration

**Purpose:** Install and rewrite `com.devlog.host.json` for Chrome, Brave, Firefox, and Zen. Check that the host path exists and, on Unix, is owned by the current user.

**Location:** `internal/manifest/manifest.go`, `internal/manifest/validate_host_unix.go` (`//go:build !windows`), `internal/manifest/validate_host_windows.go` (`//go:build windows`).

**Contains:** `ChromeManifest`, `FirefoxManifest`, directory helpers (`GetChromeNativeMessagingDir`, `GetBraveNativeMessagingDir`, `GetFirefoxNativeMessagingDirs`), `InstallChromeManifest`, `InstallBraveManifest`, `InstallFirefoxManifestWithID`, `UpdateManifestPath`, `ReadManifestPaths`, `RepairStaleManifestPaths`, `FindDevlogHostBinary`, `ValidateHostPath`. Manifest name constant is `com.devlog.host.json`. Type is `stdio`.

**Depends on:** Standard library only.

**Used by:** `cmd/devlog/cmd_register.go` for install. `manifestAdapter` in `cmd/devlog/browser_session_adapter.go` for the browser-session lifecycle.

### Native messaging wire protocol

**Purpose:** Encode and decode the browser native-messaging frame: 4-byte native-endian length plus a JSON body. Cap body size at 10 MiB.

**Location:** `internal/natmsg/natmsg.go`

**Contains:** `Message`, `Timestamp` (RFC3339 string or Unix milliseconds), `Response`, `Host`, `ReadMessage`, `WriteResponse`, `SendAck`. `NewHost` uses stdin and stdout. `NewHostWithStreams` is the test seam used by `cmd/devlog-host/main.go`.

**Depends on:** Standard library only. It does not import `internal/manifest`.

**Used by:** `cmd/devlog-host/main.go` and `internal/logger/logger.go` (`Log` takes `*natmsg.Message`).

### Browser log writer

**Purpose:** Append one formatted console line per accepted message. Drop levels that are not in the allow-list. If the allow-list is empty, keep every level.

**Location:** `internal/logger/logger.go`

**Contains:** `Logger`, `New` (creates the directory and opens `O_APPEND`), `Log`, `ShouldLog`, `Close`. A `sync.Mutex` guards writes from this process. The line shape is `[TIMESTAMP] [LEVEL] [URL] source:line:column: message`.

**Depends on:** `internal/natmsg`.

**Used by:** `cmd/devlog-host/main.go` only.

### Log retention

**Purpose:** Delete old timestamped run directories by count (`max_runs`) and age (`retention_days`). Zero for both means do nothing. Missing logs directory is not an error.

**Location:** `internal/logrotate/logrotate.go`

**Contains:** `Policy`, `Result`, `Cleanup`. `dryRun` records paths without deleting them. `cmd/devlog/cmd_up.go` calls `Cleanup` with `dryRun` false.

**Depends on:** Standard library only.

**Used by:** `cmd/devlog/cmd_up.go`, and only when `cfg.RunMode == "timestamped"`.

### Filesystem and shell helpers

**Purpose:** Create log files without truncating them, and quote strings for a POSIX `sh -c` command line.

**Location:** `internal/fileutil/touchfile.go`, `internal/shellescape/shellescape.go`

**Contains:** `TouchFile` (`MkdirAll` mode `0755`, open `O_CREATE|O_APPEND|O_WRONLY` mode `0644`). `Quote` wraps in single quotes and escapes embedded quotes as `'\''`.

**Depends on:** Standard library only.

**Used by:** `TouchFile` is used by `cmd/devlog/helpers.go` (`ensureFileExists`) and `internal/tmux/tmux.go` (`ensurePaneLogFiles`). `Quote` is used by `internal/tmux/tmux.go` and `internal/browsersession/session.go`.

### Browser extension

**Purpose:** Capture `console.*`, `window` `error`, and `unhandledrejection` in the page, and forward matching events to the native host. No application code changes.

**Location:** Canonical scripts are `browser-extension/background.js`, `content_script.js`, `page_inject.js`, `popup.js`, and `popup.html`. `browser-extension/chrome/manifest.json` is Manifest V3 (service worker). `browser-extension/firefox/manifest.json` is Manifest V2 (`background.scripts`, gecko id `devlog@devlog.local`). JS, HTML, and `icons` under `chrome/` and `firefox/` are symlinks to those canonical files. Tests live in `browser-extension/test/` and run with Vitest (`browser-extension/package.json`, `browser-extension/vitest.config.js`).

**Contains:** Page hook (`page_inject.js`), content-script bridge, background native port (`NATIVE_HOST_NAME = "com.devlog.host"`), and a popup that only displays status.

**Depends on:** Browser extension APIs (`chrome.runtime.connectNative`, `sendMessage`, `nativeMessaging` permission). It does not import the Go module and does not read `devlog.yml`.

**Used by:** Chrome, Brave (loads `browser-extension/chrome`, per `README.md`), and Firefox (loads `browser-extension/firefox`).

## Data Flow

### Flow: `devlog up` creates a tmux session

1. `cmd/devlog/main.go` walks up at most 20 directories (`maxFindConfigDepth` in `cmd/devlog/helpers.go`) for `devlog.yml`, then calls `config.Load`.
2. `config.Load` interpolates environment variables, unmarshals YAML, defaults `logs_dir` to `./logs` and `run_mode` to `timestamped`, then `Validate` requires `version`, `project`, `tmux.session`, at least one window, and a non-empty `cmd` on every pane. `run_mode` must be `timestamped` or `overwrite`.
3. `cmdUp` in `cmd/devlog/cmd_up.go` builds `tmux.NewRunner(cfg.Tmux.Session)`. If `SessionExists` is true, it returns an error and tells the user to run `devlog down` first.
4. When `run_mode` is `timestamped`, `cmdUp` calls `logrotate.Cleanup` with `Policy{MaxRuns, RetentionDays}` before creating the new run. Cleanup failures are warnings on stderr. The new session is still created.
5. `Runner.CreateSession` in `internal/tmux/tmux.go` resolves the run directory. `timestamped` joins `logs_dir` with `time.Now().Format("20060102-150405")` (for example `logs/20261001-153045`). `overwrite` uses `logs_dir` as-is. `doc/adr/0006-timestamped-run-directories.md` describes a `YYYY-MM-DD_HH-MM-SS` name; the code does not use that layout.
6. `CreateSession` creates the directory (`0755`), touches each pane log via `fileutil.TouchFile`, runs `tmux new-session -d -s <session> -n <first-window>`, and stores the absolute logs path in the tmux session environment as `DEVLOG_LOGS_DIR`.
7. For each pane, `sendCommandWithLogging` runs `tmux pipe-pane -t <target> -o "cat >> <quoted log>"` when `log` is set, then `tmux send-keys` of `sh -lc <quoted cmd>` plus Enter. Extra panes use `tmux split-window -h`. Later windows use `tmux new-window`.
8. `cmdUp` prints `runner.GetLogsDir()`. The in-memory field set by `CreateSession` wins. If it is empty, `GetLogsDir` reads `DEVLOG_LOGS_DIR` from tmux.
9. If `browser.urls` is non-empty and `browser.file` is set, `cmdUp` touches that file under the run directory and calls `browsersession.Session.Start`. Failure here is a warning. The tmux session stays up.

### Flow: extension talks to `devlog-host` on stdin

1. `Session.Start` in `internal/browsersession/session.go` finds `devlog-host` beside the `devlog` binary (`FindDevlogHostBinary`), checks `ValidateHostPath`, and repairs manifests whose `path` is missing on disk.
2. It writes an executable wrapper under the user cache directory: `<UserCacheDir>/devlog/wrappers/devlog-host-wrapper-<session>.sh` (`.bat` on Windows). The Unix script is `#!/bin/sh` plus `exec` of the host, the absolute browser log path, and each configured level, all `shellescape.Quote`d.
3. `refuseClobberActiveWrapper` reads current manifest paths. If another `devlog-host-wrapper-<session>` file still exists and `SessionChecker.SessionExists` says that tmux session is alive, `Start` returns an error. The user must `devlog down` that session first.
4. `UpdateManifestPath` rewrites the `path` field of every installed `com.devlog.host.json` (Chrome, Brave, Firefox, and Zen dirs from `ManifestDirs`) to the wrapper.
5. In the page, `content_script.js` injects `page_inject.js` (a `web_accessible_resources` script so it runs in the page world). `page_inject.js` wraps `console.log`, `info`, `warn`, `error`, `debug`, and `trace`, and also listens for `error` and `unhandledrejection`. Each event is `window.postMessage` of an object with `__devlog: true`.
6. `content_script.js` ignores the message unless `isLoggingEnabled` and the level is in the content-script allow-list. It parses a source location from the stack and sends `{type: "LOG", ...}` with `chrome.runtime.sendMessage`.
7. `background.js` handles `LOG` in `chrome.runtime.onMessage`. `isUrlEnabled` matches the URL against the in-memory `config.urls` (default `http://localhost:*/*` and `http://127.0.0.1:*/*`). Those defaults are not loaded from `devlog.yml`. Nothing in the Go tree sends `UPDATE_CONFIG`. `cmd_status.go` prints `browser.urls` from YAML, but that list does not change the extension filter. A non-empty YAML `urls` list only decides whether `cmdUp` installs the wrapper.
8. `sendToNativeHost` calls `chrome.runtime.connectNative("com.devlog.host")` on first use, then `nativePort.postMessage`. The browser starts the manifest `path` and speaks native messaging on that process's stdin and stdout.
9. `devlog-host` `run` lowercases the level arguments and builds `logger.New`. `processMessages` loops on `natmsg.Host.ReadMessage` (`binary.NativeEndian` uint32 length, then JSON). `logger.Log` applies the host level filter again and appends a line. `SendAck` writes a `{"success": true}` response. The extension's `onMessage` handler looks for `message.type === "ACK"`. The Go `Response` struct has `success` and `error` only (`internal/natmsg/natmsg.go`), so that ACK branch does not see `type`.
10. When the browser closes the port, `ReadMessage` returns `io.EOF` and `processMessages` returns nil. The host exits.

### Flow: `devlog down` stops the session and restores the host

1. `cmdDown` in `cmd/devlog/cmd_down.go` checks `SessionExists`. If the session is already gone, it still calls `browsersession.Session.Stop`, then returns an error.
2. `Runner.KillSession` lists pane ids, sends `C-c` to each, sleeps 500 ms in the Go process, sends `C-c` again, then `tmux kill-session`.
3. `Session.Stop` points manifests back at the real `devlog-host` when this session's wrapper path is in use. Otherwise it repairs missing paths. It then deletes the wrapper file. Errors inside `Stop` are ignored.

### Flow: `devlog register` installs manifests

1. `cmdRegister` does not load `devlog.yml` (`runCommandWithoutConfig` in `cmd/devlog/main.go`).
2. It resolves `devlog-host` next to the current executable.
3. Flags `--chrome`, `--brave`, and `--firefox` select browsers. With no browser flag, Firefox is always selected, and Chrome is added when `--extension-id` is present. Chrome and Brave require `--extension-id`.
4. Chrome and Brave manifests set `allowed_origins` to `chrome-extension://<id>/`. Firefox sets `allowed_extensions` to the given id, or `devlog@devlog.local` when the flag is omitted (`InstallFirefoxManifestWithID`). Files are mode `0600`.

### Flow: status, list, open, attach, init, healthcheck

1. `devlog status` (`cmd/devlog/cmd_status.go`) prints project, session, and run mode. If tmux is up, it reads `DEVLOG_LOGS_DIR` (or the newest subdirectory of `logs_dir` when that variable is missing and mode is `timestamped`) and lists windows, panes, and file sizes.
2. `devlog ls` (`cmd/devlog/cmd_ls.go`) lists subdirectories of `logs_dir` in timestamped mode, or a file count in overwrite mode. It does not call tmux.
3. `devlog open` (`cmd/devlog/cmd_open.go`) opens the live logs directory, or `logs_dir` if the session is down, with `open` (darwin), `cmd /c start` (windows), or `xdg-open`.
4. `devlog attach` (`cmd/devlog/cmd_attach.go`) checks the session, then replaces the process I/O with `tmux attach -t <session>`.
5. `devlog init` (`cmd/devlog/cmd_init.go`) writes `./devlog.yml` from `generateTemplate`. It detects a monorepo when `packages`, `apps`, or `services` exists. It does not copy `devlog.yml.example` (that copy is the `just init` recipe in `justfile`).
6. `devlog healthcheck` (`cmd/devlog/cmd_healthcheck.go`) prints tmux version (`tmux.CheckVersion`) and `Session.HealthCheck` (host binary, which browsers have a manifest, stale path repair). It does not load `devlog.yml`.

**State Management:**

No in-memory application store survives a CLI invocation. Each `devlog` process loads YAML once and exits. Durable state is outside the process:

- tmux server state: the session, panes, and `DEVLOG_LOGS_DIR` (`internal/tmux/tmux.go`).
- Log trees on disk under `logs_dir`. Timestamped runs are separate directories. Pane capture and the browser host append. They do not truncate (`O_APPEND` in `internal/logger/logger.go` and `internal/fileutil/touchfile.go`; `cat >>` in `sendCommandWithLogging`).
- Wrapper scripts under the user cache `devlog/wrappers` directory (`browserHostWrapperPath` in `internal/browsersession/session.go`).
- Browser native-messaging JSON under the OS-specific directories in `internal/manifest/manifest.go`. The `path` field is either the real `devlog-host` or the current session wrapper.
- Extension memory in `browser-extension/background.js`: `config` object and `nativePort`. Defaults are compiled into that file. `chrome.storage` is a permission in both manifests, but `background.js` does not read or write it. Popup state is a one-shot `GET_STATUS` query in `browser-extension/popup.js`.

`Runner.logsDir` exists only inside the process that called `CreateSession`. Later commands recover the path from tmux.

## Key Abstractions

- `Command` (`cmd/devlog/main.go`): `func(cfg *config.Config, args []string) error`. The `commands` map is the only dispatch table. `init`, `register`, and `healthcheck` are called with a nil config.
- `config.Config` (`internal/config/config.go`): the whole YAML document after defaults and validation. Nested `TmuxConfig`, `WindowConfig`, `PaneConfig` (`cmd`, `log`), and `BrowserConfig` (`urls`, `file`, `levels`).
- `tmux.Runner` and `tmux.SessionConfig` (`internal/tmux/tmux.go`): one session name plus the resolved logs directory. `SessionConfig` is the input to `CreateSession`. `SessionInfo` is the output of `GetSessionInfo`.
- `browsersession.Session`, `ManifestOps`, `SessionChecker` (`internal/browsersession/session.go`): the only interface-based dependency injection in the Go tree. `manifestAdapter` and `tmuxSessionChecker` (`cmd/devlog/browser_session_adapter.go`) are the production implementations.
- `natmsg.Host` and `natmsg.Message` (`internal/natmsg/natmsg.go`): the stdio frame and the log event. `Timestamp` accepts a string or a number. `Message.UnmarshalJSON` accepts `line` and `column` as numbers or numeric strings.
- `logger.Logger` (`internal/logger/logger.go`): one open append-only file plus a level set.
- `manifest.ChromeManifest` and `manifest.FirefoxManifest` (`internal/manifest/manifest.go`): the JSON written for Chromium-family browsers versus Firefox and Zen. Both use host name `com.devlog.host` and `"type": "stdio"`.
- `logrotate.Policy` (`internal/logrotate/logrotate.go`): `MaxRuns` and `RetentionDays`. A directory is removed when it falls outside the newest N or its mod time is older than the cutoff. Both rules can apply together.

## Entry Points

- `cmd/devlog/main.go` `main`: user CLI. Commands are `init`, `up`, `down`, `attach`, `status`, `ls`, `open`, `register`, `healthcheck`, and `help`.
- `cmd/devlog-host/main.go` `main`: browser-spawned host. Required argv is the log file path. Further args are level names.
- `browser-extension/chrome/manifest.json`: Chrome and Brave unpacked extension. Background entry is the service worker `background.js`. Content script runs at `document_start` on `<all_urls>`, all frames.
- `browser-extension/firefox/manifest.json`: Firefox temporary add-on. Background entry is persistent `background.js`. Gecko id `devlog@devlog.local`.
- `browser-extension/page_inject.js`: page-world hook injected by `content_script.js`, not listed as its own content script.
- `install.sh`: downloads a release into `/usr/local/bin` when that directory is writable, otherwise `~/.local/bin`.
- `justfile`: `devlog-dev` builds both binaries and symlinks them into `~/.local/bin`. `run` is `go run ./cmd/devlog`.
- `.github/workflows/release.yml`: on tags `v*`, GoReleaser (`release.yml` calls goreleaser) publishes the archive described in `.goreleaser.yml`.
- `index.html` with `.github/workflows/pages.yml`: static project site. It is not on the log-capture path.

## Error Handling

**Strategy:** Library packages return errors and do not log (`AGENTS.md`, and the packages under `internal/` follow that). The CLI formats the returned error in `cmd/devlog/main.go` as `Error: %v` on stderr and `os.Exit(1)`. Wrapped errors use `%w`. Messages produced in `internal/` start with a lowercase phrase (`failed to ...`, `config: ...`). The host is the exception that prints and continues: a bad frame or a write error goes to stderr, an ack with `success: false` is sent, and the read loop stays up (`processMessages` in `cmd/devlog-host/main.go`). EOF is a clean success.

**Patterns:**

- Config and tmux failures abort the command. `cmdUp` does not start a session that already exists (`cmd/devlog/cmd_up.go` and `Runner.CreateSession`).
- Browser setup and retention cleanup are best-effort on the `up` path. `cmdUp` prints `Warning:` to stderr for `logrotate.Cleanup`, browser log touch, and `Session.Start`, then still returns nil if tmux creation succeeded.
- `Session.Stop` and several manifest repair paths drop errors (`internal/browsersession/session.go`). `cmdDown` still reports tmux kill failures.
- `ValidateHostPath` refuses a missing path, a directory, and on Unix a file whose uid is not the current user (`internal/manifest/validate_host_unix.go`). Windows only checks that the path exists and is not a directory.
- `ReadMessage` rejects length 0 and bodies over 10 MiB (`internal/natmsg/natmsg.go`).
- `healthcheck` returns `fmt.Errorf("healthcheck failed")` after printing which checks failed (`cmd/devlog/cmd_healthcheck.go`). It does not stop on the first failure.
- Tests are table-driven `TestName_Description` with `t.Run` where the package uses subtests (`AGENTS.md`). Integration tests are behind `//go:build integration` in `internal/tmux/integration_test.go` and skip when `-short` is set or `tmux` is missing. End-to-end tests are behind `//go:build e2e` in `internal/e2e/cli_test.go`.

## Cross-Cutting Concerns

**Logging:** The programs do not use a logging framework. Operator output is `fmt` to stdout and stderr from `cmd/devlog` and `cmd/devlog-host`. Captured product logs are a different channel. tmux panes append through `pipe-pane` (`internal/tmux/tmux.go`). The browser host appends through `logger.Logger` (`internal/logger/logger.go`). `doc/adr/0005-append-only-log-writes.md` assumes one writer per file. The code matches that when each pane `log` and `browser.file` are distinct paths. The logger mutex only serializes writes inside one host process. There is no `flock`. `logger.Log` does not call `Sync`. `devlog down` relies on `kill-session` to close pane pipes.

**Validation:** `config.Config.Validate` checks required fields, `run_mode`, and non-negative `max_runs` and `retention_days` (`internal/config/config.go`). Unknown YAML keys are ignored by the unmarshal. Env placeholders that are unset stay as the original `$VAR` text (`interpolateEnvVars`). `ValidateHostPath` checks the binary the manifest will point at. `natmsg` checks frame length and JSON shape. The extension checks URL wildcards and levels in `browser-extension/background.js` and `content_script.js`. The host checks levels again. YAML `browser.levels` are the host filter, passed as wrapper arguments. They are not pushed into the extension's `config.levels`.

**Authentication:** None. No user accounts, tokens, or network listeners exist in `cmd/` or `internal/`. Local trust boundaries are the browser's native-messaging allow list (`allowed_origins` or `allowed_extensions` in `internal/manifest/manifest.go`) and the Unix owner check in `internal/manifest/validate_host_unix.go`. The extension has `nativeMessaging` and host access to `<all_urls>` (`browser-extension/chrome/manifest.json`, `browser-extension/firefox/manifest.json`). `page_inject.js` posts to `*` and the content script accepts only `event.source === window` messages with `__devlog: true`.

---

*Architecture analysis: 2026-10-01*
