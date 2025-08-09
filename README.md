# 🔒 go-pre-commit
> Lightning-fast, Git pre-commit hooks for Go projects - built in pure Go

<table>
  <thead>
    <tr>
      <th>CI&nbsp;/&nbsp;CD</th>
      <th>Quality&nbsp;&amp;&nbsp;Security</th>
      <th>Docs&nbsp;&amp;&nbsp;Meta</th>
      <th>Community</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td valign="top" align="left">
        <a href="https://github.com/mrz1836/go-pre-commit/releases">
          <img src="https://img.shields.io/github/release-pre/mrz1836/go-pre-commit?logo=github&style=flat&v=1" alt="Latest Release">
        </a><br/>
        <a href="https://github.com/mrz1836/go-pre-commit/actions">
          <img src="https://img.shields.io/github/actions/workflow/status/mrz1836/go-pre-commit/fortress.yml?branch=master&logo=github&style=flat" alt="Build Status">
        </a><br/>
		<a href="https://github.com/mrz1836/go-pre-commit/actions">
          <img src="https://github.com/mrz1836/go-pre-commit/actions/workflows/codeql-analysis.yml/badge.svg?style=flat" alt="CodeQL">
        </a><br/>
        <a href="https://github.com/mrz1836/go-pre-commit/commits/master">
		  <img src="https://img.shields.io/github/last-commit/mrz1836/go-pre-commit?style=flat&logo=clockify&logoColor=white" alt="Last commit">
		</a>
      </td>
      <td valign="top" align="left">
        <a href="https://goreportcard.com/report/github.com/mrz1836/go-pre-commit">
          <img src="https://goreportcard.com/badge/github.com/mrz1836/go-pre-commit?style=flat" alt="Go Report Card">
        </a><br/>
		<a href="https://codecov.io/gh/mrz1836/go-pre-commit/tree/master">
          <img src="https://codecov.io/gh/mrz1836/go-pre-commit/branch/master/graph/badge.svg?style=flat" alt="Code Coverage">
        </a><br/>
		<a href="https://scorecard.dev/viewer/?uri=github.com/mrz1836/go-pre-commit">
          <img src="https://api.scorecard.dev/projects/github.com/mrz1836/go-pre-commit/badge?logo=springsecurity&logoColor=white" alt="OpenSSF Scorecard">
        </a><br/>
		<a href=".github/SECURITY.md">
          <img src="https://img.shields.io/badge/security-policy-blue?style=flat&logo=springsecurity&logoColor=white" alt="Security policy">
        </a>
      </td>
      <td valign="top" align="left">
        <a href="https://golang.org/">
          <img src="https://img.shields.io/github/go-mod/go-version/mrz1836/go-pre-commit?style=flat" alt="Go version">
        </a><br/>
        <a href="https://pkg.go.dev/github.com/mrz1836/go-pre-commit?tab=doc">
          <img src="https://pkg.go.dev/badge/github.com/mrz1836/go-pre-commit.svg?style=flat" alt="Go docs">
        </a><br/>
        <a href=".github/AGENTS.md">
          <img src="https://img.shields.io/badge/AGENTS.md-found-40b814?style=flat&logo=openai" alt="AGENTS.md rules">
        </a><br/>
        <a href="Makefile">
          <img src="https://img.shields.io/badge/Makefile-supported-brightgreen?style=flat&logo=probot&logoColor=white" alt="Makefile Supported">
        </a><br/>
		<a href=".github/dependabot.yml">
          <img src="https://img.shields.io/badge/dependencies-automatic-blue?logo=dependabot&style=flat" alt="Dependabot">
        </a>
      </td>
      <td valign="top" align="left">
        <a href="https://github.com/mrz1836/go-pre-commit/graphs/contributors">
          <img src="https://img.shields.io/github/contributors/mrz1836/go-pre-commit?style=flat&logo=contentful&logoColor=white" alt="Contributors">
        </a><br/>
        <a href="https://github.com/sponsors/mrz1836">
          <img src="https://img.shields.io/badge/sponsor-MrZ-181717.svg?logo=github&style=flat" alt="Sponsor">
        </a><br/>
        <a href="https://mrz1818.com/?tab=tips&utm_source=github&utm_medium=sponsor-link&utm_campaign=go-pre-commit&utm_term=go-pre-commit&utm_content=go-pre-commit">
          <img src="https://img.shields.io/badge/donate-bitcoin-ff9900.svg?logo=bitcoin&style=flat" alt="Donate Bitcoin">
        </a>
      </td>
    </tr>
  </tbody>
</table>

<br/>

## 🗂️ Table of Contents
* [Installation](#-installation)
* [Documentation](#-documentation)
* [Examples & Tests](#-examples--tests)
* [Benchmarks](#-benchmarks)
* [Code Standards](#-code-standards)
* [AI Compliance](#-ai-compliance)
* [Maintainers](#-maintainers)
* [Contributing](#-contributing)
* [License](#-license)

<br/>

## 🚀 Quickstart

Get up and running with `go-pre-commit` in 30 seconds:

### Install the binary

```bash
# Install from source (requires Go 1.24+)
go install github.com/mrz1836/go-pre-commit/cmd/go-pre-commit@latest

# Or clone and build locally
git clone https://github.com/mrz1836/go-pre-commit.git
cd go-pre-commit
make install
```

### Set up in your project

```bash
# Navigate to your Go project
cd your-go-project

# Install the pre-commit hook
go-pre-commit install

# That's it! Your pre-commit hooks are now active
```

### Test it out

```bash
# Make a change and commit
echo "test" >> test.go
git add test.go
git commit -m "Test commit"

# The pre-commit system will automatically run checks:
# ✓ Running fumpt formatter
# ✓ Running golangci-lint
# ✓ Running go mod tidy
# ✓ Fixing trailing whitespace
# ✓ Ensuring files end with newline
```

> **Good to know:** `go-pre-commit` ships with *zero* runtime dependencies.
> It's a single Go binary that replaces Python-based pre-commit frameworks.

<br/>

## ⚙️ Configuration

`go-pre-commit` uses environment variables from `.github/.env.shared` for configuration:

```bash
# Core settings
ENABLE_PRE_COMMIT_SYSTEM=true              # Enable/disable the system
PRE_COMMIT_SYSTEM_FAIL_FAST=false          # Stop on first failure
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=120      # Overall timeout
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=2       # Number of parallel workers

# Individual checks
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=true        # Format with fumpt
PRE_COMMIT_SYSTEM_ENABLE_LINT=true         # Run golangci-lint
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=true     # Run go mod tidy
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true   # Fix trailing whitespace
PRE_COMMIT_SYSTEM_ENABLE_EOF=true          # Ensure files end with newline

# Auto-staging (automatically stage fixed files)
PRE_COMMIT_SYSTEM_WHITESPACE_AUTO_STAGE=true
PRE_COMMIT_SYSTEM_EOF_AUTO_STAGE=true
```

<br/>

## 🔄 Workflow Process

### Installing hooks

```bash
# Install default pre-commit hook
go-pre-commit install

# Install with specific hook types
go-pre-commit install --hook-type pre-commit --hook-type pre-push

# Force reinstall (overwrites existing hooks)
go-pre-commit install --force
```

### Running checks manually

```bash
# Run all checks on staged files
go-pre-commit run

# Run specific checks
go-pre-commit run --checks fumpt,lint

# Run on all files (not just staged)
go-pre-commit run --all-files

# Skip specific checks
go-pre-commit run --skip lint,mod-tidy
```

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
# Update to latest version
go update github.com/mrz1836/go-pre-commit/cmd/go-pre-commit@latest

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

<br/>

## 🎯 Available Checks

`go-pre-commit` includes these built-in checks:

| Check | Description | Auto-fix | Configuration |
|-------|-------------|----------|---------------|
| **fumpt** | Formats Go code with stricter rules than `gofmt` | ✅ | Requires `make fumpt` target |
| **lint** | Runs golangci-lint for comprehensive linting | ❌ | Requires `make lint` target |
| **mod-tidy** | Ensures go.mod and go.sum are tidy | ✅ | Requires `make mod-tidy` target |
| **whitespace** | Removes trailing whitespace | ✅ | Auto-stages changes if enabled |
| **eof** | Ensures files end with a newline | ✅ | Auto-stages changes if enabled |

All checks run in parallel for maximum performance. Make targets are detected automatically from your Makefile.

<br/>

## 🏗️ Starting a New Project

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

# Download the example configuration
curl -o .github/.env.shared https://raw.githubusercontent.com/mrz1836/go-pre-commit/master/.github/.env.shared

# Or create a minimal configuration
cat > .github/.env.shared << 'EOF'
ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=true
PRE_COMMIT_SYSTEM_ENABLE_LINT=true
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=true
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
EOF
```

### 3. Create a Makefile with required targets

```bash
cat > Makefile << 'EOF'
.PHONY: fumpt lint mod-tidy

fumpt:
	@go run mvdan.cc/gofumpt@latest -w .

lint:
	@golangci-lint run

mod-tidy:
	@go mod tidy
EOF
```

### 4. Install go-pre-commit

```bash
# Install the tool
go install github.com/mrz1836/go-pre-commit/cmd/go-pre-commit@latest

# Set up hooks in your repository
go-pre-commit install

# Verify installation
go-pre-commit status
```

### 5. Test your setup

```bash
# Create a test file with issues
echo -e "package main\n\nfunc main() {  \n}" > main.go

# Try to commit
git add .
git commit -m "Initial commit"

# Watch go-pre-commit automatically fix issues!
```

<br/>

## 📚 Advanced Documentation

- **Performance** – Parallel execution with configurable workers for blazing-fast checks
- **CI Integration** – Seamlessly integrates with GitHub Actions via shared configuration
- **Make Compatibility** – Leverages existing Makefile targets for consistency
- **Environment Config** – All settings in `.github/.env.shared` for team synchronization

<br/>

<details>
<summary><strong><code>Repository Features</code></strong></summary>
<br/>

* **Continuous Integration on Autopilot** with [GitHub Actions](https://github.com/features/actions) – every push is built, tested, and reported in minutes.
* **Pull‑Request Flow That Merges Itself** thanks to [auto‑merge](.github/workflows/auto-merge-on-approval.yml) and hands‑free [Dependabot auto‑merge](.github/workflows/dependabot-auto-merge.yml).
* **One‑Command Builds** powered by battle‑tested [Make](https://www.gnu.org/software/make) targets for linting, testing, releases, and more.
* **First‑Class Dependency Management** using native [Go Modules](https://github.com/golang/go/wiki/Modules).
* **Uniform Code Style** via [gofumpt](https://github.com/mvdan/gofumpt) plus zero‑noise linting with [golangci‑lint](https://github.com/golangci/golangci-lint).
* **Confidence‑Boosting Tests** with [testify](https://github.com/stretchr/testify), the Go [race detector](https://blog.golang.org/race-detector), crystal‑clear [HTML coverage](https://blog.golang.org/cover) snapshots, and automatic uploads to [Codecov](https://codecov.io/).
* **Hands‑Free Releases** delivered by [GoReleaser](https://github.com/goreleaser/goreleaser) whenever you create a [new Tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging).
* **Relentless Dependency & Vulnerability Scans** via [Dependabot](https://dependabot.com), [Nancy](https://github.com/sonatype-nexus-community/nancy), and [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck).
* **Security Posture by Default** with [CodeQL](https://docs.github.com/en/github/finding-security-vulnerabilities-and-errors-in-your-code/about-code-scanning), [OpenSSF Scorecard](https://openssf.org), and secret‑leak detection via [gitleaks](https://github.com/gitleaks/gitleaks).
* **Automatic Syndication** to [pkg.go.dev](https://pkg.go.dev/) on every release for instant godoc visibility.
* **Polished Community Experience** using rich templates for [Issues & PRs](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository).
* **All the Right Meta Files** (`LICENSE`, `CITATION.cff`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SUPPORT.md`, `SECURITY.md`) pre‑filled and ready.
* **Code Ownership** clarified through a [CODEOWNERS](.github/CODEOWNERS) file, keeping reviews fast and focused.
* **Zero‑Noise Dev Environments** with tuned editor settings (`.editorconfig`) plus curated *ignore* files for [VS Code](.editorconfig), [Docker](.dockerignore), and [Git](.gitignore).
* **Label Sync Magic**: your repo labels stay in lock‑step with [.github/labels.yml](.github/labels.yml).
* **Friendly First PR Workflow** – newcomers get a warm welcome thanks to a dedicated [workflow](.github/workflows/pull-request-management.yml).
* **Standards‑Compliant Docs** adhering to the [standard‑readme](https://github.com/RichardLitt/standard-readme/blob/master/spec.md) spec.
* **Instant Cloud Workspaces** via [Gitpod](https://gitpod.io/) – spin up a fully configured dev environment with automatic linting and tests.
* **Out‑of‑the‑Box VS Code Happiness** with a preconfigured [Go](https://code.visualstudio.com/docs/languages/go) workspace and [`.vscode`](.vscode) folder with all the right settings.
* **Optional Release Broadcasts** to your community via [Slack](https://slack.com), [Discord](https://discord.com), or [Twitter](https://twitter.com) – plug in your webhook.
* **AI Compliance Playbook** – machine‑readable guidelines ([AGENTS.md](.github/AGENTS.md), [CLAUDE.md](.github/CLAUDE.md), [.cursorrules](.cursorrules), [sweep.yaml](.github/sweep.yaml)) keep ChatGPT, Claude, Cursor & Sweep aligned with your repo's rules.
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

To generate a snapshot (non-versioned) release for testing purposes, run:

```bash
make release-snap
```

Before tagging a new version, update the release metadata (version) in the `CITATION.cff` file:

```bash
make citation version=0.2.1
```

Then create and push a new Git tag using:

```bash
make tag version=x.y.z
```

This process ensures consistent, repeatable releases with properly versioned artifacts and citation metadata.

</details>

<details>
<summary><strong><code>Makefile Commands</code></strong></summary>
<br/>

View all `makefile` commands

```bash script
make help
```

List of all current commands:

<!-- make-help-start -->
```text
bench                 ## Run all benchmarks in the Go application
build-go              ## Build the Go application (locally)
citation              ## Update version in CITATION.cff (use version=X.Y.Z)
clean-mods            ## Remove all the Go mod cache
coverage              ## Show test coverage
diff                  ## Show git diff and fail if uncommitted changes exist
fumpt                 ## Run fumpt to format Go code
generate              ## Run go generate in the base of the repo
godocs                ## Trigger GoDocs tag sync
govulncheck-install   ## Install govulncheck (pass VERSION= to override)
govulncheck           ## Scan for vulnerabilities
help                  ## Display this help message
install-go            ## Install using go install with specific version
install-releaser      ## Install GoReleaser
install-stdlib        ## Install the Go standard library for the host platform
install               ## Install the application binary
lint-version          ## Show the golangci-lint version
lint                  ## Run the golangci-lint application (install if not found)
loc                   ## Total lines of code table
mod-download          ## Download Go module dependencies
mod-tidy              ## Clean up go.mod and go.sum
pre-build             ## Pre-build all packages to warm cache
release-snap          ## Build snapshot binaries
release-test          ## Run release dry-run (no publish)
release               ## Run production release (requires github_token)
tag-remove            ## Remove local and remote tag (use version=X.Y.Z)
tag-update            ## Force-update tag to current commit (use version=X.Y.Z)
tag                   ## Create and push a new tag (use version=X.Y.Z)
test-ci-no-race       ## CI test suite without race detector
test-ci               ## CI test runs tests with race detection and coverage (no lint - handled separately)
test-cover-race       ## Runs unit tests with race detector and outputs coverage
test-cover            ## Unit tests with coverage (no race)
test-fuzz             ## Run fuzz tests only (no unit tests)
test-no-lint          ## Run only tests (no lint)
test-parallel         ## Run tests in parallel (faster for large repos)
test-race             ## Unit tests with race detector (no coverage)
test-short            ## Run tests excluding integration tests (no lint)
test                  ## Default testing uses lint + unit tests (fast)
uninstall             ## Uninstall the Go binary
update-linter         ## Upgrade golangci-lint (macOS only)
update-releaser       ## Reinstall GoReleaser
update                ## Update dependencies
vet-parallel          ## Run go vet in parallel (faster for large repos)
vet                   ## Run go vet only on your module packages
```
<!-- make-help-end -->

</details>

<details>
<summary><strong><code>GitHub Workflows</code></strong></summary>
<br/>


### 🎛️ The Workflow Control Center

All GitHub Actions workflows in this repository are powered by a single configuration file: [**.env.shared**](.github/.env.shared) – your one-stop shop for tweaking CI/CD behavior without touching a single YAML file! 🎯

This magical file controls everything from:
- **🚀 Go version matrix** (test on multiple versions or just one)
- **🏃 Runner selection** (Ubuntu or macOS, your wallet decides)
- **🔬 Feature toggles** (coverage, fuzzing, linting, race detection, benchmarks)
- **🛡️ Security tool versions** (gitleaks, nancy, govulncheck)
- **🤖 Auto-merge behaviors** (how aggressive should the bots be?)
- **🏷️ PR management rules** (size labels, auto-assignment, welcome messages)

> **Pro tip:** Want to disable code coverage? Just flip `ENABLE_CODE_COVERAGE=false` in [.env.shared](.github/.env.shared) and push. No YAML archaeology required!

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
make update
```

This command ensures all dependencies are brought up to date in a single step, including Go modules and any tools managed by the Makefile. It is the recommended way to keep your development environment and CI in sync with the latest versions.

</details>

<br/>

## 🧪 Examples & Tests

All unit tests and [examples](examples) run via [GitHub Actions](https://github.com/mrz1836/go-pre-commit/actions) and use [Go version 1.24.x](https://go.dev/doc/go1.24). View the [configuration file](.github/workflows/fortress.yml).

Run all tests (fast):

```bash script
make test
```

Run all tests with race detector (slower):
```bash script
make test-race
```

<br/>

## ⚡ Benchmarks

Run the Go [benchmarks](template_benchmark_test.go):

```bash script
make bench
```

<br/>

### Benchmark Results

| Benchmark                           | Iterations | ns/op | B/op | allocs/op |
|-------------------------------------|------------|------:|-----:|----------:|
| [Greet](template_benchmark_test.go) | 21,179,739 | 56.59 |   40 |         2 |

> These benchmarks reflect fast, allocation-free lookups for most retrieval functions, ensuring optimal performance in production environments.
> Performance benchmarks for the core functions in this library, executed on an Apple M1 Max (ARM64).

<br/>

## 🛠️ Code Standards
Read more about this Go project's [code standards](.github/CODE_STANDARDS.md).

<br/>

## 🤖 AI Compliance
This project documents expectations for AI assistants using a few dedicated files:

- [AGENTS.md](.github/AGENTS.md) — canonical rules for coding style, workflows, and pull requests used by [Codex](https://chatgpt.com/codex).
- [CLAUDE.md](.github/CLAUDE.md) — quick checklist for the [Claude](https://www.anthropic.com/product) agent.
- [.cursorrules](.cursorrules) — machine-readable subset of the policies for [Cursor](https://www.cursor.so/) and similar tools.
- [sweep.yaml](.github/sweep.yaml) — rules for [Sweep](https://github.com/sweepai/sweep), a tool for code review and pull request management.

Edit `AGENTS.md` first when adjusting these policies, and keep the other files in sync within the same pull request.

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
