---
allowed-tools: Task, Bash(git diff:*), Bash(git status:*), Bash(make test:*), Bash(make lint:*)
argument-hint: [specific-file-or-package]
description: Fix test errors and linting issues in parallel
model: sonnet
---

## 🔧 Fix Test and Linting Issues

### Current Status
- Git changes: !`git diff --name-only`
- Test status: !`make test 2>&1 | tail -20`

### Task
Fix all test failures and linting issues in the codebase. Work efficiently by running multiple agents in parallel:

1. **go-tester agent**: Fix any failing tests, ensure proper testify usage, achieve 90%+ coverage
2. **go-formatter agent**: Fix all linting issues, run fumpt, ensure proper formatting
3. **go-standards-enforcer agent**: Validate Go standards compliance

Focus on: $ARGUMENTS

Use the Task tool to delegate to these specialized agents in parallel for maximum efficiency. Each agent should work on their specific domain and report back results.
