<div align="center">

# 🔒&nbsp;&nbsp;go-pre-commit

**Lightning-fast, Git pre-commit hooks for Go projects - built in pure Go**

<br/>

<a href="https://github.com/mrz1836/go-pre-commit/releases"><img src="https://img.shields.io/github/release-pre/mrz1836/go-pre-commit?include_prereleases&style=flat-square&logo=github&color=black" alt="Release"></a>
<a href="https://golang.org/"><img src="https://img.shields.io/github/go-mod/go-version/mrz1836/go-pre-commit?style=flat-square&logo=go&color=00ADD8" alt="Go Version"></a>
<a href="https://github.com/mrz1836/go-pre-commit/blob/master/LICENSE"><img src="https://img.shields.io/github/license/mrz1836/go-pre-commit?style=flat-square&color=blue" alt="License"></a>

<br/>

<table align="center" border="0">
  <tr>
    <td align="right">
       <code>CI / CD</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://github.com/mrz1836/go-pre-commit/actions"><img src="https://img.shields.io/github/actions/workflow/status/mrz1836/go-pre-commit/fortress.yml?branch=master&label=build&logo=github&style=flat-square" alt="Build"></a>
       <a href="https://github.com/mrz1836/go-pre-commit/commits/master"><img src="https://img.shields.io/github/last-commit/mrz1836/go-pre-commit?style=flat-square&logo=git&logoColor=white&label=last%20update" alt="Last Commit"></a>
    </td>
    <td align="right">
       &nbsp;&nbsp;&nbsp;&nbsp; <code>Quality</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://goreportcard.com/report/github.com/mrz1836/go-pre-commit"><img src="https://goreportcard.com/badge/github.com/mrz1836/go-pre-commit?style=flat-square" alt="Go Report"></a>
       <a href="https://mrz1836.github.io/go-pre-commit/" target="_blank"><img src="https://mrz1836.github.io/go-pre-commit/coverage-flat-square.svg" alt="Code Coverage"></a>
    </td>
  </tr>

  <tr>
    <td align="right">
       <code>Security</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://scorecard.dev/viewer/?uri=github.com/mrz1836/go-pre-commit"><img src="https://api.scorecard.dev/projects/github.com/mrz1836/go-pre-commit/badge?style=flat-square" alt="Scorecard"></a>
       <a href=".github/SECURITY.md"><img src="https://img.shields.io/badge/policy-active-success?style=flat-square&logo=security&logoColor=white" alt="Security"></a>
    </td>
    <td align="right">
       &nbsp;&nbsp;&nbsp;&nbsp; <code>Community</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://github.com/mrz1836/go-pre-commit/graphs/contributors"><img src="https://img.shields.io/github/contributors/mrz1836/go-pre-commit?style=flat-square&color=orange" alt="Contributors"></a>
       <a href="https://mrz1818.com/?tab=tips&utm_source=github&utm_medium=sponsor-link&utm_campaign=go-pre-commit&utm_term=go-pre-commit&utm_content=go-pre-commit"><img src="https://img.shields.io/badge/donate-bitcoin-ff9900?style=flat-square&logo=bitcoin" alt="Bitcoin"></a>
    </td>
  </tr>
</table>

</div>

<br/>
<br/>

<div align="center">

### <code>Project Navigation</code>

</div>

<table align="center">
  <tr>
    <td align="center" width="33%">
       🚀&nbsp;<a href="#-quickstart"><code>Quickstart</code></a>
    </td>
    <td align="center" width="33%">
       ⚙️&nbsp;<a href="#-configuration"><code>Configuration</code></a>
    </td>
    <td align="center" width="33%">
       🎯&nbsp;<a href="#-available-checks"><code>Available&nbsp;Checks</code></a>
    </td>
  </tr>
  <tr>
    <td align="center">
       🔌&nbsp;<a href="#-plugin-system"><code>Plugin&nbsp;System</code></a>
    </td>
    <td align="center">
       🧪&nbsp;<a href="#-examples--tests"><code>Examples&nbsp;&&nbsp;Tests</code></a>
    </td>
    <td align="center">
       🤝&nbsp;<a href="#-contributing"><code>Contributing</code></a>
    </td>
  </tr>
</table>

<br/>

## 🚀 Quickstart

Get up and running with `go-pre-commit` in 30 seconds:

### Install the binary

```bash
# Install from source (requires Go 1.24+)
go install github.com/mrz1836/go-pre-commit/cmd/go-pre-commit@latest

# Upgrade to the latest version
go-pre-commit upgrade --force
```

<br>

### Install [MAGE-X](https://github.com/mrz1836/mage-x) build tool

```bash
# Install MAGE-X for development and building
go install github.com/mrz1836/mage-x/cmd/magex@latest
magex update:install
```

<br>

### Set up in your project

```bash
# Navigate to your Go project
cd your-go-project

# Install the pre-commit hook
go-pre-commit install

# That's it! Your pre-commit hooks are now active!
```

<br>

### Test it out

```bash
# Make a change and commit
echo "test" >> test.go
git add test.go
git commit -m "Test commit"

# The pre-commit system will automatically run checks:
# ✓ Ensuring files end with newline
# ✓ Running fumpt formatter
# ✓ Running linter (golangci-lint)
# ✓ Running go mod tidy
# ✓ Fixing trailing whitespace
```

> **Good to know:** `go-pre-commit` is a pure Go solution with minimal external dependencies.
> It's a single Go binary that replaces Python-based pre-commit frameworks.
> All formatting and linting tools are automatically installed when needed!

<br/>

## ⚙️ Configuration

<details>
<summary><strong><code>Environment Configuration</code></strong></summary>
<br/>

`go-pre-commit` uses environment variables for configuration, loaded from `.github/env/` (modular, preferred) or `.github/.env.base` + `.github/.env.custom` (legacy fallback):

```bash
# Core settings
ENABLE_GO_PRE_COMMIT=true              # Enable/disable the system
GO_PRE_COMMIT_FAIL_FAST=false          # Stop on first failure
GO_PRE_COMMIT_TIMEOUT_SECONDS=720      # Overall timeout
GO_PRE_COMMIT_PARALLEL_WORKERS=2       # Number of parallel workers

# Individual checks
GO_PRE_COMMIT_ENABLE_EOF=true           # Ensure files end with newline
GO_PRE_COMMIT_ENABLE_FUMPT=true         # Format with fumpt
GO_PRE_COMMIT_ENABLE_GITLEAKS=false     # Scan for secrets and credentials
GO_PRE_COMMIT_ENABLE_LINT=true          # Run golangci-lint
GO_PRE_COMMIT_ENABLE_MOD_TIDY=true      # Run go mod tidy
GO_PRE_COMMIT_ENABLE_WHITESPACE=true    # Fix trailing whitespace

# Auto-staging (automatically stage fixed files)
GO_PRE_COMMIT_EOF_AUTO_STAGE=true
GO_PRE_COMMIT_FUMPT_AUTO_STAGE=true
GO_PRE_COMMIT_WHITESPACE_AUTO_STAGE=true

# Color output settings (auto-detected by default)
GO_PRE_COMMIT_COLOR_OUTPUT=true             # Enable/disable color output
NO_COLOR=                                   # Set to any value to disable colors (follows standard)
```

**Configuration System (auto-detected):**
- **Modular (preferred):** `.github/env/*.env` files loaded in lexicographic order (last wins)
- **Legacy (fallback):** `.github/.env.base` (defaults) + optional `.github/.env.custom` (overrides)
- If `.github/env/` exists with >=1 `.env` file, modular mode is used; otherwise falls back to legacy

**Color Output:**
- Colors are auto-detected based on terminal capabilities and environment
- Automatically disabled in CI environments (GitHub Actions, GitLab CI, Jenkins, etc.)
- Respects standard `NO_COLOR` environment variable
- Can be controlled via `--color` flag or `GO_PRE_COMMIT_COLOR_OUTPUT` setting

</details>

<br/>

## 🔄 Workflow Process

<details>
<summary><strong><code>Installing Hooks</code></strong></summary>
<br/>

```bash
# Install default pre-commit hook
go-pre-commit install

# Install with specific hook types
go-pre-commit install --hook-type pre-commit --hook-type pre-push

# Force reinstall (overwrites existing hooks)
go-pre-commit install --force
```

</details>

<details>
<summary><strong><code>Running Checks Manually</code></strong></summary>
<br/>

```bash
# Run all checks on staged files
go-pre-commit run

# Run specific checks
go-pre-commit run --checks fumpt,lint

# Run on all files (not just staged)
go-pre-commit run --all-files

# Skip specific checks
go-pre-commit run --skip lint,mod-tidy

# Color output control
go-pre-commit run --color=never     # Disable color output
go-pre-commit run --color=always    # Force color output
go-pre-commit run --color=auto      # Auto-detect (default)
go-pre-commit run --no-color        # Same as --color=never
```

</details>

<details>
<summary><strong><code>Status & Updates</code></strong></summary>
<br/>

### Checking status

```bash
# View installation status and configuration
go-pre-commit status

# Shows:
# - Installation status
# - Enabled checks
# - Configuration location
# - Current settings
```

### Updating go-pre-commit

```bash
# Check for available updates
go-pre-commit upgrade --check

# Upgrade to the latest version
go-pre-commit upgrade

# Force upgrade even if already on latest
go-pre-commit upgrade --force

# Upgrade and reinstall hooks
go-pre-commit upgrade --reinstall

# Verify version
go-pre-commit --version
```

### Uninstalling

```bash
# Remove all installed hooks
go-pre-commit uninstall

# Remove specific hook types
go-pre-commit uninstall --hook-type pre-commit
```

</details>

<br/>

## 🎯 Available Checks

<details>
<summary><strong><code>Built-in Checks Reference</code></strong></summary>
<br/>

`go-pre-commit` includes these built-in checks:

| Check            | Description                                        | Auto-fix | Configuration                  |
|------------------|----------------------------------------------------|----------|--------------------------------|
| **eof**          | Ensures files end with a newline                   | ✅        | Auto-stages changes if enabled |
| **fumpt**        | Formats Go code with stricter rules than `gofmt`   | ✅        | Auto-installs if needed        |
| **gitleaks**     | Scans for secrets and credentials in code          | ❌        | Auto-installs if needed        |
| **lint**         | Runs golangci-lint for comprehensive linting       | ❌        | Auto-installs if needed        |
| **mod-tidy**     | Ensures go.mod and go.sum are tidy                 | ✅        | Pure Go - no dependencies      |
| **whitespace**   | Removes trailing whitespace                        | ✅        | Auto-stages changes if enabled |

All checks run in parallel for maximum performance. All checks work out-of-the-box with pure Go! Tools are automatically installed when needed.

</details>

<br/>

## 🔌 Plugin System

`go-pre-commit` supports custom plugins to extend its functionality with checks written in any language.

<details>
<summary><strong><code>Quick Start with Plugins</code></strong></summary>
<br/>

### What are Plugins?

Plugins are external executables (scripts or binaries) that integrate seamlessly with the built-in checks. They can be written in:
- Shell/Bash scripts for simple checks
- Python for complex validation
- Go for high-performance checks
- Any language that can read JSON from stdin and write to stdout

### Quick Setup

**Enable plugins** in your configuration files:
```bash
# Add to .github/env/90-project.env (modular) or .env.custom (legacy)
GO_PRE_COMMIT_ENABLE_PLUGINS=true
GO_PRE_COMMIT_PLUGIN_DIR=.pre-commit-plugins
```

**Install an example plugin**:
```bash
# Copy an example plugin to your project
cp -r examples/shell-plugin .pre-commit-plugins/

# Or use the CLI
go-pre-commit plugin add examples/shell-plugin
```

**Run checks** (plugins run alongside built-in checks):
```bash
go-pre-commit run
```

</details>

<details>
<summary><strong><code>Plugin Management & Creation</code></strong></summary>
<br/>

### Plugin Management Commands

```bash
# List available plugins
go-pre-commit plugin list

# Add a plugin
go-pre-commit plugin add ./my-plugin

# Remove a plugin
go-pre-commit plugin remove my-plugin

# Validate a plugin manifest
go-pre-commit plugin validate ./my-plugin

# Show plugin details
go-pre-commit plugin info my-plugin
```

### Creating Your Own Plugin

Every plugin needs:
1. A manifest file (`plugin.yaml`, `plugin.yml`, or `plugin.json`)
2. An executable script or binary
3. JSON-based communication protocol

**Example `plugin.yaml`:**
```yaml
name: my-custom-check
version: 1.0.0
description: Checks for custom issues in code
executable: ./check.sh
file_patterns:
  - "*.go"
timeout: 30s
category: linting
```

**Example Plugin Script:**
```bash
#!/bin/bash
INPUT=$(cat)
# Process files and output JSON response
echo '{"success": true, "output": "Check passed"}'
```

### Available Example Plugins

| Plugin                                       | Description               | Language |
|----------------------------------------------|---------------------------|----------|
| [todo-checker](examples/shell-plugin)        | Finds TODO/FIXME comments | Shell    |
| [json-validator](examples/python-plugin)     | Validates JSON formatting | Python   |
| [license-header](examples/go-plugin)         | Checks license headers    | Go       |
| [security-scanner](examples/docker-plugin)   | Security scanning         | Docker   |
| [multi-validator](examples/composite-plugin) | Multi-step validation     | Mixed    |

See the [examples directory](examples) for complete plugin implementations and documentation.

### Plugin Features

- **Parallel Execution**: Plugins run in parallel with built-in checks
- **File Filtering**: Process only relevant files using patterns
- **Timeout Protection**: Configurable timeouts prevent hanging
- **Environment Variables**: Pass configuration via environment
- **JSON Protocol**: Structured communication for reliable integration
- **Category Support**: Organize plugins by type (linting, security, etc.)

</details>

<br/>

## 🏗️ Starting a New Project

<details>
<summary><strong><code>Project Setup Guide</code></strong></summary>
<br/>

Setting up `go-pre-commit` in a new Go project:

### 1. Initialize your Go project

```bash
mkdir my-awesome-project
cd my-awesome-project
go mod init github.com/username/my-awesome-project
```

### 2. Create the configuration

```bash
# Create the .github directory
mkdir -p .github

# Download the default configuration
curl -o .github/.env.base https://raw.githubusercontent.com/mrz1836/go-pre-commit/master/.github/.env.base

# Optionally create project-specific overrides
cat > .github/.env.custom << 'EOF'
ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_ENABLE_EOF=true
GO_PRE_COMMIT_ENABLE_FUMPT=true
GO_PRE_COMMIT_ENABLE_GITLEAKS=false
GO_PRE_COMMIT_ENABLE_LINT=true
GO_PRE_COMMIT_ENABLE_MOD_TIDY=true
GO_PRE_COMMIT_ENABLE_WHITESPACE=true
EOF
```

### 3. Install go-pre-commit

```bash
# Install the tool
go install github.com/mrz1836/go-pre-commit/cmd/go-pre-commit@latest

# Set up hooks in your repository
go-pre-commit install

# Verify installation
go-pre-commit status
```

### 4. Test your setup

```bash
# Create a test file with issues
echo -e "package main\n\nfunc main() {  \n}" > main.go

# Try to commit
git add .
git commit -m "Initial commit"

# Watch go-pre-commit automatically fix issues!
```

</details>

<br/>

## 📚 Advanced Documentation

<details>
<summary><strong><code>Repository Features</code></strong></summary>
<br/>

* **Continuous Integration on Autopilot** with [GitHub Actions](https://github.com/features/actions) – every push is built, tested, and reported in minutes.
* **Pull‑Request Flow That Merges Itself** thanks to [auto‑merge](.github/workflows/auto-merge-on-approval.yml) and hands‑free [Dependabot auto‑merge](.github/workflows/dependabot-auto-merge.yml).
* **One‑Command Builds** powered by modern [MAGE-X](https://github.com/mrz1836/mage-x) with 343+ built-in commands for linting, testing, releases, and more.
* **First‑Class Dependency Management** using native [Go Modules](https://github.com/golang/go/wiki/Modules).
* **Uniform Code Style** via [gofumpt](https://github.com/mvdan/gofumpt) plus zero‑noise linting with [golangci‑lint](https://github.com/golangci/golangci-lint).
* **Confidence‑Boosting Tests** with [testify](https://github.com/stretchr/testify), the Go [race detector](https://blog.golang.org/race-detector), crystal‑clear [HTML coverage](https://blog.golang.org/cover) snapshots, and automatic uploads to [Codecov](https://codecov.io/).
* **Hands‑Free Releases** delivered by [GoReleaser](https://github.com/goreleaser/goreleaser) whenever you create a [new Tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging).
* **Relentless Dependency & Vulnerability Scans** via [Dependabot](https://dependabot.com), [Nancy](https://github.com/sonatype-nexus-community/nancy), and [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck).
* **Security Posture by Default** with [CodeQL](https://docs.github.com/en/github/finding-security-vulnerabilities-and-errors-in-your-code/about-code-scanning), [OpenSSF Scorecard](https://openssf.org), and secret‑leak detection via [gitleaks](https://github.com/gitleaks/gitleaks).
* **Automatic Syndication** to [pkg.go.dev](https://pkg.go.dev/) on every release for instant godoc visibility.
* **Polished Community Experience** using rich templates for [Issues & PRs](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository).
* **All the Right Meta Files** (`LICENSE`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SUPPORT.md`, `SECURITY.md`) pre‑filled and ready.
* **Code Ownership** clarified through a [CODEOWNERS](.github/CODEOWNERS) file, keeping reviews fast and focused.
* **Zero‑Noise Dev Environments** with tuned editor settings (`.editorconfig`) plus curated *ignore* files for [VS Code](.editorconfig), [Docker](.dockerignore), and [Git](.gitignore).
* **Label Sync Magic**: your repo labels stay in lock‑step with [.github/labels.yml](.github/labels.yml).
* **Friendly First PR Workflow** – newcomers get a warm welcome thanks to a dedicated [workflow](.github/workflows/pull-request-management.yml).
* **Standards‑Compliant Docs** adhering to the [standard‑readme](https://github.com/RichardLitt/standard-readme/blob/master/spec.md) spec.
* **Instant Cloud Workspaces** via [Gitpod](https://gitpod.io/) – spin up a fully configured dev environment with automatic linting and tests.
* **Out‑of‑the‑Box VS Code Happiness** with a preconfigured [Go](https://code.visualstudio.com/docs/languages/go) workspace and [`.vscode`](.vscode) folder with all the right settings.
* **Optional Release Broadcasts** to your community via [Slack](https://slack.com), [Discord](https://discord.com), or [Twitter](https://twitter.com) – plug in your webhook.
* **AI Playbook** – machine‑readable guidelines in [tech conventions](.github/tech-conventions/ai-compliance.md)
* **DevContainers for Instant Onboarding** – Launch a ready-to-code environment in seconds with [VS Code DevContainers](https://containers.dev/) and the included [.devcontainer.json](.devcontainer.json) config.

</details>

<details>
<summary><strong><code>Library Deployment</code></strong></summary>
<br/>

This project uses [goreleaser](https://github.com/goreleaser/goreleaser) for streamlined binary and library deployment to GitHub. To get started, install it via:

```bash
brew install goreleaser
```

The release process is defined in the [.goreleaser.yml](.goreleaser.yml) configuration file.

Then create and push a new Git tag using:

```bash
magex version:bump bump=patch push=true branch=master
```

This process ensures consistent, repeatable releases with properly versioned artifacts and citation metadata.

</details>

<details>
<summary><strong><code>Build Commands</code></strong></summary>
<br/>

View all build commands

```bash script
magex help
```

</details>

<details>
<summary><strong><code>GitHub Workflows</code></strong></summary>
<br/>


### 🎛️ The Workflow Control Center

All GitHub Actions workflows in this repository are powered by environment configuration files – your one-stop shop for tweaking CI/CD behavior without touching a single YAML file! 🎯

**Configuration (auto-detected):**
- **[.github/env/](.github/env/)** – Modular configuration (preferred) — files load in lexicographic order, last wins
- **[.env.base](.github/.env.base)** – Legacy default configuration (fallback if `.github/env/` doesn't exist)
- **[.env.custom](.github/.env.custom)** – Legacy project-specific overrides

These files control everything from:
- **🚀 Go version matrix** (test on multiple versions or just one)
- **🏃 Runner selection** (Ubuntu or macOS, your wallet decides)
- **🔬 Feature toggles** (coverage, fuzzing, linting, race detection, benchmarks)
- **🛡️ Security tool versions** (gitleaks, nancy, govulncheck)
- **🤖 Auto-merge behaviors** (how aggressive should the bots be?)
- **🏷️ PR management rules** (size labels, auto-assignment, welcome messages)

> **Pro tip:** Want to disable code coverage? Just add `ENABLE_CODE_COVERAGE=false` to `.github/env/90-project.env` (or `.env.custom` in legacy mode) and push. No YAML archaeology required!

<br/>

| Workflow Name                                                                      | Description                                                                                                            |
|------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------|
| [auto-merge-on-approval.yml](.github/workflows/auto-merge-on-approval.yml)         | Automatically merges PRs after approval and all required checks, following strict rules.                               |
| [codeql-analysis.yml](.github/workflows/codeql-analysis.yml)                       | Analyzes code for security vulnerabilities using [GitHub CodeQL](https://codeql.github.com/).                          |
| [dependabot-auto-merge.yml](.github/workflows/dependabot-auto-merge.yml)           | Automatically merges [Dependabot](https://github.com/dependabot) PRs that meet all requirements.                       |
| [fortress.yml](.github/workflows/fortress.yml)                                     | Runs the GoFortress security and testing workflow, including linting, testing, releasing, and vulnerability checks.    |
| [pull-request-management.yml](.github/workflows/pull-request-management.yml)       | Labels PRs by branch prefix, assigns a default user if none is assigned, and welcomes new contributors with a comment. |
| [scorecard.yml](.github/workflows/scorecard.yml)                                   | Runs [OpenSSF](https://openssf.org/) Scorecard to assess supply chain security.                                        |
| [stale.yml](.github/workflows/stale-check.yml)                                     | Warns about (and optionally closes) inactive issues and PRs on a schedule or manual trigger.                           |
| [sync-labels.yml](.github/workflows/sync-labels.yml)                               | Keeps GitHub labels in sync with the declarative manifest at [`.github/labels.yml`](./.github/labels.yml).             |

</details>

<details>
<summary><strong><code>Updating Dependencies</code></strong></summary>
<br/>

To update all dependencies (Go modules, linters, and related tools), run:

```bash
magex deps:update
```

This command ensures all dependencies are brought up to date in a single step, including Go modules and any managed tools. It is the recommended way to keep your development environment and CI in sync with the latest versions.

</details>

<br/>

## 🧪 Examples & Tests

All unit tests and fuzz tests run via [GitHub Actions](https://github.com/mrz1836/go-pre-commit/actions) and use [Go version 1.24.x](https://go.dev/doc/go1.24). View the [configuration file](.github/workflows/fortress.yml).

Run all tests (fast):

```bash script
magex test
```

Run all tests with race detector (slower):
```bash script
magex test:race
```

<details>
<summary><strong><code>Fuzz Tests</code></strong></summary>
<br/>

The project includes comprehensive fuzz tests for security-critical components:

```bash script
magex test:fuzz
```

**Available Fuzz Tests:**
- **[Config Parsing](internal/config/fuzz_test.go)** - Tests environment variable parsing with malformed inputs
- **[Builtin Checks](internal/checks/builtin/fuzz_test.go)** - Tests whitespace/EOF checks with binary and malformed files
- **[Git Operations](internal/git/fuzz_test.go)** - Tests file path parsing and repository operations
- **[Runner Engine](internal/runner/fuzz_test.go)** - Tests check execution with edge case configurations

Fuzz tests help ensure the system handles malformed inputs gracefully without crashes or security vulnerabilities.

</details>

<br/>

## ⚡ Benchmarks

Run the Go [benchmarks](internal/benchmark_test.go):

```bash script
magex bench
```

<br/>

<details>
<summary><strong><code>Benchmark Results</code></strong></summary>

### Benchmark Results

| Benchmark                                                                   | Iterations |     ns/op |    B/op | allocs/op | Description                      |
|-----------------------------------------------------------------------------|------------|----------:|--------:|----------:|----------------------------------|
| [PreCommitSystem_SmallProject](internal/benchmark_test.go)                  | 89,523     |    13,555 |  15,390 |        73 | Small project (3 files)          |
| [PreCommitSystem_EndToEnd](internal/benchmark_test.go)                      | 44,742     |    24,436 |  36,070 |       111 | Full system (8 files)            |
| [PreCommitSystem_LargeProject](internal/benchmark_test.go)                  | 24,704     |    48,146 | 108,986 |       229 | Large project (25+ files)        |
| [Runner_New](internal/runner/runner_bench_test.go)                          | 4,086,028  |       293 |     592 |        10 | Runner creation                  |
| [Runner_SingleCheck](internal/runner/runner_bench_test.go)                  | 187,984    |     6,415 |   7,312 |        33 | Single check execution           |
| [WhitespaceCheck_SmallFile](internal/checks/builtin/builtin_bench_test.go)  | 6,148,348  |       195 |     128 |         2 | Whitespace check (small file)    |
| [WhitespaceCheck_Parallel](internal/checks/builtin/builtin_bench_test.go)   | 14,334,333 |        85 |     128 |         2 | Parallel whitespace processing   |
| [Repository_GetAllFiles](internal/git/git_bench_test.go)                    | 315        | 3,776,237 |  69,746 |       210 | Git file enumeration             |
| [Runner_Performance_SmallCommit](internal/runner/performance_bench_test.go) | 58,266     |    20,899 |  16,990 |       112 | Typical small commit (1-3 files) |

> These benchmarks demonstrate lightning-fast pre-commit processing with minimal memory overhead.
> Performance results measured on Apple M1 Max (ARM64) showing microsecond-level execution times for individual checks and sub-second processing for complete commit workflows.
> The system scales efficiently from small single-file changes to large multi-file commits while maintaining consistent low-latency performance.

</details>

<br/>

## 🛠️ Code Standards
Read more about this Go project's [code standards](.github/CODE_STANDARDS.md).

<br/>

## 🤖 AI Usage & Assistant Guidelines
Read the [AI Usage & Assistant Guidelines](.github/tech-conventions/ai-compliance.md) for details on how AI is used in this project and how to interact with the AI assistants.

<br/>

## 👥 Maintainers
| [<img src="https://github.com/mrz1836.png" height="50" width="50" alt="MrZ" />](https://github.com/mrz1836) |
|:-----------------------------------------------------------------------------------------------------------:|
|                                      [MrZ](https://github.com/mrz1836)                                      |

<br/>

## 🤝 Contributing
View the [contributing guidelines](.github/CONTRIBUTING.md) and please follow the [code of conduct](.github/CODE_OF_CONDUCT.md).

### How can I help?
All kinds of contributions are welcome :raised_hands:!
The most basic way to show your support is to star :star2: the project, or to raise issues :speech_balloon:.
You can also support this project by [becoming a sponsor on GitHub](https://github.com/sponsors/mrz1836) :clap:
or by making a [**bitcoin donation**](https://mrz1818.com/?tab=tips&utm_source=github&utm_medium=sponsor-link&utm_campaign=go-pre-commit&utm_term=go-pre-commit&utm_content=go-pre-commit) to ensure this journey continues indefinitely! :rocket:

[![Stars](https://img.shields.io/github/stars/mrz1836/go-pre-commit?label=Please%20like%20us&style=social&v=1)](https://github.com/mrz1836/go-pre-commit/stargazers)

<br/>

## 📝 License

[![License](https://img.shields.io/github/license/mrz1836/go-pre-commit.svg?style=flat&v=1)](LICENSE)
