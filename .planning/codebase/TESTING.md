# Testing Patterns

**Analysis Date:** 2026-10-01

## Test Framework

**Runner:**

- Go uses the standard `testing` package. There is no testify, gomega, gomock, or quicktest dependency in `go.mod`. The only non-stdlib test helper beyond `testing` is `testing/quick` in `internal/shellescape/shellescape_test.go`.
- The browser extension uses Vitest 4 with jsdom (`browser-extension/package.json`, `browser-extension/vitest.config.js`). Tests call `describe`, `it`, `expect`, and sometimes `beforeEach` / `vi` (`browser-extension/test/page_inject.test.js`).

**Assertion Library:**

- Go: none. Failures use `t.Fatal`, `t.Fatalf`, `t.Error`, `t.Errorf`, `t.Skip`, and `t.Skipf`. The usual comparison text is `got` / `want`:

```go
if cfg.Version != "1.0" {
    t.Errorf("Version = %q, want %q", cfg.Version, "1.0")
}
```

That check is from `TestLoad_ValidConfig` in `internal/config/config_test.go`. Setup failures use `Fatalf` so the test stops. Field checks use `Errorf` so later checks still run.

- Extension: Vitest `expect`. Example from `browser-extension/test/background.test.js`:

```js
expect(status.connected).toBe(false);
expect(status.enabled).toBe(true);
```

**Run Commands:**

From `justfile`:

- `just test` — `go test ./...` (no race, no `integration` tag, no `e2e` tag).
- `just test-v` — `go test -v ./...`.
- `just test-one TestLoad_ValidConfig` — `go test -run {{name}} ./...`.
- `just test-cover` — `go test -cover ./...`.
- `just test-race` — `go test -race ./...`.
- `just test-integration` — `go test -tags=integration ./internal/tmux/`.
- `just ci` — `just lint` then `just test`. It does not run race, integration, or e2e.
- `go test -short ./...` does not compile `integration` or `e2e` tests, because those files are build-tagged. It does not skip `internal/tmux/tmux_test.go`. That file has no `testing.Short` check and calls real tmux unless tmux is missing.
- `go test -short -tags=integration ./internal/tmux/` does skip the integration file. `skipIfNoTmux` in `internal/tmux/integration_test.go` calls `t.Skip("skipping integration test")` when `testing.Short()` is true, and `t.Skip("tmux not available in PATH")` when `exec.LookPath("tmux")` fails.

From `.github/workflows/ci.yml`:

- `test` job: install tmux, `gofmt -l .`, `go vet ./...`, `go test -race -coverprofile=coverage.txt -covermode=atomic ./...`, `go test -tags=integration ./internal/tmux/`, `go test -tags=e2e -v ./internal/e2e/`, then `go build` for `./cmd/devlog` and `./cmd/devlog-host`.
- `test-extension` job: `working-directory: browser-extension`, Node 24, `npm ci`, `npm test` (`vitest run`).
- `multi-platform` job: `go test ./...` on Ubuntu, macOS, and Windows. Integration tests run only when `runner.os != 'Windows'`. This job does not run e2e or extension tests.

## Test File Organization

**Location:**

- Go tests are colocated with the package. There is no `testdata/` directory.
- Unit-style files (default `go test` build):
  - `internal/config/config_test.go`
  - `internal/natmsg/natmsg_test.go`
  - `internal/logger/logger_test.go`
  - `internal/logrotate/logrotate_test.go`
  - `internal/manifest/manifest_test.go`
  - `internal/shellescape/shellescape_test.go`
  - `internal/fileutil/touchfile_test.go`
  - `internal/browsersession/browsersession_test.go`
  - `internal/browsersession/helpers_test.go`
  - `internal/tmux/tmux_test.go` (real tmux, skip if absent; not behind the integration tag)
  - `cmd/devlog/init_test.go`, `cmd/devlog/status_test.go`, `cmd/devlog/healthcheck_test.go`
  - `cmd/devlog-host/main_test.go`
- Tagged files:
  - `internal/tmux/integration_test.go` — `//go:build integration`
  - `internal/e2e/cli_test.go` — `//go:build e2e`
- Extension tests live under `browser-extension/test/`: `background.test.js`, `content_script.test.js`, `page_inject.test.js`, and `mocks/chrome.js`. Vitest includes only `test/**/*.test.js`.

**Naming:**

- Go files are `*_test.go` or the specific names `integration_test.go` and `helpers_test.go`.
- Go test functions are `TestName_Description`: `TestLoad_MissingRequiredFields`, `TestHost_ReadMessage_EOF`, `TestCleanup_DryRun`, `TestCmdInit_CreatesFile`, `TestRun_RequiresLogPath`, `TestE2E_UpFailsWhenAlreadyRunning`.
- Extension files are `<script>.test.js`. The `describe` string is the script name (`"background.js"`, `"page_inject.js"`).

**Structure:**

- Every Go test is `package` equal to the code under test (`package config`, `package main`), not an external `foo_test` package. Tests can call unexported functions such as `run`, `cmdInit`, and `start`.
- Extension tests do not import the scripts as ES modules. They `readFileSync` the source and evaluate it in a `vm` context or a jsdom `window`.

## Test Structure

**Suite Organization:**

- No `TestMain` except `internal/e2e/cli_test.go`, which builds `../../cmd/devlog` into a temp binary once, stores the path in `binary`, runs `m.Run()`, then deletes the temp dir.
- No `t.Parallel()` anywhere in `*_test.go`.
- Subtests use `t.Run` in five places: `internal/config/config_test.go`, `internal/fileutil/touchfile_test.go`, `internal/manifest/manifest_test.go`, `cmd/devlog/init_test.go`, and `internal/tmux/integration_test.go`. Many cases are separate top-level `Test*` functions instead of a table (`internal/logger/logger_test.go`, `internal/natmsg/natmsg_test.go`).
- Helpers call `t.Helper()`: `encodeMessage` (`internal/natmsg/natmsg_test.go`), `encodeNativeMessage` and `decodeAck` (`cmd/devlog-host/main_test.go`), `withIsolatedHome` and `readChromePath` (`internal/browsersession/browsersession_test.go`), `skipIfNoTmux` and `runDevlog` (`internal/e2e/cli_test.go`).

**Patterns:**

Table-driven validation, from `TestLoad_MissingRequiredFields` in `internal/config/config_test.go`:

```go
tests := []struct {
    name    string
    content string
    wantErr string
}{
    {
        name:    "missing version",
        content: `...`,
        wantErr: "version is required",
    },
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        _, err := Load(configPath)
        if err == nil {
            t.Errorf("Load() expected error containing %q, got nil", tt.wantErr)
            return
        }
        if !strings.Contains(err.Error(), tt.wantErr) {
            t.Errorf("Load() error = %q, want containing %q", err.Error(), tt.wantErr)
        }
    })
}
```

`internal/fileutil/touchfile_test.go` uses a smaller table whose case body is a named function (`t.Run(tt.name, tt.fn)`). `internal/browsersession/helpers_test.go` uses an `in` / `want` table with no name field for `TestSanitizeSessionForFileName`.

A single-case test still uses the `TestSubject_Condition` name, builds state with `t.TempDir()`, and checks one behavior. `TestLoad_Defaults` and `TestLoad_EnvVarInterpolation` in `internal/config/config_test.go` follow that shape. Env changes are restored with `defer os.Unsetenv(...)`.

Manifest tests sometimes mark sections `Arrange` / `Act` / `Assert` (`internal/manifest/manifest_test.go`). Other packages do not.

Extension shape, from `browser-extension/test/background.test.js`:

```js
describe("background.js", () => {
    it("does not auto-connect on load", () => {
        const { chrome } = loadBackground();
        expect(chrome._lastNativeHost).toBeUndefined();
    });
});
```

`loadBackground` builds a `chrome` mock, runs `background.js` with `vm.runInContext`, and returns the mock plus captured console calls.

## Mocking

**Framework:**

None for Go. No gomock, no mockery, no testify mocks. The extension hand-rolls a Chrome stub in `browser-extension/test/mocks/chrome.js` (`createChromeMock`). It is not `vitest-chrome` or `sinon`.

**Patterns:**

- Inject streams instead of stdin. `NewHostWithStreams(reader, writer)` in `internal/natmsg/natmsg.go` is documented as the test constructor. Tests pass `bytes.NewReader` and `bytes.Buffer` (`internal/natmsg/natmsg_test.go`).
- Inject the host entry point. `run(args, stdin, stdout, stderr)` in `cmd/devlog-host/main.go` is what `cmd/devlog-host/main_test.go` calls. Native frames are built by `encodeNativeMessage`.
- Inject interfaces at the edge that touches other packages. `fixedHostManifest` in `internal/browsersession/browsersession_test.go` implements `ManifestOps` and delegates most methods to `manifest` while pinning `FindDevlogHostBinary`. `realSessionChecker` still calls `tmux.NewRunner`.
- `withIsolatedHome` points `HOME` at `t.TempDir()` and clears `XDG_CONFIG_HOME` so manifest writes do not touch the developer home directory.
- Extension mocks record `connectNative`, `postMessage`, `sendMessage`, and listener lists on `chrome._lastNativeHost`, `chrome._nativeMessages`, and `chrome._listeners`.

**What to Mock:**

- Process boundaries and host-wide paths: stdin/stdout, `HOME`, the native-host path, and `chrome.*`.
- The extension has no browser in CI, so the Chrome API and `console` are stubbed. Page and content-script tests use jsdom (`browser-extension/test/content_script.test.js`, `browser-extension/test/page_inject.test.js`).

**What NOT to Mock:**

- The filesystem. Tests create real files with `t.TempDir()` and `os.WriteFile` (`internal/config/config_test.go`, `internal/logger/logger_test.go`, `internal/logrotate/logrotate_test.go`).
- YAML parsing and config validation. `Load` reads a real temp `devlog.yml`.
- Shell quoting. `TestQuote_InjectionCannotBreakOut` and `TestQuote_RoundTripProperty` run `sh -c` (`internal/shellescape/shellescape_test.go`). The property test skips when `sh` is not on `PATH`.
- tmux, in `internal/tmux/tmux_test.go`, `internal/tmux/integration_test.go`, and `internal/e2e/cli_test.go`. Those tests skip or fail when tmux is absent. They are not fakes.
- The `devlog` binary in e2e. `TestMain` runs `go build -o <tmp>/devlog ../../cmd/devlog` and the tests exec that binary.

## Fixtures and Factories

**Test Data:**

- YAML and JSON live inline in the test function. `TestLoad_ValidConfig` assigns a raw string to `content` and writes it under `t.TempDir()`.
- Log-run directories use fixed timestamp names such as `20240101-120000` (`internal/logrotate/logrotate_test.go`). Retention tests move mtime with `os.Chtimes`.
- Native messages are Go structs passed through `encodeMessage` or `encodeNativeMessage`, which prefix a native-endian uint32 length. `sampleMessage` in `cmd/devlog-host/main_test.go` fills a `natmsg.Message` with a fixed `time.Date`.
- E2e config is `writeConfig` in `internal/e2e/cli_test.go`: a `fmt.Sprintf` template with `version`, `project`, `logs_dir`, `run_mode: overwrite`, one window, and two panes.
- Session names include `time.Now().UnixNano()` (`generateTestSessionName`, `sessionName`) so parallel developers do not share one tmux name. Integration tests `defer` `KillSession` when the session still exists.
- There is no golden-file directory and no factory package.

**Location:**

- Helpers stay in the `*_test.go` file that uses them, or in `internal/browsersession/helpers_test.go` for pure string cases.
- The only shared extension fixture module is `browser-extension/test/mocks/chrome.js`.

## Coverage

**Requirements:**

None. No coverage percent is enforced in `justfile`, `go.mod`, or `.github/workflows/ci.yml`. There is no `codecov.yml`. The Codecov step uses `codecov/codecov-action@v7` with `continue-on-error: true`, so a missing `CODECOV_TOKEN` or an upload failure does not fail CI. `README.md` only shows a Codecov badge.

The profile written to `coverage.txt` is from `go test -race -covermode=atomic ./...`. That command does not pass `-tags=integration` or `-tags=e2e`, so `internal/tmux/integration_test.go` and `internal/e2e/cli_test.go` are absent from that profile. Extension coverage is not collected. `npm test` is `vitest run` with no coverage flag.

**View Coverage:**

- `just test-cover` prints a per-package percent (`go test -cover ./...`).
- CI writes `coverage.txt` at the repo root and uploads it. The file is not kept as a source artifact in git.

## Test Types

**Unit Tests:**

- Default `go test ./...` runs the colocated `*_test.go` files listed above. They cover config load and validation, native-message framing, logger formatting and level filters, log rotation (including dry-run), manifest path layout and JSON contents, shell quoting, `TouchFile`, browser-session wrapper setup under a fake `HOME`, and CLI `init` / `status` / `healthcheck`.
- `internal/tmux/tmux_test.go` is compiled in this set but talks to a real tmux server. It skips when tmux is not on `PATH`. Treat it as a live session test, not a pure unit test.
- `cmd/devlog/healthcheck_test.go` allows the error text `healthcheck failed` because tmux or `devlog-host` may be missing. It still fails on any other error.
- Extension unit tests load each script in isolation. They do not start Chrome or Firefox.

**Integration Tests:**

- File: `internal/tmux/integration_test.go`. Build tag: `integration`.
- Require tmux on `PATH`. CI installs tmux first (`sudo apt-get install -y tmux` on Ubuntu; Homebrew on macOS). Windows CI skips this step (`if: runner.os != 'Windows'`).
- `skipIfNoTmux` also skips when `-short` is set, so a short integration run does not open sessions.
- Cases cover session create, multiple windows and panes, duplicate session names, and log file names with spaces, dashes, underscores, and dots (`TestTmuxIntegration_SpecialCharactersInPaths`). Cleanup kills the session in a `defer`.
- Run: `just test-integration` or `go test -tags=integration ./internal/tmux/`. Verbose: `just test-integration-v`.

**E2E Tests:**

- File: `internal/e2e/cli_test.go`. Build tag: `e2e`. Package comment says these tests need tmux and are for CI or pre-release checks.
- `TestMain` builds the CLI. Tests call `runDevlog` and a real tmux session. Names look like `devlog-e2e-<unix nano>`.
- Cases: `TestE2E_Lifecycle` (up, status, down), `TestE2E_UpFailsWhenAlreadyRunning`, `TestE2E_Healthcheck`, `TestE2E_Init`, `TestE2E_DownWithoutSession`, `TestE2E_TimestampedMode`.
- CI command: `go test -tags=e2e -v ./internal/e2e/` in the Ubuntu `test` job only. `just test` and `just ci` do not run them. There is no Playwright or browser-store e2e suite.

## Common Patterns

**Async Testing:**

- Go tests do not use a fake clock, `synctest`, or `t.Parallel`. Timing is real. `internal/tmux/integration_test.go` sleeps `100 * time.Millisecond` after `CreateSession` before it stats the log file. `internal/logrotate/logrotate_test.go` sets directory mtime rather than waiting for retention days.
- The logger mutex is not stressed by a concurrent test.
- Extension tests are synchronous. The Chrome mock invokes `sendMessage` callbacks immediately (`browser-extension/test/mocks/chrome.js`). Tests do not use `async` / `await` or fake timers. `vi` is imported by `page_inject.test.js` for local spies, not for a timer queue.
- Native-host tests write a full stdin buffer, then call `run`, which reads until EOF. They do not start a second process.

**Error Testing:**

- Expected failure: `err == nil` then `t.Fatal` or `t.Error`, then `strings.Contains(err.Error(), substring)`. Tests do not use `errors.Is` on the returned error. Config wants substrings such as `version is required` and `run_mode must be 'timestamped' or 'overwrite'` (`internal/config/config_test.go`). Tmux wants `already exists` and `does not exist` (`internal/tmux/tmux_test.go`).
- EOF is identity, not a substring. `TestHost_ReadMessage_EOF` requires `err == io.EOF` (`internal/natmsg/natmsg_test.go`).
- Host argument errors also check stderr. `TestRun_RequiresLogPath` requires the word `Usage:` on stderr (`cmd/devlog-host/main_test.go`).
- Malformed input is "error, and continue" at the product level. `TestRun_MalformedMessageContinues` feeds a bad frame and still expects the process helper to finish the rest of the stream (`cmd/devlog-host/main_test.go`).
- Skip is the third outcome, used when a tool is missing (`tmux`, `sh`) or when `-short` is set on the integration build. A skip is not a pass of the behavior.
- Extension failures use `expect(...).toBe(...)` and, for a missing listener, `throw new Error("no onMessage listener registered")` in `getHandler` (`browser-extension/test/background.test.js`).

---

*Testing analysis: 2026-10-01*
