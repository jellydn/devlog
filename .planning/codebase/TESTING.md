# Testing Patterns

**Analysis Date:** 2026-02-23

## Test Framework

**Runner:**
- Go standard `testing` package (no third-party test framework)
- Go 1.25+ (`go.mod`)

**Assertion Library:**
- Standard library only — manual `if` checks with `t.Errorf` / `t.Fatalf`
- No assertion library (no testify, no gomega)

**Run Commands:**
```bash
go test ./...                          # Run all tests
go test -v ./...                       # Verbose output
just test-one TestLoad_ValidConfig     # Single test by name
go test -cover ./...                   # Coverage report
go test -race ./...                    # Race detection
go test -tags=integration ./internal/tmux/  # Integration tests
go test -short ./...                   # Skip integration tests
just ci                                # Format + vet + all tests
```

## Test File Organization

**Location:**
- Co-located with source files (same directory, same package)
- All tests use package-internal access (no `_test` package suffix)

**Naming:**
- Test files: `*_test.go` (`config_test.go`, `tmux_test.go`, `logger_test.go`, `natmsg_test.go`)
- Integration tests: `integration_test.go` with build tag
- Test functions: `TestFunctionName_Description` pattern (e.g., `TestLoad_ValidConfig`, `TestLoad_MissingRequiredFields`, `TestRunner_CreateSession_AlreadyExists`)

**Files (9 total):**
- `internal/config/config_test.go` — 18 test functions
- `internal/tmux/tmux_test.go` — 8 test functions
- `internal/tmux/integration_test.go` — 12 integration test functions
- `internal/natmsg/natmsg_test.go` — 17 test functions
- `internal/natmsg/manifest_test.go` — 7 test functions
- `internal/logger/logger_test.go` — 10 test functions
- `cmd/devlog/init_test.go` — 5 test functions
- `cmd/devlog/status_test.go` — 4 test functions
- `cmd/devlog/healthcheck_test.go` — 5 test functions

## Test Structure

**Suite Organization:**
```go
// Single test case — from internal/config/config_test.go
func TestLoad_ValidConfig(t *testing.T) {
	content := `
version: "1.0"
project: myapp
...
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "devlog.yml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Version != "1.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1.0")
	}
}
```

**Table-driven tests — from `internal/config/config_test.go`:**
```go
func TestLoad_MissingRequiredFields(t *testing.T) {
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
		{
			name:    "missing project",
			content: `...`,
			wantErr: "project is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "devlog.yml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

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
}
```

## Mocking

**Framework:** None — uses manual dependency injection

**Patterns:**
```go
// Constructor with injectable I/O streams — from internal/natmsg/natmsg.go
func NewHostWithStreams(reader io.Reader, writer io.Writer) *Host {
	return &Host{
		reader: bufio.NewReader(reader),
		writer: writer,
	}
}

// Used in tests — from internal/natmsg/natmsg_test.go
func TestHost_ReadMessage_Success(t *testing.T) {
	data := encodeMessage(t, inputMsg)
	reader := bytes.NewReader(data)
	host := NewHostWithStreams(reader, &bytes.Buffer{})

	msg, err := host.ReadMessage()
	// ...
}

func TestHost_WriteResponse(t *testing.T) {
	var output bytes.Buffer
	host := NewHostWithStreams(&bytes.Buffer{}, &output)

	response := Response{Success: true}
	if err := host.WriteResponse(response); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read from output buffer to verify...
}
```

No mock libraries. External dependencies (tmux binary) are tested via:
- `t.Skip("tmux not available in PATH")` when tmux is missing
- Real tmux sessions with unique names for isolation

## Fixtures and Factories

**Test Data:**
```go
// Inline YAML strings for config tests — from internal/config/config_test.go
content := `
version: "1.0"
project: myapp
tmux:
  session: dev
  windows:
    - name: server
      panes:
        - cmd: npm run dev
          log: server.log
`

// Helper function for encoding — from internal/natmsg/natmsg_test.go
func encodeMessage(t *testing.T, msg interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}
	lengthBytes := make([]byte, 4)
	binary.NativeEndian.PutUint32(lengthBytes, uint32(len(data)))
	return append(lengthBytes, data...)
}

// Helper for time parsing — from internal/natmsg/natmsg_test.go
func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatalf("failed to parse time: %v", err)
	}
	return ts
}

// Unique session name generator — from internal/tmux/integration_test.go
func generateTestSessionName() string {
	return fmt.Sprintf("devlog-test-%d", time.Now().UnixNano())
}
```

**Temporary Files:**
- Always use `t.TempDir()` for temp directories (auto-cleaned)
- Write test config files via `os.WriteFile` into temp dirs
- Environment variables set/unset with `defer os.Unsetenv()`

## Coverage

**Requirements:** None enforced — no minimum threshold configured

**Command:** `go test -cover ./...` or `just test-cover`

## Test Types

**Unit Tests:**
- Majority of tests are unit tests
- Test individual functions/methods in isolation
- Located in `*_test.go` alongside source
- Use `t.TempDir()` for filesystem tests, `bytes.Buffer` for I/O tests
- Config parsing, validation, env var interpolation, log formatting, message encoding/decoding

**Integration Tests:**
- Located in `internal/tmux/integration_test.go`
- Gated with `//go:build integration` build tag
- Require real tmux binary in PATH
- Skip with `t.Skip()` if tmux unavailable or `-short` flag used:
```go
func skipIfNoTmux(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available in PATH")
	}
}
```
- Create real tmux sessions with unique names for isolation
- Use `defer` cleanup to kill sessions after tests
- Test full lifecycle: create → verify → kill → verify gone

**E2E Tests:**
- Not used (no browser automation or full-system tests)

## Common Patterns

**Assertion Style:**
```go
// Fatal for setup failures
if err != nil {
    t.Fatalf("Load() failed: %v", err)
}

// Error for assertion failures (allows multiple to report)
if cfg.Version != "1.0" {
    t.Errorf("Version = %q, want %q", cfg.Version, "1.0")
}

// Fatal with descriptive message for unexpected nil errors
if err == nil {
    t.Fatal("Validate() expected error for negative max_runs, got nil")
}
```

**Error Testing:**
```go
// Check error contains expected substring
_, err := Load(configPath)
if err == nil {
    t.Errorf("Load() expected error containing %q, got nil", tt.wantErr)
    return
}
if !strings.Contains(err.Error(), tt.wantErr) {
    t.Errorf("Load() error = %q, want containing %q", err.Error(), tt.wantErr)
}
```

**Environment Variable Testing:**
```go
// Set env vars with deferred cleanup — from internal/config/config_test.go
os.Setenv("TEST_PORT", "3000")
defer os.Unsetenv("TEST_PORT")

// Override HOME for path tests — from internal/natmsg/manifest_test.go
home := os.Getenv("HOME")
defer os.Setenv("HOME", home)
os.Setenv("HOME", tmpDir)
```

**Tmux Tests Availability Guard:**
```go
// Skip pattern used in both unit and integration tests
if _, err := exec.LookPath("tmux"); err != nil {
    t.Skip("tmux not available in PATH")
}
```

**Deferred Cleanup Pattern:**
```go
// Session cleanup in integration tests
defer func() {
    if runner.SessionExists() {
        runner.KillSession()
    }
}()
```

**Timing in Integration Tests:**
```go
// Wait for async operations (tmux commands, log file writes)
time.Sleep(100 * time.Millisecond)  // Short wait for file creation
time.Sleep(300 * time.Millisecond)  // Longer wait for command output
```

---
*Testing analysis: 2026-02-23*
