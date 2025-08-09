---
allowed-tools: Task, Bash(git diff:*), Bash(git log:*), Read, Grep
argument-hint: [branch-or-commit]
description: Comprehensive code review with multiple agents in parallel
model: opus
---

## 🔍 Comprehensive Code Review

### Changes to Review
- Diff: !`git diff ${ARGUMENTS:-HEAD} 2>/dev/null || git diff`
- Changed files: !`git diff --name-only ${ARGUMENTS:-HEAD} 2>/dev/null || git diff --name-only`

### Multi-Agent Review Process

Run these specialized agents **in parallel** using the Task tool for comprehensive review:

1. **code-reviewer agent**: Security vulnerabilities, error handling, performance, maintainability
2. **go-standards-enforcer agent**: AGENTS.md compliance, context-first design, interface patterns
3. **performance-optimizer agent**: Performance implications, memory allocations, bottlenecks
4. **dependency-auditor agent**: Dependency changes, security vulnerabilities, license compliance

Provide a consolidated report with:
- 🚨 Critical issues (must fix)
- ⚠️ Warnings (should fix)
- 💡 Suggestions (consider)
- ✅ Positive observations

Focus on actionable feedback with specific examples and fixes.
