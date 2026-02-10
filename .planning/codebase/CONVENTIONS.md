# Code Conventions

**Analysis Date:** 2026-02-10

## Code Style

**Formatting:**
- `gofmt` for all Go code
- Tabs for indentation (Go standard)
- 100 character max line length target
- No trailing whitespace

**Imports:**
- Standard library first, then third-party
- No blank lines between import groups
- Use full import paths (no relative imports)

**Comments:**
- `// ` for comments (space after slashes)
- Exported functions/types get comments starting with function name
- Package comments at top of file
- Self-documenting code preferred over excessive comments

## Naming Conventions

**Go Code:**
- **Packages:** lowercase, no underscores (`config`, `tmux`, `logger`)
- **Types:** PascalCase (`Runner`, `Config`, `Message`)
- **Functions:** PascalCase (`NewRunner`, `CreateSession`, `Load`)
- **Variables:** camelCase (`logsDir`, `configPath`, `hostPath`)
- **Constants:** PascalCase for exported, camelCase for unexported
- **Interfaces:** Rarely used, concrete structs preferred
- **Acronyms:** Preserved case (`URL` not `Url`, `TMUX` not `Tmux`)

## Error Handling Patterns

**Error Wrapping:**
```go
return fmt.Errorf("failed to create tmux session: %w", err)
```
- Error messages start with lowercase
- Use `%w` for error wrapping (Go 1.13+)
- Context added at call site, not in library code

**Guard Clauses:**
```go
if err != nil {
    return fmt.Errorf("failed to open file: %w", err)
}
// Continue with happy path
```

**Error Checking:**
- Always check errors from function calls
- Use `t.Fatalf` for test setup errors
- Use `t.Errorf` for test assertions
- Don't ignore errors intentionally (no `_ = err`)

## Function Design

**Constructor Pattern:**
```go
func NewRunner(sessionName string) *Runner {
    return &Runner{sessionName: sessionName}
}
```

**Configuration Functions:**
```go
func New(logPath string, levels []string) (*Logger, error) {
    // Create directory if needed
    // Open file
    // Return populated struct
}
```

**Method Conventions:**
- Pointer receivers for methods that modify the struct
- Value receivers for methods that don't modify (rare in this codebase)
- Methods return errors, don't log in library code

## Testing Conventions

**Test Structure:**
- Table-driven tests preferred
- Test name: `TestFunctionName_Description` (`TestLoad_ValidConfig`)
- `t.TempDir()` for temporary directories
- `t.Helper()` for test helper functions

**Test Organization:**
```go
func TestFeature_Scenario(t *testing.T) {
    // Arrange
    tmpDir := t.TempDir()

    // Act
    result, err := DoSomething(tmpDir)

    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != expected {
        t.Errorf("got %q, want %q", result, expected)
    }
}
```

## Type System

**Struct Tags:**
- YAML: `yaml:"field_name"` for config parsing
- JSON: `json:"fieldName"` for JSON marshaling
- `omitempty` for optional fields

**Interface Usage:**
- `interface{}` used sparingly for flexible JSON fields (timestamp, line, column)
- Type assertions with `ok` pattern: `if n, ok := t.Int64(); ok { ... }`

**Type Conversions:**
- Helper functions for numeric type conversions
- `toInt64()` consolidates numeric type handling
- `formatTimestamp()` handles multiple timestamp formats

## Concurrency

**Mutex Usage:**
- `sync.Mutex` for protecting shared state
- `defer mu.Unlock()` pattern for lock release
- Lock held for minimal duration

**Goroutines:**
- None used in current codebase
- All operations are synchronous

## Shell Command Execution

**Command Pattern:**
```go
cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
if err := cmd.Run(); err != nil {
    return fmt.Errorf("failed to create session: %w", err)
}
```

**Security:**
- Always quote shell arguments to prevent injection
- Use `strings.ReplaceAll(s, "'", "'\\''")` for shell escaping
- Store absolute paths to avoid path resolution issues

## Configuration Management

**YAML Structure:**
- `version:` field for format versioning
- Environment variable interpolation: `$VAR` or `${VAR}`
- Required fields validated on load
- Defaults provided where appropriate

**Run Modes:**
- `timestamped`: Create new directory with timestamp for each run
- `overwrite`: Reuse same log directory

---

*Conventions analysis: 2026-02-10*
