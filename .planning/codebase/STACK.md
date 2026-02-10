# Technology Stack

**Analysis Date:** 2026-02-10

## Languages

**Primary:**
- Go 1.25.6 - Core CLI application, native messaging host, and tmux integration

**Secondary:**
- JavaScript - Browser extension components (Chrome/Firefox)

## Runtime

**Environment:**
- Native Go runtime
- Cross-platform support (macOS, Windows, Linux)

**Package Manager:**
- Go modules (go.mod)
- Lockfile: go.sum present

## Frameworks

**Core:**
- No external web frameworks - CLI-only application
- Standard library for all core functionality

**Testing:**
- Go testing framework built-in
- Table-driven test patterns throughout

**Build/Dev:**
- Just build automation (justfile)
- Goreleaser for cross-platform binary distribution
- Go fmt for code formatting

## Key Dependencies

**Critical:**
- gopkg.in/yaml.v3 v3.0.1 - YAML configuration parsing
- tmux integration via exec.Command
- Native messaging protocol implementation

**Infrastructure:**
- No external databases or storage services
- Local file system for logs and configuration
- Tmux for session management

## Configuration

**Environment:**
- YAML configuration files (devlog.yml)
- Environment variable interpolation supported ($VAR, ${VAR})
- Platform-specific native messaging directories

**Build:**
- go.mod for dependency management
- .goreleaser.yml for release configuration
- Platform-specific build targets

## Platform Requirements

**Development:**
- Go 1.25+ compiler
- Tmux installed on system
- Chrome or Firefox browser for extension support

**Production:**
- Static binaries for Linux, macOS, Windows
- No external runtime dependencies beyond OS and tmux
- Browser extension requires manual installation

---

*Stack analysis: 2026-02-10*
