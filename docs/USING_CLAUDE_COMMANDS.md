# 📚 Claude Code Commands Reference

This document provides a comprehensive guide to all custom slash commands available for managing the go-pre-commit project. These commands leverage our specialized sub-agents team for efficient project management.

## 🎯 Quick Command Reference

### Core Commands (Most Used)
| Command     | Description                                    | Example                |
|-------------|------------------------------------------------|------------------------|
| `/fix`      | Fix test errors and linting issues in parallel | `/fix internal/runner` |
| `/test`     | Create or fix tests with go-tester             | `/test ProcessCheck`   |
| `/review`   | Comprehensive code review with multiple agents | `/review master`       |
| `/docs`     | Update documentation for features              | `/docs hooks`          |
| `/clean`    | Remove duplicate code and improve structure    | `/clean internal/`     |
| `/validate` | Full validation suite (parallel execution)     | `/validate`            |

### Development Workflow
| Command     | Description                            | Example                        |
|-------------|----------------------------------------|--------------------------------|
| `/pr`       | Complete pull request workflow         | `/pr "Add parallel execution"` |
| `/ci`       | Diagnose and fix CI/CD issues          | `/ci fortress.yml`             |
| `/explain`  | Explain how code/features work         | `/explain runner`              |
| `/prd`      | Design product requirement document    | `/prd "multi-repo support"`    |
| `/refactor` | Refactor code with duplication removal | `/refactor internal/checks`    |

### Specialized Commands
| Command      | Description                                | Example                  |
|--------------|--------------------------------------------|--------------------------|
| `/audit`     | Security audit with vulnerability scanning | `/audit`                 |
| `/optimize`  | Performance optimization workflow          | `/optimize ProcessCheck` |
| `/release`   | Prepare and validate release               | `/release v1.2.3`        |
| `/hooks`     | Fix pre-commit hook issues                 | `/hooks`                 |
| `/build`     | Fix Makefile and build issues              | `/build test`            |
| `/standards` | Validate Go standards compliance           | `/standards internal/`   |

### Advanced Commands (Namespaced)
| Command        | Description                   | Example                           |
|----------------|-------------------------------|-----------------------------------|
| `/dev:feature` | Start new feature development | `/dev:feature parallel-checks`    |
| `/dev:hotfix`  | Emergency fix workflow        | `/dev:hotfix "race condition"`    |
| `/dev:debug`   | Debug complex issues          | `/dev:debug "timeout in CI"`      |
| `/go:bench`    | Run and analyze benchmarks    | `/go:bench Runner`                |
| `/go:deps`     | Manage Go dependencies        | `/go:deps update testify`         |
| `/go:profile`  | Profile performance issues    | `/go:profile cpu internal/runner` |

---

## 💫 Command Details

### Core Commands

<details>
<summary><strong><code>/fix - Fix Test and Linting Issues</code></strong></summary>

**Purpose**: Rapidly fix test failures and linting violations using parallel agent execution.

**Usage**: `/fix [specific-file-or-package]`

**Agents Used**:
- go-tester (test fixes)
- go-formatter (linting)
- go-standards-enforcer (compliance)

**Example**:
```bash
> /fix internal/runner
# Fixes all issues in the runner package

> /fix
# Fixes all issues in the project
```

**What it does**:
1. Identifies all test failures and linting issues
2. Runs specialized agents in parallel
3. Automatically fixes formatting issues
4. Updates tests to ensure they pass
5. Validates standards compliance

</details>

<details>
<summary><strong><code>/test - Create or Fix Tests</code></strong></summary>

**Purpose**: Create comprehensive tests or fix existing test failures.

**Usage**: `/test [package-or-function-name]`

**Agents Used**:
- go-tester (primary)

**Example**:
```bash
> /test ProcessCheck
# Creates tests for ProcessCheck function

> /test internal/config
# Fixes/creates tests for config package
```

**Features**:
- Uses testify exclusively
- Creates table-driven tests
- Ensures 90%+ coverage
- Follows naming conventions
- Handles edge cases

</details>

<details>
<summary><strong><code>/review - Comprehensive Code Review</code></strong></summary>

**Purpose**: Perform thorough code review using multiple specialized agents in parallel.

**Usage**: `/review [branch-or-commit]`

**Agents Used** (in parallel):
- code-reviewer (security, quality)
- go-standards-enforcer (compliance)
- performance-optimizer (performance)
- dependency-auditor (dependencies)

**Example**:
```bash
> /review
# Reviews current changes

> /review feat/new-feature
# Reviews specific branch
```

**Output Categories**:
- 🚨 Critical issues (must fix)
- ⚠️ Warnings (should fix)
- 💡 Suggestions (consider)
- ✅ Positive observations

</details>

### Workflow Commands

<details>
<summary><strong><code>/pr - Pull Request Workflow</code></strong></summary>

**Purpose**: Orchestrate complete pull request preparation.

**Usage**: `/pr [pr-title]`

**Agents Used**:
- pr-orchestrator (coordination)
- go-formatter (formatting)
- go-tester (testing)
- go-standards-enforcer (standards)
- code-reviewer (review)

**Example**:
```bash
> /pr "Add parallel execution support"
```

**Workflow**:
1. Validates branch naming
2. Runs all checks in parallel
3. Prepares PR description
4. Applies appropriate labels
5. Ensures CI readiness

</details>

<details>
<summary><strong><code>/ci - Diagnose CI Issues</code></strong></summary>

**Purpose**: Identify and fix CI/CD pipeline problems.

**Usage**: `/ci [workflow-name-or-run-id]`

**Agents Used**:
- ci-guardian (primary)
- go-tester (test failures)
- go-formatter (linting)
- dependency-auditor (security)

**Example**:
```bash
> /ci fortress.yml
# Diagnoses specific workflow

> /ci
# Analyzes all recent failures
```

**Capabilities**:
- Analyzes workflow failures
- Identifies root causes
- Coordinates fixes
- Optimizes performance

</details>

### Specialized Commands

<details>
<summary><strong><code>/audit - Security Audit</code></strong></summary>

**Purpose**: Comprehensive security vulnerability scanning.

**Usage**: `/audit`

**Agents Used**:
- dependency-auditor (primary)

**Scans Performed**:
- govulncheck (Go vulnerabilities)
- nancy (dependency CVEs)
- gitleaks (secret detection)
- License compliance
- Security best practices

**Output**:
- Vulnerability report by severity
- Remediation steps
- Update recommendations

</details>

<details>
<summary><strong><code>/optimize - Performance Optimization</code></strong></summary>

**Purpose**: Profile and optimize code performance.

**Usage**: `/optimize [package-or-function]`

**Agents Used**:
- performance-optimizer (primary)

**Process**:
1. Run benchmarks
2. Generate profiles
3. Identify bottlenecks
4. Apply optimizations
5. Validate improvements

**Techniques**:
- Memory optimization
- CPU optimization
- I/O optimization
- Concurrency improvements

</details>

---

## 🚀 Common Workflows

### Starting a New Feature
```bash
> /dev:feature user-authentication
```
This command will:
1. Create feature branch
2. Set up initial structure
3. Create documentation plan
4. Prepare test framework
5. Set up PR tracking

### Fixing a Production Issue
```bash
> /dev:hotfix "database connection timeout"
```
Rapid response workflow:
1. Create hotfix branch
2. Diagnose issue
3. Implement minimal fix
4. Fast validation
5. Prepare patch release

### Pre-Release Validation
```bash
> /validate
> /audit
> /release v1.2.3
```
Complete release preparation:
1. Run full validation suite
2. Security audit
3. Prepare release artifacts

### Performance Investigation
```bash
> /go:profile cpu internal/runner
> /go:bench Runner
> /optimize internal/runner
```
Performance workflow:
1. Profile to find bottlenecks
2. Run benchmarks
3. Apply optimizations

---

## ⚡ Performance Tips

### Parallel Execution
Commands like `/fix`, `/review`, and `/validate` run multiple agents in parallel for maximum efficiency. This can reduce execution time by 60-70%.

### Model Selection
- **Haiku**: Fast commands like `/hooks`, `/build`
- **Sonnet**: Most commands use Sonnet for balance
- **Opus**: Complex analysis like `/review`, `/explain`

### Argument Usage
Most commands accept arguments to focus their scope:
```bash
/test ProcessCheck       # Specific function
/fix internal/runner    # Specific package
/review feat/branch     # Specific branch
```

---

## 🔧 Troubleshooting

### Command Not Working?
1. Check if the command file exists in `.claude/commands/`
2. Verify frontmatter syntax is correct
3. Ensure required agents are available
4. Check argument format

### Slow Execution?
1. Use specific arguments to narrow scope
2. Check if parallel execution is enabled
3. Consider using lighter model for simple tasks

### Getting Help
```bash
> /help
# Shows all available commands

> /agents
# Manage sub-agents

> /status
# Check system status
```

---

## 📝 Creating Custom Commands

### Basic Structure
```markdown
---
allowed-tools: Task, Read, Edit
argument-hint: [parameter]
description: Brief description
model: sonnet
---

## Command Title

Your command prompt here with $ARGUMENTS placeholder.
```

### Best Practices
1. Use Task tool for parallel agent execution
2. Include bash commands with `!` for context
3. Reference files with `@` notation
4. Choose appropriate model for complexity
5. Provide clear argument hints

---

## 🔗 Related Documentation

- [Sub-Agents Documentation](../README.md#-sub-agents-team)
- [AGENTS.md](../.github/AGENTS.md) - Coding standards
- [CLAUDE.md](../.github/CLAUDE.md) - Claude-specific guide
- [Slash Commands Docs](https://docs.anthropic.com/en/docs/claude-code/slash-commands)

---

## 📊 Command Statistics

- **Total Commands**: 23
- **Core Commands**: 6
- **Workflow Commands**: 5
- **Specialized Commands**: 6
- **Namespaced Commands**: 6
- **Parallel Execution**: 8 commands
- **Multi-Agent**: 12 commands

These commands provide comprehensive coverage for all aspects of managing the go-pre-commit project, from development to release, with optimized workflows that leverage our specialized sub-agents team.
