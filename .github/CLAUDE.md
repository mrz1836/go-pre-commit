# CLAUDE.md

## 🤖 Welcome, Claude

This is **go-pre-commit**: a lightning-fast, Go-native git pre-commit framework that replaces Python-based alternatives with a single binary. Zero runtime dependencies, parallel execution, and environment-based configuration make it ideal for Go projects.

### 🎯 Project Overview

**What it does:**
- Provides git pre-commit hooks as a single Go binary
- Runs checks in parallel: `fumpt`, `lint`, `mod-tidy`, `whitespace`, `eof`
- Configures via `.github/.env.shared` (no YAML files)
- Integrates seamlessly with existing Makefile targets

**Key commands:**
- `go-pre-commit install` - Install hooks in repository
- `go-pre-commit run` - Run checks on staged files
- `go-pre-commit status` - Show configuration and installation status
- `go-pre-commit upgrade` - Upgrade to the latest version
- `go-pre-commit uninstall` - Remove installed hooks

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
  │   └── makewrap/    # Make-based checks (fumpt, lint, mod-tidy)
  ├── config/          # Configuration loader
  ├── git/             # Git operations and hook management
  ├── runner/          # Parallel check execution
  └── output/          # Formatted output
```

### ⚙️ Technical Requirements

- **Go version:** 1.24+ (check `go.mod`)
- **Dependencies:** Minimal - Cobra, testify, color, godotenv
- **Configuration:** All settings in `.github/.env.shared`
- **Make targets required:** `fumpt`, `lint`, `mod-tidy`

### ✅ Quick Checklist for Claude

1. **Study `AGENTS.md`**
   Make sure every automated change or suggestion respects those rules.

2. **Understand the check system**
   - Checks implement the `Check` interface in `internal/checks/`
   - Make-based checks wrap existing Makefile targets
   - Built-in checks operate directly on files

3. **Follow branch‑prefix and commit‑message standards**
   They drive auto‑labeling and CI gates.

4. **Never tag releases**
   Only repository code‑owners run `make tag` / `make release`.

5. **Pass all checks before PR**
   ```bash
   make test          # Run tests with testify
   make lint          # golangci-lint
   make fumpt         # Format with gofumpt
   make mod-tidy      # Clean dependencies
   ```

6. **Environment configuration**
   All settings controlled via `GO_PRE_COMMIT_*` variables in `.github/.env.shared`

### 🧪 Testing Notes

- Tests use **testify** exclusively (no bare `testing` package)
- Run `make test` for fast tests, `make test-race` for race detection
- Mock external dependencies for deterministic tests
- Check implementations have comprehensive test coverage

### 🚀 CI/CD Integration

- GitHub Actions workflows in `.github/workflows/`
- All CI configuration via `.github/.env.shared`
- GoReleaser handles releases (`.goreleaser.yml`)
- Pre-commit checks run automatically in CI

If you encounter conflicting guidance elsewhere, `AGENTS.md` wins.
Questions or ambiguities? Open a discussion or ping @mrz1836 instead of guessing.

Happy hacking!
