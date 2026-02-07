# CLAUDE.md

## 🤖 Welcome, Claude

This is **go-pre-commit**: a lightning-fast, **pure Go** git pre-commit framework that replaces Python-based alternatives with a single binary. Zero runtime dependencies, automatic tool installation, parallel execution, and environment-based configuration make it ideal for Go projects.

### 🎯 Project Overview

**What it does:**
- Provides git pre-commit hooks as a single Go binary
- Runs checks in parallel: `eof`, `fumpt`, `gitleaks`, `lint`, `mod-tidy`, `whitespace`
- **Pure Go implementation** - all checks run directly without Make dependencies
- **Auto-installs tools** - fumpt, golangci-lint, goimports, and gitleaks are installed automatically when needed
- **Plugin system** - Extend with custom checks in any language (Shell, Python, Go, Docker, etc.)
- Configures via environment files: `.github/env/` (modular, preferred) or `.github/.env.base` + `.github/.env.custom` (legacy fallback)

**Key commands:**
- `go-pre-commit install` - Install hooks in repository
- `go-pre-commit run` - Run checks on staged files
- `go-pre-commit run --color=never` - Run without color output (CI-friendly)
- `go-pre-commit run --no-color` - Same as --color=never
- `go-pre-commit status` - Show configuration and installation status
- `go-pre-commit upgrade` - Upgrade to the latest version
- `go-pre-commit uninstall` - Remove installed hooks

**Available checks:**
- `eof`, `fumpt`, `gitleaks`, `lint`, `mod-tidy`, `whitespace`

### 📚 Documentation Hierarchy

**`AGENTS.md`** remains the single source of truth for:
* Coding conventions (naming, formatting, commenting, testing)
* Contribution workflows (branch prefixes, commit message style, PR templates)
* Release, CI, and dependency‑management policies
* Security reporting and governance links

> **TL;DR:** **Read `AGENTS.md` first.**
> All technical or procedural questions are answered there.

### 🏗️ Project Structure

```
/cmd/go-pre-commit/     # CLI commands (Cobra framework)
  ├── cmd/             # Command implementations
  └── main.go          # Entry point
/internal/             # Core packages
  ├── checks/          # Check implementations
  │   ├── builtin/     # Built-in checks (whitespace, eof)
  │   └── gotools/     # Go tool checks (fumpt, gitleaks, lint, mod-tidy)
  ├── config/          # Configuration loader (modular + legacy)
  ├── envfile/         # Environment file parsing (Load, Overload, LoadDir)
  ├── errors/          # Sentinel errors and error types
  ├── git/             # Git operations and hook management
  ├── output/          # Formatted output and color control
  ├── plugins/         # Plugin system (registry, JSON protocol)
  ├── runner/          # Parallel check execution
  ├── shared/          # Shared context for check coordination
  ├── tools/           # Tool auto-installation (fumpt, golangci-lint, goimports, gitleaks)
  ├── validation/      # Input validation
  └── version/         # Version management
/examples/             # Plugin examples
  ├── shell-plugin/    # Shell script plugin example
  ├── python-plugin/   # Python plugin example
  ├── go-plugin/       # Go binary plugin example
  ├── docker-plugin/   # Docker-based plugin example
  └── composite-plugin/ # Multi-step composite plugin example
/docs/                 # Documentation
```

### ⚙️ Technical Requirements

- **Go version:** 1.24+ (check `go.mod`)
- **Build system:** [Magex](https://github.com/mrz1836/mage-x) - enterprise-grade build automation
- **Dependencies:** Minimal - Cobra, testify, color, go-isatty, yaml.v3
- **Configuration:**
  - **Modular (preferred):** `.github/env/*.env` files loaded in lexicographic order (last wins)
  - **Legacy (fallback):** `.github/.env.base` (defaults) + optional `.github/.env.custom` (overrides)
  - If `.github/env/` exists with >=1 `.env` file, modular mode is used; otherwise falls back to legacy
- **Build targets:** All checks work with pure Go

### 🎨 Color Output Control

`go-pre-commit` provides comprehensive color output control for better CI/CD integration:

**Smart Auto-Detection:**
- Automatically detects CI environments and disables colors
- Respects `NO_COLOR` environment variable (https://no-color.org/)
- Checks for dumb terminals (`TERM=dumb`)
- Uses TTY detection to determine terminal capabilities

**Manual Control:**
```bash
# Command-line flags (highest priority)
go-pre-commit run --color=auto      # Smart detection (default)
go-pre-commit run --color=always    # Force colors
go-pre-commit run --color=never     # Disable colors
go-pre-commit run --no-color        # Same as --color=never

# Environment variables
NO_COLOR=1                          # Disable colors (standard)
GO_PRE_COMMIT_COLOR_OUTPUT=false    # Disable colors (legacy)
TERM=dumb                           # Disable colors (terminal detection)
```

**CI Environment Detection:**
Automatically detected: GitHub Actions, GitLab CI, Jenkins, CircleCI, Travis CI, BuildKite, Drone, TeamCity, Azure DevOps, AppVeyor, AWS CodeBuild, Semaphore

### 🛠️ Magex Build System

This project uses **Magex** for build automation. Magex provides enterprise-grade development tools with a friendly user experience.

**Installation:**
```bash
# Install mage (required for magex)
go install github.com/magefile/mage@latest

# Clone and enter project
git clone https://github.com/mrz1836/go-pre-commit.git
cd go-pre-commit

# Magex is ready to use immediately
magex test
```

**Essential Commands:**
```bash
magex test         # Run all tests
magex lint         # Run linters (golangci-lint)
magex format       # Format code (fumpt, goimports)
magex build        # Build the binary
magex install      # Install binary to $GOPATH/bin
magex tidy         # Clean and update dependencies
magex -l           # List all available commands (260+ total)
```

**Development Workflow:**
```bash
# Start developing
magex deps:download    # Download dependencies
magex build            # Initial build

# Make changes, then run quality checks
magex format && magex lint && magex test

# Advanced commands
magex test:race       # Run tests with race detection
magex test:cover      # Generate coverage report
magex audit:report    # Security and dependency audit
```

### ✅ Quick Checklist for Claude

1. **Study `AGENTS.md`**
   Make sure every automated change or suggestion respects those rules.

2. **Understand the pure Go check system**
   - All checks in `gotools/` execute tools directly
   - Tools are auto-installed when not found
   - All checks operate directly on files
   - No external dependencies required

3. **Follow branch‑prefix and commit‑message standards**
   They drive auto‑labeling and CI gates.

4. **Never tag releases**
   Only repository code‑owners run `magex version` / `magex release`.

5. **Pass all checks before PR**
   ```bash
   magex test         # Run tests with testify
   magex lint         # golangci-lint
   magex format       # Format with fumpt, goimports
   magex tidy         # Clean dependencies
   ```

6. **Environment configuration**
   All settings controlled via `GO_PRE_COMMIT_*` variables in `.github/env/` (modular, preferred) or `.github/.env.base` + `.github/.env.custom` (legacy fallback)

   **Color output priority:**
   1. Command-line flags (`--color`, `--no-color`)
   2. `NO_COLOR` environment variable
   3. `GO_PRE_COMMIT_COLOR_OUTPUT` setting
   4. `TERM=dumb` terminal detection
   5. CI environment detection
   6. Terminal/TTY detection

### 🧪 Testing Notes

- Tests use **testify** exclusively (no bare `testing` package)
- Run `magex test` for fast tests, `magex test:race` for race detection
- Mock external dependencies for deterministic tests
- Check implementations have comprehensive test coverage

### 🚀 CI/CD Integration

- GitHub Actions workflows in `.github/workflows/`
- CI configuration via `.github/env/` (modular, preferred) or `.github/.env.base` + `.github/.env.custom` (legacy fallback)
- GoReleaser handles releases (`.goreleaser.yml`)
- Pre-commit checks run automatically in CI

### 🎯 Pure Go Benefits

- **Zero external dependency** - Works on any system with Go installed
- **Automatic tool installation** - No manual setup required
- **Faster execution** - No Make overhead
- **Better portability** - Works in more environments
- **Flexible** - Works with any build system or none at all
- **Plugin system** - Extend with custom checks in any language

### 🔌 Plugin System Notes

- **Plugin directory**: `.pre-commit-plugins/` by default
- **Manifest files**: `plugin.yaml`, `plugin.yml`, or `plugin.json` define plugin metadata
- **Communication**: JSON over stdin/stdout protocol
- **Examples**: See `/examples/` for Shell, Python, Go, Docker, and Composite plugins
- **CLI commands**: `go-pre-commit plugin list/add/remove/validate/info`
- **Configuration**: Enable with `GO_PRE_COMMIT_ENABLE_PLUGINS=true` in `.github/env/` or `.github/.env.custom`

If you encounter conflicting guidance elsewhere, `AGENTS.md` wins.
Questions or ambiguities? Open a discussion or ping @mrz1836 instead of guessing.

Happy hacking!
