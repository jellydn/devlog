# Codebase Concerns
**Analysis Date:** 2026-10-01

Search covered `TODO|FIXME|HACK|XXX` in Go, JS, YAML, and Markdown (only a hash false positive in `browser-extension/package-lock.json`). No in-source task markers. Findings below come from reading the host, tmux, config, manifest, log, and extension paths, plus two local checks: tmux 3.7c pane targeting, and the extension URL matcher.

Confirmed defect means the code does the wrong thing on a path that exists today. Risk means the code allows a bad outcome that this audit did not execute end to end.

## Tech Debt

### Accepted ADRs and the README disagree with the implementation
- **Problem:** `doc/adr/0006-timestamped-run-directories.md` says timestamped runs use `logs/YYYY-MM-DD_HH-MM-SS/` and that `overwrite` replaces previous output. `README.md` shows `logs/2026-02-10_17-23-11/`. The code creates `20060102-150405` (`internal/tmux/tmux.go`) and never truncates. `internal/e2e/cli_test.go` locks the 15-character `YYYYMMDD-HHMMSS` name. `internal/fileutil/touchfile.go` opens with `O_APPEND` and the touch test asserts an existing file is not truncated. `doc/adr/0004-native-messaging-for-browser-logs.md` says 4-byte little-endian frames and "a single tab". `internal/natmsg/natmsg.go` uses `binary.NativeEndian` (correct for Chrome's native-order rule on amd64/arm64, not what the ADR says). The extension injects every frame of every URL (`browser-extension/chrome/manifest.json`). `doc/adr/0005-append-only-log-writes.md` says each file has one writer. Two panes may share one log (`internal/tmux/integration_test.go` `TestTmuxIntegration_MultiplePanesSameLogFile`) and both use `cat >>`.
- **Files:** `doc/adr/0006-timestamped-run-directories.md`, `doc/adr/0004-native-messaging-for-browser-logs.md`, `doc/adr/0005-append-only-log-writes.md`, `README.md`, `internal/tmux/tmux.go`, `internal/fileutil/touchfile.go`, `internal/natmsg/natmsg.go`
- **Impact:** Operators and later edits follow the ADR, not the binary. Overwrite looks destructive in the ADR and is not.
- **Fix approach:** Update the ADRs and README to the code, or change the code. Do not leave both.
- **Priority:** Medium

### Local `just ci` is a weaker gate than GitHub CI
- **Problem:** `just ci` is `lint` then `go test ./...`. That omits `-race`, `-tags=integration`, and `-tags=e2e`. `just build` and `just install` build only `cmd/devlog`, not `cmd/devlog-host`. GitHub CI builds both, runs race and both tagged suites (`.github/workflows/ci.yml`).
- **Files:** `justfile`, `.github/workflows/ci.yml`
- **Impact:** A green local `just ci` can still fail CI, or install a CLI whose sibling host binary is stale.
- **Fix approach:** Make `just ci` match the Ubuntu job. Install both binaries together.
- **Priority:** Medium

### Dead branches and comments that describe the wrong mechanism
- **Problem:** `KillSession` comments say the second `C-c` force-kills processes. `C-c` is only SIGINT (`internal/tmux/tmux.go`). `browsersession.Session.start` matches error strings and then does nothing in either branch (`internal/browsersession/session.go`). `createWindow` takes `windowIndex` and does not use it. Chrome `action.onClicked` and Firefox `browserAction.onClicked` only log, and a `default_popup` means the click listener does not run (`browser-extension/background.js`, `browser-extension/chrome/manifest.json`).
- **Files:** `internal/tmux/tmux.go`, `internal/browsersession/session.go`, `browser-extension/background.js`
- **Impact:** Readers trust comments that the shutdown path does not implement. The string match will break silently if the error text changes, and it does not change control flow today.
- **Fix approach:** Delete the empty branch. Fix the shutdown comment when the signal bug is fixed. Remove or implement the click handler.
- **Priority:** Low

## Known Bugs

### `devlog down` never delivers Ctrl-C, and it only looks at the current window
- **Symptoms:** Graceful stop does not signal pane processes. Confirmed on tmux 3.7c: `tmux send-keys -t session:%7` returns `can't find window: %7`. The same pane id as a bare target works. `tmux list-panes -t session` without `-s` lists only the current window. `getPaneIDs` uses that form. `send-keys` errors are ignored. The process still sleeps 500ms, sends `C-c` again, then `kill-session`.
- **Files:** `internal/tmux/tmux.go` (`KillSession`, `getPaneIDs`), `cmd/devlog/cmd_down.go`
- **Trigger:** Any `devlog down` on tmux 3.7. Multi-window sessions also miss every non-current window even after the target string is fixed.
- **Impact:** Processes do not get SIGINT before the session is destroyed. Servers that flush on SIGINT lose that chance. The sleep does not help.
- **Workaround:** Stop the servers inside the panes, then `devlog down`. `kill-session` still removes the session.
- **Priority:** High

### `run_mode: overwrite` appends
- **Symptoms:** Confirmed defect against `doc/adr/0006-timestamped-run-directories.md` ("replacing previous output"). `TouchFile` uses `O_CREATE|O_APPEND` and does not truncate. `pipe-pane` runs `cat >>`. A second `devlog up` after `down` adds to the same files. The touch unit test treats non-truncation as success, so the code and the ADR cannot both be right.
- **Files:** `internal/fileutil/touchfile.go`, `internal/tmux/tmux.go`, `doc/adr/0006-timestamped-run-directories.md`, `internal/fileutil/touchfile_test.go`
- **Trigger:** `run_mode: overwrite` and an existing log file.
- **Impact:** "Overwrite" runs keep old lines. Disk use grows for the life of the files.
- **Workaround:** Delete the log files before `up`, or use `timestamped`.
- **Priority:** Medium

### `browser.urls` in `devlog.yml` does not control capture
- **Symptoms:** Confirmed wiring gap. `cmd_up` only checks that the URL list is non-empty, then starts the wrapper with the log path and levels (`cmd/devlog/cmd_up.go`). Nothing sends `UPDATE_CONFIG`. `browser-extension/background.js` keeps a hardcoded list: `http://localhost:*/*` and `http://127.0.0.1:*/*`. `cmd_status` prints the YAML URLs under "URLs monitored". `PRIVACY.md` says collection follows `devlog.yml` patterns. A YAML entry such as `https://app.example.test:3000/*` does not enable that origin. The example file `devlog.yml.example` uses `http://localhost:3000/*`, which matches only because of the hardcoded list.
- **Files:** `cmd/devlog/cmd_up.go`, `cmd/devlog/cmd_status.go`, `browser-extension/background.js`, `PRIVACY.md`, `devlog.yml.example`
- **Trigger:** Any `browser.urls` value other than the two hardcoded patterns.
- **Impact:** Status and the privacy policy describe a filter the extension does not apply. Localhost is captured even when the YAML list names other hosts. Other hosts are not captured.
- **Workaround:** None inside the YAML file. Change `config.urls` in `browser-extension/background.js`.
- **Priority:** High

### URL match is an unanchored substring
- **Symptoms:** Confirmed. `isUrlEnabled` escapes some regex characters, turns `*` into `.*`, and never adds `^` or `$` (`browser-extension/background.js`). Evaluated with those rules: `https://evil.example/?q=http://localhost:1/x` matches `http://localhost:*/*`. `https://evil.example/` does not, which is the only negative case in `browser-extension/test/background.test.js`. The `LOG` handler forwards the payload and does not check the URL again.
- **Files:** `browser-extension/background.js`, `browser-extension/content_script.js`, `browser-extension/test/background.test.js`
- **Trigger:** A page URL that contains the pattern text, while the extension is enabled.
- **Impact:** That page's console output is sent to the native host and appended to the session log.
- **Workaround:** Do not visit URLs that embed the localhost pattern. The real fix is an anchored match plus a check on `LOG`.
- **Priority:** Medium

### YAML log levels do not match the extension filter
- **Symptoms:** Confirmed split brain. Host levels come from `cfg.Browser.Levels` via the wrapper (`internal/browsersession/session.go`, `cmd/devlog-host/main.go`). The extension filter is the in-memory list `error`, `warn`, `info`, `log` (`browser-extension/background.js`). `GET_CONFIG` overwrites the content script's wider default (`debug` and `trace` included in `browser-extension/content_script.js`). `debug` in YAML is accepted by the host and never sent. `PRIVACY.md` says `console.debug` is collected.
- **Files:** `browser-extension/background.js`, `browser-extension/content_script.js`, `internal/browsersession/session.go`, `PRIVACY.md`
- **Trigger:** `browser.levels` includes `debug` or `trace`, or omits a level the extension still emits.
- **Impact:** Users cannot turn debug capture on from YAML. The host drops extra levels only after the extension has already built the message.
- **Workaround:** Edit the hardcoded `config.levels` array.
- **Priority:** Medium

### One `--extension-id` is written into the Firefox manifest
- **Symptoms:** Confirmed from control flow. `devlog register --chrome --firefox --extension-id <chrome-id>` passes that id to `InstallFirefoxManifestWithID`. The Firefox default `devlog@devlog.local` is replaced. Firefox's packaged id is `devlog@devlog.local` (`browser-extension/firefox/manifest.json`).
- **Files:** `cmd/devlog/cmd_register.go`, `internal/manifest/manifest.go`, `browser-extension/firefox/manifest.json`
- **Trigger:** Chrome and Firefox registration in one command with a Chrome extension id.
- **Impact:** Firefox will not launch the host for the shipped extension id.
- **Workaround:** Register Firefox without `--extension-id`, in a separate command.
- **Priority:** Medium

### Session names that sanitize to the same wrapper clobber each other
- **Symptoms:** Confirmed logic bug. `sanitizeSessionForFileName` maps `/`, spaces, and `:` to `-` (`internal/browsersession/session.go`). `a/b:c` and `a-b-c` share one wrapper path (the helper test shows the first mapping). `refuseClobberActiveWrapper` treats an equal path as the same session and allows the rewrite. A live other session with a different raw name is not detected.
- **Files:** `internal/browsersession/session.go`, `internal/browsersession/helpers_test.go`
- **Trigger:** Two live tmux sessions whose names sanitize to one filename, both with browser logging.
- **Impact:** The second `devlog up` replaces the wrapper args (log path and levels) and repoints every `com.devlog.host.json` at that file. The first session's browser logs go to the second session's file.
- **Workaround:** Use session names that stay unique after sanitizing.
- **Priority:** Medium

### An oversized native frame desynchronizes the host
- **Symptoms:** Confirmed protocol bug. `ReadMessage` rejects `messageLen > 10MiB` after reading the 4-byte length and does not discard the body (`internal/natmsg/natmsg.go`). `processMessages` logs the error and continues (`cmd/devlog-host/main.go`). The next read treats payload bytes as a new length. A bad JSON body whose length prefix is correct does resync. That path is tested (`cmd/devlog-host/main_test.go`). The oversized path is not.
- **Files:** `internal/natmsg/natmsg.go`, `cmd/devlog-host/main.go`, `cmd/devlog-host/main_test.go`
- **Trigger:** One declared length above 10MiB on stdin.
- **Impact:** Later valid logs in that connection can be dropped or acked as failures until the browser closes stdin.
- **Workaround:** None in the host. The browser must reconnect.
- **Priority:** Low

## Security Considerations

### Native-messaging trust boundary
- **Risk:** The browser will start whatever path is in `com.devlog.host.json` for an allowlisted extension. `devlog up` rewrites that path to a wrapper under the user cache dir (`internal/browsersession/session.go`). The wrapper `exec`s `devlog-host` with a fixed log path. The host appends message text. It does not run the message as a command. A malicious or confused extension that is allowlisted can still write arbitrary lines into that log. The content script accepts any same-window `postMessage` with `__devlog: true` (`browser-extension/content_script.js`), so page script can forge log lines once the URL filter is on. That is the same power as `console.log`, not a sandbox escape.
- **Files:** `internal/manifest/manifest.go`, `internal/browsersession/session.go`, `cmd/devlog-host/main.go`, `browser-extension/content_script.js`, `PRIVACY.md`
- **Impact:** Log files are not an audit trail. They contain whatever the page, or an allowlisted extension, sends.
- **Current mitigation:** The browser enforces `allowed_origins` / `allowed_extensions` before it starts the host. Manifests are mode `0600`. Unix wrappers are mode `0700`. Shell args in the wrapper use `shellescape.Quote`.
- **Priority:** Medium

### Host-path check is narrower than the comment
- **Risk:** Unix `ValidateHostPath` checks that the path exists, is not a directory, and has `st_uid == getuid()`. If `Sys()` is not `*syscall.Stat_t`, it returns nil (fail open) (`internal/manifest/validate_host_unix.go`). It does not reject group- or world-writable files, and `os.Stat` follows symlinks, so the check is not on the link itself. There is a TOCTOU gap between the stat and the later browser exec. Windows `ValidateHostPath` only checks that the path exists and is not a directory (`internal/manifest/validate_host_windows.go`). The extension id is not checked for the 32-character `[a-p]` form. It is interpolated into `chrome-extension://%s/` (`internal/manifest/manifest.go`). `--extension-id '*'` would be written through as a match pattern.
- **Files:** `internal/manifest/validate_host_unix.go`, `internal/manifest/validate_host_windows.go`, `internal/manifest/manifest.go`, `cmd/devlog/cmd_register.go`, `PRIVACY.md`
- **Impact:** A path the current user owns but that another local user can rewrite still passes on Unix. On Windows, any existing file next to the CLI can be registered. A bad extension id either registers a host no real extension can call, or a broader origin if the browser treats the string as a match pattern. The second case was not executed against Chrome in this audit.
- **Current mitigation:** `FindDevlogHostBinary` only returns `devlog-host` beside the CLI executable. The user must pass `--extension-id`. Unix ownership check blocks a root-owned binary such as `/bin/sh`.
- **Priority:** Medium

### Log paths are not confined to `logs_dir`, and log files are world-readable
- **Risk:** `filepath.Join(logsDir, pane.Log)` drops `logsDir` when `pane.Log` is absolute, and it allows `..` (`internal/tmux/tmux.go` `ensurePaneLogFiles` and `sendCommandWithLogging`). `config.Validate` does not check the path. `TouchFile` creates parent directories at `0755` and the file at `0644` (`internal/fileutil/touchfile.go`). The browser log uses the same join and the logger opens `0644` (`internal/logger/logger.go`). `shellescape.Quote` stops the path from breaking out of the `cat >>` argument. It does not stop the write from leaving the project. Console text often includes tokens the app printed. Those files are readable by other local users.
- **Files:** `internal/tmux/tmux.go`, `internal/config/config.go`, `internal/fileutil/touchfile.go`, `internal/logger/logger.go`, `cmd/devlog/cmd_up.go`
- **Impact:** A shared or generated `devlog.yml` can create or append files outside `logs_dir`. On a multi-user machine, browser logs are not private.
- **Current mitigation:** The operator normally owns the config and the home directory. This is not a remote write.
- **Priority:** Medium

### Environment interpolation runs on raw YAML text, then pane commands run in a shell
- **Risk:** `interpolateEnvVars` replaces `$VAR` and `${...}` before `yaml.Unmarshal` (`internal/config/config.go`). Values are not YAML-quoted. A value with newlines or `: ` can change keys, including `cmd`. Unset names stay as the literal `$VAR` text. Pane commands then run as `sh -lc` with the whole command in one single-quoted string (`internal/tmux/tmux.go`). Metacharacters inside the command are shell syntax on purpose. Metacharacters that arrived from the environment are also shell syntax, because they were spliced into the command body.
- **Files:** `internal/config/config.go`, `internal/tmux/tmux.go`, `internal/shellescape/shellescape.go`
- **Impact:** An environment variable named by a committed `devlog.yml` can change which commands `devlog up` runs. This is local developer trust, not a remote service.
- **Current mitigation:** Interpolation is the documented feature (`README.md`). The shell wrapper is quoted as one argument, so the outer `tmux send-keys` argv is not split by the user's shell.
- **Priority:** Medium

### Privacy policy does not match extension behavior
- **Risk:** Confirmed policy break, not only a doc typo. `PRIVACY.md` says data is collected only from `devlog.yml` URL patterns, that other pages are not touched, that `storage` holds URL and level config, and that `activeTab` limits capture to the current tab. The Chrome manifest requests `<all_urls>`, `storage`, and `activeTab`, and it registers a content script on `<all_urls>` at `document_start` for all frames (`browser-extension/chrome/manifest.json`). `page_inject.js` is injected on every page before `GET_CONFIG` returns, wraps `console.*`, and `postMessage`s with target origin `*` (`browser-extension/content_script.js`, `browser-extension/page_inject.js`). No `chrome.storage` call exists in the extension sources. Capture defaults to enabled.
- **Files:** `PRIVACY.md`, `browser-extension/chrome/manifest.json`, `browser-extension/firefox/manifest.json`, `browser-extension/content_script.js`, `browser-extension/page_inject.js`, `browser-extension/background.js`
- **Impact:** Every page pays the console hook, including pages that will not be logged. The stored-config permission is unused. Store reviewers can compare the policy text to the manifest.
- **Current mitigation:** Forwarding to the host still depends on `isLoggingEnabled`. Disabled pages drop the `postMessage` in the content script. The hook still runs.
- **Priority:** High

### Windows wrapper quoting is not a full escape
- **Risk:** `batchQuote` wraps in double quotes and doubles embedded quotes (`internal/browsersession/session.go`). It does not escape `%`, `^`, `&`, or newlines. Those characters are special to `cmd.exe`. The Unix script path uses `shellescape.Quote` and has an injection test. Release builds do not ship Windows (`.goreleaser.yml`), but the batch generator is in the tree and CI compiles it.
- **Files:** `internal/browsersession/session.go`, `internal/browsersession/helpers_test.go`, `.goreleaser.yml`
- **Impact:** A log path or level containing `&` can become extra `cmd` syntax if a Windows host wrapper is used. Not exercised in this audit.
- **Current mitigation:** Paths usually come from `filepath.Abs` of a local log file. Tests cover spaces and quotes only.
- **Priority:** Low

## Performance Bottlenecks

### Console hook runs on every frame of every site
- **Problem:** Content scripts match `<all_urls>` with `all_frames: true`. Each frame appends `page_inject.js`. Every `console.log/info/warn/error/debug/trace` JSON-stringifies object arguments on the page thread, including pages that fail the URL filter.
- **Files:** `browser-extension/chrome/manifest.json`, `browser-extension/firefox/manifest.json`, `browser-extension/page_inject.js`, `browser-extension/content_script.js`
- **Impact:** Extra work and possible jank on unrelated sites, and inside ad or widget iframes on localhost. Large or cyclic objects fall back to `String`, but a large graph is still walked.
- **Priority:** High

### Localhost console traffic starts a host process whenever the port is down
- **Problem:** The extension stays enabled with the hardcoded localhost patterns. The first `LOG` calls `connectNative` (`browser-extension/background.js`). After `devlog down`, manifests point at `devlog-host` with no arguments. That process prints usage and exits (`cmd/devlog-host/main.go`). Disconnect clears the flag, so the next console line connects again.
- **Files:** `browser-extension/background.js`, `cmd/devlog-host/main.go`, `internal/browsersession/session.go`
- **Impact:** A noisy localhost page can spawn and exit the host repeatedly while devlog is not in a session.
- **Priority:** Medium

### Logs grow without a size cap, and the host holds the whole message
- **Problem:** Append-only writes have no max bytes (`internal/logger/logger.go`, `internal/tmux/tmux.go`). Retention deletes old timestamped directories only at the next `devlog up`, and only in timestamped mode (`cmd/devlog/cmd_up.go`). `ReadMessage` allocates `messageLen` up to 10MiB. The logger does not `Sync`. A crash can drop the last buffered write.
- **Files:** `internal/logger/logger.go`, `internal/natmsg/natmsg.go`, `internal/logrotate/logrotate.go`, `cmd/devlog/cmd_up.go`
- **Impact:** A long `overwrite` run, or a timestamped run left up, fills the disk. One large console dump allocates up to the cap in the host.
- **Priority:** Medium

### `pipe-pane` records the terminal, not the child process stdout
- **Problem:** Logging is `cat >> file` on the pane (`internal/tmux/tmux.go`). A local tmux 3.7c check wrote the typed command, the shell prompt, and startup noise into the file, not only `echo` output. `doc/adr/0003-tmux-for-server-log-capture.md` already notes ANSI codes. Short output was visible within 400ms on this Mac, so full `stdio` buffering of `cat` was not confirmed here.
- **Files:** `internal/tmux/tmux.go`, `doc/adr/0003-tmux-for-server-log-capture.md`
- **Impact:** Log files are larger and noisier than raw server stdout. `tail -f` shows prompts and escape sequences.
- **Priority:** Low

## Fragile Areas

### `CreateSession` leaves a live session when a later step fails
- **Issue:** `new-session` runs before env setup, `pipe-pane`, and `send-keys`. Failures return the error and do not call `kill-session` (`internal/tmux/tmux.go`). `cmd_up` then returns. The next `up` refuses to start because the session exists.
- **Files:** `internal/tmux/tmux.go`, `cmd/devlog/cmd_up.go`
- **Trigger:** tmux created the session, then `set-environment`, a split, or `send-keys` failed.
- **Impact:** A half-built session holds the name until `devlog down` or `tmux kill-session`.
- **Priority:** Medium

### Manifest restore is skipped when `KillSession` returns an error
- **Issue:** `cmd_down` calls `browsersession.Stop` only after a successful kill (`cmd/devlog/cmd_down.go`). `Stop` also ignores its own errors. `getPaneIDs` failure aborts `KillSession` before `kill-session`.
- **Files:** `cmd/devlog/cmd_down.go`, `internal/tmux/tmux.go`, `internal/browsersession/session.go`
- **Trigger:** `list-panes` fails, or `kill-session` fails.
- **Impact:** `com.devlog.host.json` can keep pointing at the wrapper.
- **Priority:** Medium

### Killing tmux outside `devlog down` leaves the wrapper in place
- **Issue:** The wrapper path and log path are fixed in the script. The manifest points at that script until `Stop` or the next successful browser `Start`. `README.md` warns about this. `RepairStaleManifestPaths` repairs a path only when `os.Stat` fails. A wrapper file that still exists is not stale (`internal/manifest/manifest.go`).
- **Files:** `internal/browsersession/session.go`, `internal/manifest/manifest.go`, `README.md`
- **Trigger:** `tmux kill-session` or a crash, without `devlog down`.
- **Impact:** The extension keeps appending browser logs to the old run directory. `devlog healthcheck` will not treat that path as broken.
- **Priority:** Medium

### Commands are sent as soon as the pane exists
- **Issue:** `send-keys` runs immediately after `new-session`, `new-window`, or `split-window`. There is no wait for the shell to read stdin (`internal/tmux/tmux.go`). Integration tests sleep after `CreateSession` and then read the log (`internal/tmux/integration_test.go`).
- **Files:** `internal/tmux/tmux.go`, `internal/tmux/integration_test.go`
- **Trigger:** A slow shell startup (login shell, heavy prompt) under load.
- **Impact:** Risk, not a failure reproduced in this audit: the keystrokes can land before the shell, and the pane command never runs. CI can flake for that reason.
- **Priority:** Medium

### Pane targets use the window name
- **Issue:** Targets are `session:windowName` (`internal/tmux/tmux.go`). Names are not checked for uniqueness, emptiness, or characters tmux treats as target syntax (`:`, `.`). `config.Validate` only checks that `cmd` is non-empty (`internal/config/config.go`).
- **Files:** `internal/tmux/tmux.go`, `internal/config/config.go`
- **Trigger:** Two windows with the same name, or a name that contains `:`.
- **Impact:** Risk: `pipe-pane` and `send-keys` can hit the wrong window or fail after the session already exists.
- **Priority:** Low

### Chrome's host lifetime is tied to a non-persistent service worker
- **Issue:** Chrome MV3 uses a service worker (`browser-extension/chrome/manifest.json`). The native port and `isNativeHostConnected` are module globals (`browser-extension/background.js`). Firefox uses a persistent background script (`browser-extension/firefox/manifest.json`). The host returns on stdin EOF (`cmd/devlog-host/main.go`). Worker suspend closes the port.
- **Files:** `browser-extension/chrome/manifest.json`, `browser-extension/firefox/manifest.json`, `browser-extension/background.js`, `cmd/devlog-host/main.go`
- **Trigger:** Chrome suspends the worker while a session is up.
- **Impact:** Risk from the platform model: a gap in capture until the next log reconnects and the browser starts the wrapper again. In-memory `UPDATE_CONFIG` changes would also vanish. Nothing in the CLI sends that message today.
- **Priority:** Medium

### `healthcheck` writes manifests, and config discovery walks parent directories
- **Issue:** `HealthCheck` calls `RepairStaleManifestPaths` (`internal/browsersession/session.go`). A read-looking command rewrites `path` fields. `findConfigFile` walks up to 20 parents and uses the first `devlog.yml` (`cmd/devlog/helpers.go`). `GetChromeNativeMessagingDir` and the Firefox helpers set `home = "/"` when `UserHomeDir` fails (`internal/manifest/manifest.go`).
- **Files:** `internal/browsersession/session.go`, `cmd/devlog/helpers.go`, `internal/manifest/manifest.go`, `cmd/devlog/cmd_healthcheck.go`
- **Trigger:** A stale manifest during `devlog healthcheck`. Running the CLI in a subdirectory of an unrelated project. `UserHomeDir` failure while registering as a user who can write `/`.
- **Impact:** Healthcheck mutates browser config. The wrong project session can start. The `/` fallback is a footgun for a root shell with a broken home. The fallback is not covered by a test that forces the error. `TestGetChromeNativeMessagingDir_FallbackToHomeOnError` does not make `UserHomeDir` fail.
- **Priority:** Low

### Early browser logs are dropped
- **Issue:** `isLoggingEnabled` starts false. `updateConfig` is async. The page script is injected immediately and can `postMessage` before the callback (`browser-extension/content_script.js`).
- **Files:** `browser-extension/content_script.js`, `browser-extension/page_inject.js`
- **Trigger:** Console output at `document_start`, which is when many app errors happen.
- **Impact:** The first logs of a page load are lost. This is a race in the source, not a timed measurement.
- **Priority:** Medium

## Scaling Limits

### One browser-log session per machine
- **Limit:** Every browser shares one manifest name, `com.devlog.host.json` (`internal/manifest/manifest.go`). `devlog up` points all of them at one wrapper. A second live session is rejected by `refuseClobberActiveWrapper`.
- **Files:** `internal/manifest/manifest.go`, `internal/browsersession/session.go`
- **Impact:** Two projects cannot capture browser consoles at the same time. Server tmux sessions can still run side by side when their session names differ.

### Pane layout is only horizontal splits
- **Limit:** Extra panes call `split-window -h` (`internal/tmux/tmux.go`). There is no layout config. tmux will refuse a split when the window is too narrow.
- **Files:** `internal/tmux/tmux.go`
- **Impact:** A window with many panes fails part way. Combined with the no-rollback behavior, the session is left behind.

### Run directories collide within one second
- **Limit:** The directory name is `time.Now().Format("20060102-150405")` (`internal/tmux/tmux.go`). Resolution is one second. There is no pid or counter.
- **Files:** `internal/tmux/tmux.go`
- **Impact:** Two timestamped starts in the same second share one directory and append to the same files. Fast `down`/`up` scripts can do this.

### Retention is directory-count and mtime, not file size
- **Limit:** `logrotate.Cleanup` deletes subdirectories of `logs_dir`. It does not look at the `YYYYMMDD-HHMMSS` name. Age is directory mtime (`internal/logrotate/logrotate.go`). `max_runs` and `retention_days` of 0 mean "do not apply". Cleanup runs only when timestamped `up` starts and the tmux session is not already up (`cmd/devlog/cmd_up.go`).
- **Files:** `internal/logrotate/logrotate.go`, `cmd/devlog/cmd_up.go`
- **Impact:** A long single run is never trimmed. A `logs_dir` that also holds other directories can lose those directories. Writing into a run does not refresh directory mtime, so a long-lived run looks old to the next cleanup from another config that shares `logs_dir`.

### Windows is built in CI and not shipped
- **Limit:** `.goreleaser.yml` builds `linux` and `darwin` for `amd64` and `arm64` only. Windows-specific host checks and the `.bat` wrapper exist. The multi-OS CI job skips integration tests on Windows (`.github/workflows/ci.yml`).
- **Files:** `.goreleaser.yml`, `.github/workflows/ci.yml`, `internal/manifest/validate_host_windows.go`, `internal/browsersession/session.go`
- **Impact:** Windows support is compile-time only. Pane capture is not tested there.

## Dependencies at Risk

### `gopkg.in/yaml.v3` v3.0.1 is the only module dependency
- **Risk:** `go.mod` requires Go 1.25.6 and `gopkg.in/yaml.v3 v3.0.1`. v3.0.1 is still the latest tag on that line and dates from 2022. The config loader is the program's trust boundary for untrusted text (see env interpolation). No newer release is available to pick up parser fixes.
- **Files:** `go.mod`, `internal/config/config.go`
- **Impact:** YAML parser bugs, if any remain in v3.0.1, stay until the project replaces the library. This audit did not match a specific CVE to this pin. Do not treat the version pin itself as a confirmed vulnerability.
- **Priority:** Low

### Extension runtime dependencies
- **Risk:** None found in the shipped extension. `browser-extension/package.json` has `vitest` and `jsdom` as devDependencies only. Packaged scripts are plain JS (`scripts/package-chrome.sh`, `scripts/package-firefox.sh`). `renovate.json` extends Renovate's recommended config for the lockfile.
- **Files:** `browser-extension/package.json`, `scripts/package-chrome.sh`, `renovate.json`
- **Impact:** Runtime supply chain for the extension is the source files themselves, not npm.

## Missing Critical Features

### No path from `devlog.yml` browser settings into the extension
- **Problem:** URL patterns and the on/off switch never leave the CLI. Levels are applied only in the host. `UPDATE_CONFIG` is implemented and unused. `storage` is declared and unused, so there is no persisted extension config either.
- **Files:** `cmd/devlog/cmd_up.go`, `browser-extension/background.js`, `browser-extension/chrome/manifest.json`, `PRIVACY.md`
- **Impact:** The documented configuration surface for browser capture does not work. See the known bug on hardcoded URLs.

### Popup and toolbar do not toggle capture
- **Problem:** `popup.js` only renders `GET_STATUS`. The background click handlers log and do not change `config.enabled`. A popup is set, so those click handlers do not run.
- **Files:** `browser-extension/popup.js`, `browser-extension/background.js`, `browser-extension/chrome/manifest.json`
- **Impact:** The user cannot turn capture off without removing the extension. Combined with the always-on localhost default, capture starts on the first matching console line.

### Overwrite does not clear logs, and down does not signal processes
- **Problem:** Both are specified by names and by ADR 0006 ("replacing") and by the `KillSession` comments ("gracefully terminates"). Neither is implemented. Listed here because callers will treat the command names as the feature.
- **Files:** `doc/adr/0006-timestamped-run-directories.md`, `internal/fileutil/touchfile.go`, `internal/tmux/tmux.go`
- **Impact:** Same as the known bugs above. No second mechanism exists (no truncate flag, no `kill-pane`).

### No per-file size limit and no private log mode
- **Problem:** Retention is optional, timestamped-only, and directory-based. There is no byte cap and no `0600` log mode.
- **Files:** `internal/logrotate/logrotate.go`, `internal/logger/logger.go`, `internal/fileutil/touchfile.go`
- **Impact:** Disk fill, and world-readable console logs, with no config key to change either.

## Test Coverage Gaps

Integration tests are behind `-tags=integration` and call `skipIfNoTmux` (`internal/tmux/integration_test.go`). They also skip when `testing.Short` is set. E2E tests are behind `-tags=e2e` and skip without tmux, but they do not honor `-short` (`internal/e2e/cli_test.go`). `go test ./...` and `just test` run neither suite. GitHub Ubuntu CI runs both. The Windows matrix job does not run integration tests.

### Shutdown signals
- **Gap:** `TestTmuxIntegration_SessionLifecycle` checks that the session disappears. It does not check that a foreground process got SIGINT, and it does not use two windows. No test uses the `session:%id` target that tmux 3.7c rejects.
- **Files:** `internal/tmux/integration_test.go`, `internal/tmux/tmux_test.go`, `internal/tmux/tmux.go`
- **Impact:** The confirmed Ctrl-C bug is green in CI.

### Browser config contract
- **Gap:** No test connects `devlog.yml` `browser.urls` or `browser.levels` to `GET_CONFIG`. The extension test only checks `http://localhost:3000/` (enabled) and `https://evil.example/` (disabled). It does not check a URL that contains the pattern, `debug`, or `type: "ACK"`. The host ack JSON has `success` and `error` and no `type` field (`internal/natmsg/natmsg.go`). The extension looks for `message.type === "ACK"` (`browser-extension/background.js`). That mismatch is untested and is a dead handler, not a dropped log.
- **Files:** `browser-extension/test/background.test.js`, `browser-extension/background.js`, `cmd/devlog/cmd_up.go`, `internal/natmsg/natmsg.go`
- **Impact:** The high-priority config bug and the unanchored matcher stay green.

### Path and config safety
- **Gap:** No test that `pane.Log` of `/tmp/x` or `../x` stays under `logs_dir`. No test that an env value with a newline changes or is rejected by `Load`. Interpolation tests only check a port number and a project name (`internal/config/config_test.go`). No test that two session names sanitize to one wrapper while both sessions are live.
- **Files:** `internal/config/config_test.go`, `internal/tmux/tmux_test.go`, `internal/browsersession/browsersession_test.go`
- **Impact:** The write-outside-logs and YAML-splice risks have no regression net.

### Host validation and native-message recovery
- **Gap:** `TestValidateHostPath_Missing` covers a missing file only (`internal/manifest/manifest_test.go`). No test uses a file owned by another uid, a world-writable file, or the Windows build. `TestHost_ReadMessage_TooLarge` checks the error and not the following message (`internal/natmsg/natmsg_test.go`). `TestRun_MalformedMessageContinues` covers a short bad JSON body only.
- **Files:** `internal/manifest/manifest_test.go`, `internal/manifest/validate_host_unix.go`, `internal/manifest/validate_host_windows.go`, `internal/natmsg/natmsg_test.go`, `cmd/devlog-host/main_test.go`
- **Impact:** Fail-open and stream desync are unguarded.

### Retention and overwrite
- **Gap:** Logrotate tests use directories the test itself created. They do not add a non-timestamp directory, a live session directory, or an absolute `logs_dir` of `.`. No test expects overwrite to truncate, and one test expects the opposite (`internal/fileutil/touchfile_test.go`).
- **Files:** `internal/logrotate/logrotate_test.go`, `internal/fileutil/touchfile_test.go`, `internal/e2e/cli_test.go`
- **Impact:** Cleanup can delete unexpected directories without a failing test. The ADR overwrite contract is not what CI checks.

### End-to-end browser path
- **Gap:** E2E builds `cmd/devlog` only (`internal/e2e/cli_test.go`). It does not build `devlog-host`, register a manifest, or send a native message. Extension tests mock Chrome. They do not load the MV3 service worker or Firefox's persistent page.
- **Files:** `internal/e2e/cli_test.go`, `browser-extension/test/background.test.js`, `.github/workflows/ci.yml`
- **Impact:** The extension-to-host path, including the wrapper and manifest rewrite, is untested in CI.

### Batch escaping
- **Gap:** Tests check spaces and embedded double quotes (`internal/browsersession/helpers_test.go`). They do not check `%` or `&`.
- **Files:** `internal/browsersession/helpers_test.go`, `internal/browsersession/session.go`
- **Impact:** The Windows quoting risk has no failing test. CI still would not run it as a real `cmd` script.

---
*Concerns audit: 2026-10-01*
