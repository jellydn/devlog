# Coding Conventions

**Analysis Date:** 2026-10-01

## Naming Patterns

**Files:**

- Go packages are one lowercase directory name with no underscores: `internal/config`, `internal/natmsg`, `internal/browsersession`, `internal/logrotate`, `internal/shellescape`, `internal/fileutil`.
- The main source file usually matches the package: `internal/config/config.go`, `internal/natmsg/natmsg.go`, `internal/tmux/tmux.go`. Extra files name the concern: `internal/fileutil/touchfile.go`, `internal/browsersession/session.go`.
- CLI commands are one file per command under `cmd/devlog/`: `cmd_up.go`, `cmd_down.go`, `cmd_attach.go`, `cmd_status.go`, `cmd_ls.go`, `cmd_open.go`, `cmd_register.go`, `cmd_init.go`, `cmd_healthcheck.go`. Shared CLI code is `main.go` and `helpers.go`.
- OS-specific files use a suffix plus a build tag: `internal/manifest/validate_host_unix.go` (`//go:build !windows`) and `internal/manifest/validate_host_windows.go` (`//go:build windows`).
- Tests sit next to the code as `*_test.go` (`internal/config/config_test.go`). Extra test helpers use `helpers_test.go` (`internal/browsersession/helpers_test.go`). The tmux integration file is `internal/tmux/integration_test.go`.
- The browser extension is plain JavaScript, not TypeScript. Entry scripts use snake_case: `browser-extension/background.js`, `browser-extension/content_script.js`, `browser-extension/page_inject.js`. Tests are `browser-extension/test/*.test.js`. The Chrome mock is `browser-extension/test/mocks/chrome.js`.

**Functions:**

- Exported Go functions and methods are PascalCase and start with a verb or `New`/`Get`: `Load`, `Validate`, `NewHost`, `NewHostWithStreams`, `CreateSession`, `TouchFile`, `GetChromeNativeMessagingDir` in `internal/config/config.go`, `internal/natmsg/natmsg.go`, `internal/tmux/tmux.go`, `internal/fileutil/touchfile.go`, and `internal/manifest/manifest.go`.
- Unexported helpers are camelCase: `interpolateEnvVars`, `parseOptionalInt`, `ensurePaneLogFiles`, `formatLoc`, `rewriteManifestPath`.
- CLI handlers are unexported `cmd` + command name: `cmdUp`, `cmdInit`, `cmdHealthcheck`. They share the signature `func(cfg *config.Config, args []string) error` declared as `Command` in `cmd/devlog/main.go`.
- The native host keeps `main` thin and puts behavior in `run` and `processMessages` (`cmd/devlog-host/main.go`) so tests can call them.
- Test functions are `Test<Subject>_<Condition>`: `TestLoad_ValidConfig`, `TestHost_ReadMessage_Success`, `TestTmuxIntegration_CreateSession`, `TestE2E_Lifecycle`, `TestQuote_RoundTripProperty`.
- Extension functions are camelCase (`connectToNativeHost`, `loadBackground`, `createChromeMock`). Test titles are plain sentences inside `describe` / `it` (`browser-extension/test/background.test.js`).

**Variables:**

- Locals and fields are camelCase: `configPath`, `logsDir`, `hostPath`, `extensionID`, `dryRun`.
- Initialisms stay uppercase in Go (`URL`, `ID` as in `extensionID`). YAML fields use snake_case tags (`logs_dir`, `run_mode`, `max_runs` in `internal/config/config.go`). JSON fields use snake_case (`allowed_origins`, `allowed_extensions` in `internal/manifest/manifest.go`).
- Receivers are one letter or a short word: `c` for `Config`, `l` for `Logger`, `h` for `Host`, `r` for `Runner`, `s` for `Session`, `m` for `Message`.
- Errors are `err`. A second error uses a specific name such as `remErr` in `cmd/devlog/cmd_up.go`.
- Test tables use `tests` and `tt`, or `testCases` and `tc` (`internal/tmux/integration_test.go`). Comparisons use `got` / `want`.
- Constants are MixedCaps in Go (`ManifestFileName` in `internal/manifest/manifest.go`) and `SCREAMING_SNAKE` in the extension (`NATIVE_HOST_NAME` in `browser-extension/background.js`).
- File modes are numeric literals: directories `0755`, new files `0644` (`internal/fileutil/touchfile.go`), rewritten manifests `0600` (`internal/manifest/manifest.go`).

**Types:**

- Exported structs are PascalCase, often with a `Config` or role suffix: `Config`, `TmuxConfig`, `WindowConfig`, `PaneConfig`, `BrowserConfig`, `SessionConfig`, `ChromeManifest`, `FirefoxManifest`, `HealthResult`, `Policy`, `Result`.
- Interfaces are named for the role. Consumer-side seams are exported when tests in another sense need them (`ManifestOps`, `SessionChecker` in `internal/browsersession/session.go`) and unexported when they only narrow a local dependency (`messageLogger` in `cmd/devlog-host/main.go`).
- The only named function type is `Command` in `cmd/devlog/main.go`. There is no `iota` enum set. Run mode is a string checked against `"timestamped"` and `"overwrite"` in `(*Config).Validate`.

## Code Style

**Formatting:**

- Go is `gofmt` (tabs for indent). `gofmt -l .` is empty on this tree as of this analysis. CI fails the job when `gofmt -l .` prints any path (`.github/workflows/ci.yml`, both the `test` job and the `multi-platform` job).
- `just fmt` runs `go fmt ./...`, which rewrites files. The CI step only lists diffs; it does not rewrite.
- The extension is also tab-indented. `browser-extension/background.js` and `browser-extension/vitest.config.js` use tabs, double quotes, semicolons, and `const` / `let`. There is no Prettier or ESLint config in `browser-extension/`.

**Linting:**

- There is no `.golangci.yml`, no `staticcheck`, and no `goimports` step. Lint is format plus vet.
- `just lint` in `justfile` runs `go fmt ./...` and then `go vet ./...`. `just ci` runs `lint` then `test` (`go test ./...` only).
- GitHub Actions (`.github/workflows/ci.yml`) also runs `go vet ./...`, `go test -race`, integration tests, e2e tests, and both binary builds. The `test-extension` job runs `npm ci` and `npm test` in `browser-extension/` on Node 24. The multi-platform job repeats `gofmt` and `go vet` on Ubuntu, macOS, and Windows.
- `go.mod` sets `go 1.25.6`. The only external module is `gopkg.in/yaml.v3`.

## Import Organization

**Order:**

- `gofmt` sorts each import group and aligns blocks. The usual layout, which CI accepts, is the standard library, a blank line, then this module. Example from `internal/logger/logger.go`: `fmt`, `os`, `path/filepath`, `strings`, `sync`, then `github.com/jellydn/devlog/internal/natmsg`. The same split is in `cmd/devlog/main.go`, `cmd/devlog-host/main.go`, and `internal/tmux/tmux.go`.
- External modules use the same third group as internal imports. `internal/config/config.go` puts `gopkg.in/yaml.v3` after the stdlib group. Nothing else is imported from outside the module.
- `gofmt` does not merge groups. `internal/tmux/tmux_test.go` is gofmt-clean but splits the stdlib: `fmt` and `time` sit in the second group with `github.com/jellydn/devlog/internal/config`. New files should follow the stdlib-then-module split used everywhere else, not that file.
- No import aliases appear in `*.go` files (no `name "path"` lines).

**Path Aliases:**

None. `go.mod` has no `replace` directive. There is no `tsconfig.json` and no Vitest `resolve.alias`. `browser-extension/vitest.config.js` only sets `environment: "jsdom"` and `include: ["test/**/*.test.js"]`. Extension tests import with relative paths (`./mocks/chrome.js`) and `node:` specifiers (`node:fs`, `node:vm`).

## Error Handling

- Library and command code returns `error`. The dominant wrap is `fmt.Errorf("lowercase context: %w", err)`, for example `failed to read config file` in `internal/config/config.go`, `failed to open log file` in `internal/logger/logger.go`, `touch %q` in `internal/fileutil/touchfile.go`, and `host path %q` in `internal/manifest/validate_host_unix.go`.
- New failures that do not wrap also start lowercase: `config: version is required`, `tmux session '%s' already exists`, `message too large: %d bytes`, `at least one window with one pane is required`.
- Two strings do not follow that lowercase rule. `internal/manifest/manifest.go` returns `Firefox extension ID is required` (capital because of the product name). `cmd/devlog-host/main.go` returns `fmt.Errorf("Error: failed to create logger: %v\n", err)`, which capitalizes `Error`, uses `%v` instead of `%w`, and embeds a newline. Callers should not copy that form.
- Sentinel and OS errors:
  - `internal/natmsg/natmsg.go` returns `io.EOF` unwrapped when the length prefix hits EOF. `cmd/devlog-host/main.go` checks `err == io.EOF` and then returns nil so a closed browser is a clean exit.
  - `rewriteManifestPath` in `internal/manifest/manifest.go` returns `os.ReadFile` errors unwrapped on purpose so callers can use `errors.Is`. The comment above that function says so. Other manifest reads use `errors.Is(err, os.ErrNotExist)`.
  - `internal/logrotate/logrotate.go` still uses `os.IsNotExist(err)` and treats a missing logs directory as an empty success.
- Partial failure is one error string, not `errors.Join`. Manifest update and read failures are joined with `"; "` (`failed to update native messaging manifests`, `failed to read some manifests`).
- Validation errors name the field path: `config: tmux.windows[%d].panes[%d].cmd is required` in `internal/config/config.go`.
- The CLI boundary prints and exits. `cmd/devlog/main.go` writes `Error loading config`, `Unknown command`, or `Error: %v` to stderr and calls `os.Exit(1)`. Command functions themselves return the error and do not exit.
- Non-fatal problems are warnings, not returned errors. `cmd/devlog/cmd_up.go` prints `Warning: failed to cleanup old runs` and still starts the session. `(*Session).Stop` in `internal/browsersession/session.go` returns nothing and ignores a host-lookup error.

## Logging

**Framework:**

None. No Go file imports `log` or `log/slog`, and there is no Zap, Logrus, or similar dependency in `go.mod`. `internal/logger` is a file writer for browser console lines, not a logging facade.

**Patterns:**

- Packages under `internal/` return errors. They do not print. `Logger.Log` in `internal/logger/logger.go` formats one line and writes it to the open file. On write failure it returns `failed to write log`.
- Line shape from `Logger.Log`: `[TIMESTAMP] [LEVEL]` plus optional ` [URL]`, optional `source:line:column`, then `: message`. Timestamp layout is `2006-01-02 15:04:05.000`. Level is uppercased. Level filters are case-insensitive; an empty level list logs everything (`ShouldLog`).
- The file is opened with `O_CREATE|O_WRONLY|O_APPEND` (`internal/logger/logger.go`). Writes take `sync.Mutex`.
- User-facing CLI text uses `fmt.Printf` / `fmt.Println` for status (`Starting devlog session`, healthcheck lines in `cmd/devlog/cmd_healthcheck.go`) and `fmt.Fprintf(os.Stderr, ...)` for warnings and hard errors.
- `devlog-host` writes protocol problems to stderr and keeps reading: `Error reading message`, `Error writing log`, `Error sending ack` in `processMessages` (`cmd/devlog-host/main.go`). EOF is not logged as a failure.
- The extension logs with `console.log` / `console.warn` / `console.error` and a `devlog:` prefix (`browser-extension/background.js`). That is browser-console text, not the Go logger.

## Comments

**When to Comment:**

- Package comments document purpose on `internal/natmsg`, `internal/logger`, `internal/shellescape`, `internal/fileutil`, `internal/logrotate`, `internal/manifest`, `internal/browsersession`, and `internal/e2e`. `internal/config` and `internal/tmux` have no package comment.
- Exported functions usually have one godoc sentence that starts with the function name: `Load reads and parses the devlog.yml file`, `Quote returns a shell-safe single-quoted representation`. Some exported manifest types and getters have no comment (`ChromeManifest`, `GetChromeNativeMessagingDir` in `internal/manifest/manifest.go`).
- Comments that stay explain a constraint, not the next statement. Examples: why `rewriteManifestPath` returns `os.ErrNotExist` unwrapped; why `ValidateHostPath` checks ownership (`internal/manifest/validate_host_unix.go`); why `SessionConfig` keeps window types in `internal/config` (`internal/tmux/tmux.go`); why Windows permission bits are not asserted (`internal/fileutil/touchfile_test.go`).
- Protocol comments in `internal/natmsg/natmsg.go` state the wire rule (4-byte native-endian length, 10MB cap, timestamp as RFC3339 or Unix milliseconds).
- A few manifest tests use `// Arrange`, `// Act`, and `// Assert` (`internal/manifest/manifest_test.go`). That is local test commentary, not a required style. Most other tests use a short intent comment or none.
- Extension files start with a `//` banner (`browser-extension/background.js`) and comment non-obvious browser behavior inline.

**JSDoc/TSDoc:**

None. A search of `*.go` and `*.js` finds no `@param`, `@returns`, or `@type` tags. The extension has no TypeScript, so TSDoc does not apply. Comments are `//` lines only.

## Function Design

**Size:**

- Production Go has about 104 functions. The median length is about 18 lines. Fourteen are longer than 40 lines. Five are longer than 60.
- The long functions are CLI flows: `cmdRegister` (`cmd/devlog/cmd_register.go`, about 105 lines), `cmdHealthcheck` (about 77), `cmdStatus` (about 73), `cmdUp` (about 59), `generateTemplate` (about 55). Library outliers are `Cleanup` (`internal/logrotate/logrotate.go`, about 77) and `CreateSession` (`internal/tmux/tmux.go`, about 63).
- New library functions should stay near the median. Command handlers may be longer because they sequence I/O and print status. Split when a helper has its own error context, as `ensurePaneLogFiles` and `formatLoc` already do.

**Parameters:**

- Most functions take one to four parameters. Constructors take only what they store: `New(logPath string, levels []string)`, `NewRunner(sessionName string)`, `New(manifest ManifestOps, tmux SessionChecker)`.
- Related tmux inputs are one struct, `SessionConfig`, instead of a long argument list (`internal/tmux/tmux.go`). Retention inputs are `Policy`.
- Every CLI command takes `(cfg *config.Config, args []string)` even when `cfg` is nil (`init`, `register`, `healthcheck`). Extra flags stay in `args`.
- `run` on the host takes `args`, `stdin`, `stdout`, and `stderr` so tests do not touch `os.Stdin` (`cmd/devlog-host/main.go`).
- Boolean flags are explicit (`dryRun bool` on `Cleanup`). Context is not passed; nothing uses `context.Context`.

**Return Values:**

- Fallible functions return `error` last. Constructors and loaders return `(*T, error)`: `*Config`, `*Logger`, `*Host` via a non-error constructor, `*Message`, `*Result`, `*HealthResult`.
- Success is a nil error. Empty results are valid: `Cleanup` returns an empty `*Result` when no policy is set or the directory is missing.
- Pure helpers return a value only: `Quote`, `ShouldLog`, `SessionExists`, `LogPath`.
- Pair returns are `(int, error)`, `(string, error)`, or `(map[string]string, error)` (`RepairStaleManifestPaths`, `FindDevlogHostBinary`, `ReadManifestPaths`).
- `Stop` returns nothing and drops the host-lookup error. Do not copy that for new code that the caller must notice.
- Early returns are the normal control flow. Nested `else` after a handled error is rare.

## Module Design

**Exports:**

- Visibility is Go's capital letter. Tests use `package config`, not `package config_test`, so they can call unexported helpers. The same is true for `natmsg`, `tmux`, `manifest`, `browsersession`, `logger`, `logrotate`, `shellescape`, `fileutil`, and both `package main` test files.
- Cross-package calls use only exported names: `config.Load`, `tmux.NewRunner`, `logger.New`, `natmsg.NewHostWithStreams`, `manifest.InstallChromeManifest`, `shellescape.Quote`.
- Small interfaces live next to the caller that needs a fake. `ManifestOps` is the seam `browsersession` uses instead of linking every test to real manifest writes. `messageLogger` is the seam the host loop uses so a test could substitute `Log` (today's host tests still use a real `logger.Logger`).
- `cmd/devlog/main.go` registers handlers in an unexported `commands` map. Adding a command means a `cmd_*.go` file plus a map entry.
- `internal/e2e` is a separate package that builds and execs the `devlog` binary. It does not import CLI internals.

**Barrel Files:**

None. Go has no `index` re-export files. The extension has no `index.js` barrel. `background.js`, `content_script.js`, and `page_inject.js` are loaded by the browser manifests. `browser-extension/chrome/` and `browser-extension/firefox/` hold packaged copies of those scripts. Tests read the root scripts with `readFileSync`, not a package export (`browser-extension/test/background.test.js`).

---

*Convention analysis: 2026-10-01*
