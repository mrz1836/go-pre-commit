---
allowed-tools: Task, Bash(go test:*), Bash(make test:*), Read, Write, Edit
argument-hint: [package-or-function-name]
description: Create or fix tests with go-tester agent
claude-sonnet-4-0
---

## 🧪 Create or Fix Tests

### Context
- Target: $ARGUMENTS
- Current coverage: !`go test -cover ./... 2>&1 | grep coverage || echo "No coverage data"`

### Task
Use the **go-tester agent** to:

1. **If creating new tests**: Write comprehensive test cases using testify, following AGENTS.md testing standards
2. **If fixing tests**: Debug failures, fix test logic, ensure deterministic behavior
3. Achieve 90%+ test coverage for the target code
4. Use table-driven tests with named test cases
5. Follow naming pattern: TestFunctionNameScenarioDescription

Ensure all tests are fast, deterministic, and follow the project's testing conventions strictly.
