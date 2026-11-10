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
       <a href="https://github.com/mrz1836/go-pre-commit/actions"><img src="https://img.shields.io/github/actions/workflow/status/mrz1836/go-pre-commit/fortress.yml?branch=master&label=build&logo=github&style=flat-square&v=2" alt="Build"></a>
       <a href="https://github.com/mrz1836/go-pre-commit/commits/master"><img src="https://img.shields.io/github/last-commit/mrz1836/go-pre-commit?style=flat-square&logo=git&logoColor=white&label=last%20update" alt="Last Commit"></a>
    </td>
    <td align="right">
       &nbsp;&nbsp;&nbsp;&nbsp; <code>Quality</code> &nbsp;&nbsp;
    </td>
    <td align="left">
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

Install the latest prebuilt release into `~/.local/bin` — a user‑writable directory, so
no `sudo`, and `go-pre-commit update` can self‑update in place afterward.

No `curl … | sudo bash` here — we don't ask you to pipe a mystery script into your shell
and hope for the best. Every line below is in the open, and the download is checked against
the release's published SHA‑256 checksums before anything lands on your `PATH`:

```bash
# Install the latest go-pre-commit release into ~/.local/bin, verified against checksums.txt
VER=$(curl -fsSLI -o /dev/null -w '%{url_effective}' https://github.com/mrz1836/go-pre-commit/releases/latest | sed 's#.*/v##')
OS=$(uname -s | tr '[:upper:]' '[:lower:]'); ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
F="go-pre-commit_${VER}_${OS}_${ARCH}.tar.gz"; U="https://github.com/mrz1836/go-pre-commit/releases/download/v${VER}"
mkdir -p ~/.local/bin && cd "$(mktemp -d)" && curl -fsSLO "$U/$F" \
  && WANT=$(curl -fsSL "$U/go-pre-commit_${VER}_checksums.txt" | awk -v f="$F" '$2==f{print $1}') \
  && GOT=$( { command -v sha256sum >/dev/null && sha256sum "$F" || shasum -a 256 "$F"; } | awk '{print $1}') \
  && [ -n "$WANT" ] && [ "$WANT" = "$GOT" ] \
  && tar -xzf "$F" -C ~/.local/bin go-pre-commit
go-pre-commit --version
```

If `go-pre-commit` isn't found afterward, add `~/.local/bin` to your `PATH` (put
`export PATH="$HOME/.local/bin:$PATH"` in your `~/.zshrc` or `~/.bashrc`).

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
> All formatting and linting tools are automatically installed when needed, from checksum-verified
> downloads (see [How tools are installed](#how-tools-are-installed)), unless you turn that off with
> `GO_PRE_COMMIT_AUTO_INSTALL_TOOLS=false`.

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
GO_PRE_COMMIT_TIMEOUT_SECONDS=720      # Overall timeout (seconds)
GO_PRE_COMMIT_PARALLEL_WORKERS=0       # Parallel workers (0 = auto-detect from CPU cores)
GO_PRE_COMMIT_LOG_LEVEL=info           # Log level: debug, info, warn, error
GO_PRE_COMMIT_MAX_FILE_SIZE_MB=10      # Skip files larger than this

# Individual checks
GO_PRE_COMMIT_ENABLE_EOF=true           # Ensure files end with newline
GO_PRE_COMMIT_ENABLE_FUMPT=true         # Format with gofumpt
GO_PRE_COMMIT_ENABLE_GITLEAKS=false     # Scan for secrets and credentials
GO_PRE_COMMIT_ENABLE_LINT=true          # Run golangci-lint
GO_PRE_COMMIT_ENABLE_MOD_TIDY=true      # Run go mod tidy
GO_PRE_COMMIT_ENABLE_WHITESPACE=true    # Fix trailing whitespace
GO_PRE_COMMIT_GITLEAKS_ALL_FILES=false  # Scan all files, not just staged

# Auto-staging (automatically stage fixed files)
GO_PRE_COMMIT_EOF_AUTO_STAGE=true
GO_PRE_COMMIT_FUMPT_AUTO_STAGE=true
GO_PRE_COMMIT_WHITESPACE_AUTO_STAGE=true

# Tool versions (tools are auto-installed; pin a version or use "latest")
GO_PRE_COMMIT_FUMPT_VERSION=latest
GO_PRE_COMMIT_GOLANGCI_LINT_VERSION=latest
GO_PRE_COMMIT_GITLEAKS_VERSION=v8.29.0

# Tool installation (see "How tools are installed" below)
GO_PRE_COMMIT_AUTO_INSTALL_TOOLS=true     # false: never install; a missing tool fails its check
GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT=300    # seconds allowed for one tool install

# Dual-version pinning for gofumpt (optional). gofumpt raises its go.mod 'go'
# directive over time, so a newer build will not `go install` on an older Go
# toolchain. When _LATEST is set, the version is chosen from the active Go:
#   Go >= _LATEST_MIN_GO -> _LATEST ; otherwise -> GO_PRE_COMMIT_FUMPT_VERSION.
# This lets one set of synced env files work across repos on different Go
# versions without forcing every repo to upgrade Go alongside a tool bump.
# Leave _LATEST unset to keep the single-version behavior above.
GO_PRE_COMMIT_FUMPT_VERSION_LATEST=            # e.g. v0.12.0 (requires Go 1.26+)
GO_PRE_COMMIT_FUMPT_VERSION_LATEST_MIN_GO=1.26 # minimum Go for the _LATEST build

# Per-check timeouts (seconds)
GO_PRE_COMMIT_LINT_TIMEOUT=600          # golangci-lint is usually the slowest
GO_PRE_COMMIT_MOD_TIDY_TIMEOUT=60
GO_PRE_COMMIT_GITLEAKS_TIMEOUT=60
GO_PRE_COMMIT_FUMPT_TIMEOUT=30          # whitespace and eof also default to 30

# File filtering
GO_PRE_COMMIT_EXCLUDE_PATTERNS="vendor/,node_modules/,.git/"

# Plugins (see the Plugin System section below)
GO_PRE_COMMIT_ENABLE_PLUGINS=false
GO_PRE_COMMIT_PLUGIN_DIR=.pre-commit-plugins
GO_PRE_COMMIT_PLUGIN_TIMEOUT=60

# Color output settings (auto-detected by default)
GO_PRE_COMMIT_COLOR_OUTPUT=true             # Enable/disable color output
NO_COLOR=                                   # Set to any value to disable colors (follows standard)
```

> **Full reference:** the variables above are the most commonly used subset. For the complete, annotated list of every `GO_PRE_COMMIT_*` setting and its default, see [.github/env/10-pre-commit.env](.github/env/10-pre-commit.env) (and [.github/env/README.md](.github/env/README.md) for how the modular files are loaded).

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

`install` updates a hook it installed before, but leaves another tool's hook in place unless
you pass `--force`, which moves that hook aside as a backup (`uninstall` puts it back). It exits
non-zero unless every requested hook type is installed, so a script or CI step can rely on its
exit status. When it skips a hook type, it says so and still installs the others.

</details>

<details>
<summary><strong><code>Running Checks Manually</code></strong></summary>
<br/>

```bash
# Run all checks on staged files
go-pre-commit run

# Run a single check by name (positional)
go-pre-commit run lint

# Run only specific checks
go-pre-commit run --only fumpt,lint

# Skip specific checks
go-pre-commit run --skip lint,mod-tidy

# Run on all files (not just staged)
go-pre-commit run --all-files

# Run against specific files
go-pre-commit run --files main.go,utils.go

# List available checks and exit
go-pre-commit run --show-checks

# Control parallelism (0 = auto-detect from CPU cores)
go-pre-commit run --parallel 4

# Stop on the first failing check
go-pre-commit run --fail-fast

# Skip checks that can't run instead of failing
go-pre-commit run --graceful

# Suppress progress output (show only errors and results)
go-pre-commit run --quiet

# Print the results as one JSON document on stdout (see "Machine-Readable Output")
go-pre-commit run --json

# Color output control
go-pre-commit run --color=never     # Disable color output
go-pre-commit run --color=always    # Force color output
go-pre-commit run --color=auto      # Auto-detect (default)
go-pre-commit run --no-color        # Same as --color=never

# Verbose output (global flag, works with any command)
go-pre-commit --verbose run
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

# Print the status as one JSON document on stdout (see "Machine-Readable Output")
go-pre-commit status --json
```

### Updating go-pre-commit

`go-pre-commit update` (alias `go-pre-commit upgrade`) downloads the latest release,
verifies its SHA‑256 checksum against the published `go-pre-commit_<ver>_checksums.txt`,
and atomically replaces the running binary — no `sudo` when it lives in `~/.local/bin`.

```bash
go-pre-commit update            # download & install the latest release
go-pre-commit update --check    # report whether a newer version is available
go-pre-commit update --force    # reinstall the latest even if already current
go-pre-commit update --verbose  # narrate each step
go-pre-commit --version         # print the current version
```

Every other command also runs a passive, cached background check and prints a one‑line
"a new version is available" notice. It never blocks or fails a command, is skipped for
development builds, and is silenced by `GO_PRE_COMMIT_NO_UPDATE_CHECK=1` (or the shared
`NO_UPDATE_CHECK` / `CI`). If you hit GitHub API rate limits, a token is read from
`GO_PRE_COMMIT_GITHUB_TOKEN`, then `GITHUB_TOKEN`, then `GH_TOKEN`.

> **Heads up:** a binary that another installer owns — `go install`'s `~/go/bin`, or a
> Homebrew prefix — is **refused** by `go-pre-commit update` rather than overwritten (that
> would break the tool that owns it). Install the release binary into `~/.local/bin` as
> shown above to keep self‑update working.

### Uninstalling

```bash
# Remove all installed hooks
go-pre-commit uninstall

# Remove specific hook types
go-pre-commit uninstall --hook-type pre-commit
```

</details>

<details>
<summary><strong><code>Machine-Readable Output (--json)</code></strong></summary>
<br/>

`go-pre-commit status --json` and `go-pre-commit run --json` print exactly one JSON document on
stdout, for scripts, editors, and CI steps. The text output without `--json` is unchanged.

- **stdout** carries only the JSON document. With `run --json`, all human-readable output (progress,
  the results block, and tool installation messages, including a golangci-lint download that falls back
  to `go install`) goes to **stderr** instead, still shaped by `--quiet` and `--progress`;
  `status --json` prints nothing else.
- **Exit status** is unchanged: `run` exits 1 when a check failed, else 0; `status` exits 0. Both exit 1
  when they print an error document instead (see **Errors** below).
- Field names are `snake_case`; lists are arrays, empty rather than `null` (except `changed_files`, below);
  durations are whole milliseconds in `duration_ms`. Paths in `status` are absolute.
- New fields may be added in later releases; existing fields keep their names and meaning.

**`go-pre-commit status --json`**

```json
{
  "version": "v1.14.0",
  "enabled": true,
  "config": { "mode": "modular", "path": "/home/me/project/.github/env" },
  "repository_root": "/home/me/project",
  "hooks_dir": "/home/me/project/.git/hooks",
  "hooks": [
    { "name": "pre-commit", "state": "outdated", "path": "/home/me/project/.git/hooks/pre-commit", "executable": true, "outdated": true },
    { "name": "pre-push", "state": "missing", "path": "/home/me/project/.git/hooks/pre-push", "executable": false, "outdated": false },
    { "name": "commit-msg", "state": "missing", "path": "/home/me/project/.git/hooks/commit-msg", "executable": false, "outdated": false },
    { "name": "post-commit", "state": "other", "path": "/home/me/project/.git/hooks/post-commit", "executable": true, "outdated": false }
  ]
}
```

| Field                | Meaning                                                                                            |
|----------------------|----------------------------------------------------------------------------------------------------|
| `enabled`            | `ENABLE_GO_PRE_COMMIT`                                                                             |
| `config.mode`        | `modular` (`.github/env/*.env`) or `legacy` (`.github/.env.base`)                                  |
| `config.path`        | The `.github/env` directory or the `.github/.env.base` file that was read                          |
| `hooks_dir`          | The directory git runs hooks from (honors worktrees and `core.hooksPath`)                          |
| `hooks[]`            | One entry per hook type, always in the order `pre-commit`, `pre-push`, `commit-msg`, `post-commit` |
| `hooks[].state`      | `ready`, `outdated`, `not_executable`, `other`, `missing`, or `error` (see below)                  |
| `hooks[].executable` | The hook file is executable                                                                        |
| `hooks[].outdated`   | The hook is go-pre-commit's but differs from the script this binary would install                  |
| `hooks[].error`      | Why the hook could not be read; present only when `state` is `error`                               |

| State            | Meaning                                                                                      |
|------------------|----------------------------------------------------------------------------------------------|
| `ready`          | go-pre-commit's current hook, executable                                                     |
| `outdated`       | go-pre-commit's hook from another release; run `go-pre-commit install` to update it          |
| `not_executable` | go-pre-commit's hook, but not executable, so git skips it (`outdated` may also be `true`)    |
| `other`          | A hook another tool installed                                                                |
| `missing`        | No hook file                                                                                 |
| `error`          | The hook file could not be read                                                              |

**`go-pre-commit run --json`**

```json
{
  "version": "v1.14.0",
  "enabled": true,
  "result": "failed",
  "files": ["main.go", "README.md"],
  "checks": [
    { "name": "eof", "result": "passed", "duration_ms": 0, "files": ["main.go", "README.md"], "changed_files": [] },
    { "name": "fumpt", "result": "passed", "duration_ms": 41, "files": ["main.go"], "changed_files": ["main.go"] },
    { "name": "lint", "result": "failed", "duration_ms": 2310, "files": ["main.go"], "changed_files": null,
      "error": "Found 1 linting issue(s):\nmain.go:10:5: ineffectual assignment to x (ineffassign)",
      "suggestion": "Fix the linting issues shown above. Run 'golangci-lint run .' to see full details.",
      "output": "Found 1 linting issue(s):\nmain.go:10:5: ineffectual assignment to x (ineffassign)" },
    { "name": "whitespace", "result": "failed", "duration_ms": 1, "files": ["main.go", "README.md"],
      "changed_files": ["README.md"], "error": "whitespace issues found" }
  ],
  "totals": { "checks": 4, "passed": 2, "failed": 2, "skipped": 0, "files": 2, "duration_ms": 2353 },
  "warnings": []
}
```

| Field                    | Meaning                                                                                                   |
|--------------------------|-----------------------------------------------------------------------------------------------------------|
| `enabled`                | `ENABLE_GO_PRE_COMMIT`; when `false`, nothing ran                                                         |
| `result`                 | `failed` if any check failed; `skipped` if no check ran (system disabled, or no files to check); else `passed` |
| `files`                  | The files given to the checks (staged files, `--all-files`, or `--files`)                                 |
| `checks[]`               | One entry per check that ran, sorted by name                                                              |
| `checks[].result`        | `passed`, `failed`, or `skipped` (the check could not run and `--graceful` skipped it)                     |
| `checks[].files`         | The files the check processed, after its own file filter                                                 |
| `checks[].changed_files` | The files the check rewrote (`eof`, `whitespace`, `fumpt`); `[]` if none; `null` for a check that does not report what it changes |
| `checks[].error`         | Why the check failed or was skipped; present only then                                                    |
| `checks[].suggestion`    | How to fix it; present only when there is one                                                             |
| `checks[].output`        | The tool's output, without color codes; present only for a failed check that has some                    |
| `totals`                 | Counts of `checks`, `passed`, `failed`, `skipped`, and `files`, and the run's `duration_ms`              |
| `warnings`               | Messages about the run, such as a disabled system or a skipped check                                     |

Note that `eof` and `whitespace` fail when they fix a file, so the fix can be reviewed and staged, while
`fumpt` formats files and passes; `changed_files` lists what each of them rewrote either way.

**Errors.** When a command cannot produce its report, it prints an error document instead and exits 1:

```json
{
  "version": "v1.14.0",
  "error": { "code": "config_not_found", "message": "failed to load configuration: ..." }
}
```

| Code               | Commands      | Meaning                                                             |
|--------------------|---------------|---------------------------------------------------------------------|
| `config_not_found` | status, run   | No `.github/env/*.env` and no `.github/.env.base`                   |
| `config_invalid`   | status, run   | The configuration could not be read, or failed validation           |
| `not_a_repository` | status, run   | Not inside a git repository                                         |
| `git_failed`       | run           | git could not list the files to check                               |
| `no_checks_to_run` | run           | `--only`, `--skip`, or the check argument matched no enabled check  |
| `run_failed`       | run           | Any other failure before the checks could report                   |

`--json` cannot be combined with `run --show-checks`.

```bash
# Is the pre-commit hook current?
go-pre-commit status --json | jq -r '.hooks[] | select(.name == "pre-commit") | .state'

# Which files did the fixing checks rewrite?
go-pre-commit run --json 2>/dev/null | jq -r '.checks[].changed_files // [] | .[]'
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

All checks run in parallel for maximum performance. The whitespace, eof, and mod-tidy checks are pure Go with no dependencies; fumpt, lint, and gitleaks shell out to external tools (gofumpt, golangci-lint, gitleaks) that are auto-installed on first use — so everything works out of the box.

#### How tools are installed

When a check's tool is not on your `PATH`, `go-pre-commit` installs the version you pinned:

- **golangci-lint and gitleaks** are downloaded from the tool's GitHub release over HTTPS. `go-pre-commit`
  fetches the release's checksums file, takes the one line for the archive's exact file name, downloads
  the archive, and checks its SHA‑256 before extracting anything. Only the tool's binary is extracted, into
  `$GOBIN` (or `$GOPATH/bin`). No install script is downloaded or run. If golangci-lint's download fails at
  any step, `go-pre-commit` falls back to `go install` of the same version.
- **gofumpt and goimports** are installed with `go install`, which the Go module proxy and checksum
  database verify.

A failed install is never silent: the check fails and reports the tool, the version, and the step that
failed (for example `verify checksum`). Make sure `$GOBIN` (or `$GOPATH/bin`) is on your `PATH`, so the
installed tools are found.

To never install anything, set `GO_PRE_COMMIT_AUTO_INSTALL_TOOLS=false`. A missing tool then fails its check
with the exact command that installs the pinned version, for example
`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0`. Tools already on your `PATH`
are used either way.

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

# Remove a plugin (add --force to skip the confirmation prompt)
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

**Full manifest reference** — `name`, `version`, `description`, `executable`, and `file_patterns` are required; everything else is optional:

| Field                                                     | Description                                                    |
|-----------------------------------------------------------|----------------------------------------------------------------|
| `name` / `version` / `description`                        | Plugin identity and summary                                    |
| `executable`                                              | Path to the script or binary to run                            |
| `file_patterns`                                           | Globs selecting which files to process (e.g. `"*.go"`)         |
| `author` / `license` / `homepage`                         | Optional metadata                                              |
| `args`                                                    | Extra arguments passed to the executable                       |
| `timeout`                                                 | Per-plugin timeout (e.g. `30s`)                                |
| `category`                                                | Grouping label (e.g. `linting`, `security`)                    |
| `requires_files`                                          | Skip the plugin when no matching files are staged              |
| `environment`                                             | Map of environment variables passed to the plugin             |
| `dependencies`                                            | Tools/binaries the plugin requires                             |
| `min_go_pre_commit_version` / `max_go_pre_commit_version` | Version constraints                                            |
| `read_only`                                               | Declare that the plugin will not modify files                  |
| `allowed_paths`                                           | Restrict the plugin's filesystem access                        |
| `max_memory_mb` / `max_cpu_percent`                       | Resource limits                                                |

**JSON protocol** — `go-pre-commit` pipes a request to the plugin's stdin and reads a response from stdout:

```jsonc
// Request (stdin)
{
  "command": "check",           // requested action
  "files": ["main.go"],         // files to process
  "config": { "key": "value" }  // optional config map
}

// Response (stdout)
{
  "success": true,              // did the check pass?
  "output": "Check passed",     // human-readable output
  "error": "",                  // error message when success is false
  "suggestion": "",             // optional remediation hint
  "modified": []                // files the plugin changed (for auto-staging)
}
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

### 2. (Optional) Configure go-pre-commit

`go-pre-commit` ships with sensible defaults, so configuration is **optional** — you can skip straight to step 3. To customize, create a modular env file (the preferred approach):

```bash
# Create the modular env directory and a project override file
mkdir -p .github/env

cat > .github/env/90-project.env << 'EOF'
ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_ENABLE_EOF=true
GO_PRE_COMMIT_ENABLE_FUMPT=true
GO_PRE_COMMIT_ENABLE_GITLEAKS=false
GO_PRE_COMMIT_ENABLE_LINT=true
GO_PRE_COMMIT_ENABLE_MOD_TIDY=true
GO_PRE_COMMIT_ENABLE_WHITESPACE=true
EOF
```

> Prefer the legacy single-file layout? Create `.github/.env.base` (and an optional `.github/.env.custom`) instead — see the [Configuration](#-configuration) section for how the two modes are detected.

### 3. Install go-pre-commit

Install the `go-pre-commit` binary as shown in the Quickstart above, then set up hooks in
your repository:

```bash
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
<summary><strong>Repository Features</strong></summary>
<br/>

This repository includes 25+ built-in features covering CI/CD, security, code quality, developer experience, and community tooling.

**[View the full Repository Features list →](.github/docs/repository-features.md)**

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
<summary><strong>GitHub Workflows</strong></summary>
<br/>

All workflows are driven by modular configuration in [`.github/env/`](.github/env/README.md) — no YAML editing required.

**[View all workflows and the control center →](.github/docs/workflows.md)**

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

All unit tests and fuzz tests run via [GitHub Actions](https://github.com/mrz1836/go-pre-commit/actions) and use [Go version 1.26.x](https://go.dev/doc/go1.26). View the [configuration file](.github/workflows/fortress.yml).

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

| Benchmark                                                                               | Iterations |     ns/op |    B/op | allocs/op | Description                      |
| --------------------------------------------------------------------------------------- | ---------- | --------: | ------: | --------: | -------------------------------- |
| [BenchmarkPreCommitSystem_SmallProject](internal/benchmark_test.go)                     | 91,188     |    12,737 |  15,542 |        70 | Small project (3 files)          |
| [BenchmarkPreCommitSystem_EndToEnd](internal/benchmark_test.go)                         | 50,778     |    21,962 |  36,218 |       108 | Full system (8 files)            |
| [BenchmarkPreCommitSystem_LargeProject](internal/benchmark_test.go)                     | 23,516     |    53,930 | 109,041 |       226 | Large project (25+ files)        |
| [BenchmarkRunner_New](internal/runner/runner_bench_test.go)                             | 3,575,842  |     337.0 |     624 |        11 | Runner creation                  |
| [BenchmarkRunner_Run_SingleCheck](internal/runner/runner_bench_test.go)                 | 236,641    |     5,037 |   7,288 |        31 | Single check execution           |
| [BenchmarkWhitespaceCheck_Run_SmallFile](internal/checks/builtin/builtin_bench_test.go) | 6,065,563  |     184.3 |     128 |         2 | Whitespace check (small file)    |
| [BenchmarkWhitespaceCheck_Parallel](internal/checks/builtin/builtin_bench_test.go)      | 8,422,394  |     142.6 |     128 |         2 | Parallel whitespace processing   |
| [BenchmarkRepository_GetAllFiles](internal/git/git_bench_test.go)                       | 154        | 7,638,277 | 109,620 |       353 | Git file enumeration             |
| [BenchmarkRunner_Performance_SmallCommit](internal/runner/performance_bench_test.go)    | 53,103     |    21,797 |  16,929 |       105 | Typical small commit (1-3 files) |

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
