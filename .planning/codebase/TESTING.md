# Testing Strategy

**Analysis Date:** 2026-02-10

## Framework

**Go Testing:**
- Built-in `testing` package
- Table-driven tests for multiple scenarios
- No external testing frameworks

**Browser Extension:**
- Manual testing required
- No automated test framework for extension JavaScript

## Test Structure

**Organization:**
```
internal/
├── config/
│   ├── config.go
│   └── config_test.go       # Config parsing tests
├── logger/
│   ├── logger.go
│   └── logger_test.go        # Logger and type conversion tests
├── natmsg/
│   ├── natmsg.go
│   ├── manifest.go
│   └── natmsg_test.go       # Protocol and manifest tests
└── tmux/
    ├── tmux.go
    └── tmux_test.go         # Tmux operations tests
```

**Test Patterns:**
```go
func TestFunctionName_Description(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    WantType
        wantErr bool
    }{
        {name: "valid case", input: valid, want: expected, wantErr: false},
        {name: "error case", input: invalid, want: nil, wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionUnderTest(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Test Coverage

**Covered Areas:**
- Config parsing with environment variables
- Logger creation, level filtering, and log writing
- Native messaging protocol (read/write)
- Type conversion for timestamps and line numbers
- Tmux session management (create, kill, status)

**Not Covered (Integration):**
- Actual tmux session operations (require tmux running)
- Browser extension communication (requires browser context)
- Native messaging host integration (requires extension setup)

**Coverage Commands:**
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/logger/...

# Run with race detection
go test -race ./...
```

## Test Utilities

**Temporary Directories:**
```go
tmpDir := t.TempDir()  // Automatically cleaned up
```

**Helper Functions:**
```go
func encodeMessage(t *testing.T, msg interface{}) []byte {
    t.Helper()
    data, err := json.Marshal(msg)
    if err != nil {
        t.Fatalf("failed to marshal message: %v", err)
    }
    // ... length prefix and return
}
```

## Mocking & Isolation

**Current Approach:**
- No mocking framework used
- Tests use real file I/O (with temp directories)
- `bytes.Buffer` for testing stream operations

**Example Mock Pattern:**
```go
// Instead of real stdout, use buffer
var output bytes.Buffer
host := NewHostWithStreams(&bytes.Buffer{}, &output)
```

## Browser Extension Testing

**Manual Testing Required:**
1. Install extension in Chrome/Firefox
2. Run `devlog up` to start session
3. Enable browser logging in config
4. Open browser console and log messages
5. Verify logs appear in configured log file

**Test Scenarios:**
- Console log capture (log, error, warn, info)
- Stack trace parsing
- URL and source file attribution
- Level filtering
- Multiple tabs/windows

## Integration Testing

**Manual Integration Flow:**
```bash
# 1. Start devlog session
devlog up

# 2. Check session status
devlog status

# 3. List log runs
devlog ls

# 4. Open logs directory
devlog open

# 5. Stop session
devlog down
```

**Browser Integration:**
```bash
# 1. Register native messaging host
devlog register --chrome --extension-id <extension-id>

# 2. Start devlog with browser logging configured
devlog up

# 3. Open browser, navigate to configured URL
# 4. Check logs include browser console output
```

## Test Data

**Fixtures:**
- Example configurations in `devlog.yml.example`
- Test messages use realistic values
- Timestamps use expected formats (RFC3339, Unix milliseconds)

## Known Testing Gaps

1. **No E2E tests** - Full flow requires tmux and browser
2. **No extension automation** - Requires browser driver setup
3. **No concurrent access tests** - Native messaging not stress-tested
4. **No platform-specific tests** - macOS/Linux/Windows differences not tested

---

*Testing analysis: 2026-02-10*
