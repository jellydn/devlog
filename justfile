# Justfile for devlog project
# https://github.com/casey/just

# Default recipe - show available commands
default:
    @just --list

# Build the binary
build:
    go build -o devlog ./cmd/devlog

# Run the CLI (pass args after --)
run *args:
    go run ./cmd/devlog {{args}}

# Install locally
install:
    go install ./cmd/devlog

# Clean build artifacts
clean:
    rm -f devlog

# Run all tests
test:
    go test ./...

# Run tests with verbose output
test-v:
    go test -v ./...

# Run a specific test (e.g., just test-one TestLoad_ValidConfig)
test-one name:
    go test -run {{name}} ./...

# Run tests with coverage
test-cover:
    go test -cover ./...

# Run tests with race detection
test-race:
    go test -race ./...

# Format code
fmt:
    go fmt ./...

# Vet code for issues
vet:
    go vet ./...

# Run all linting (fmt + vet)
lint: fmt vet

# Check if code compiles without building
check:
    go build -o /dev/null ./cmd/devlog

# Run full CI checks (lint + test)
ci: lint test

# Build and symlink to ~/.local/bin for easy testing
devlog-dev:
    go build -o devlog ./cmd/devlog
    go build -o devlog-host ./cmd/devlog-host
    mkdir -p ~/.local/bin
    ln -sf {{justfile_directory()}}/devlog ~/.local/bin/devlog
    ln -sf {{justfile_directory()}}/devlog-host ~/.local/bin/devlog-host
    @echo "Symlinked devlog -> ~/.local/bin/devlog"

# Create devlog.yml from example
init:
    cp devlog.yml.example devlog.yml
