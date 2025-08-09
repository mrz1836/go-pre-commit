---
allowed-tools: Task, Read, Edit, MultiEdit, Grep, Glob
argument-hint: [target-package-or-file]
description: Refactor code to remove duplicates and improve structure
claude-sonnet-4-0
---

## ♻️ Refactor Code

### Refactoring Target
- Scope: ${ARGUMENTS:-.}
- Files: !`find ${ARGUMENTS:-.} -name "*.go" -type f | wc -l` Go files

### Refactoring Process

Ultrathink about code improvements, then:

1. **Analysis Phase**:
   - Identify code smells
   - Find duplicate patterns
   - Detect complex functions (cyclomatic complexity > 10)
   - Locate inconsistent patterns

2. **Refactoring Actions**:
   - **Extract Methods**: Break down complex functions
   - **Extract Interfaces**: Create abstractions for similar types
   - **Consolidate Duplicates**: Merge similar code paths
   - **Improve Naming**: Make code self-documenting
   - **Simplify Logic**: Reduce nesting, use early returns

3. **Validation** (use Task tool for parallel execution):
   - **go-standards-enforcer**: Ensure compliance
   - **go-tester**: Verify functionality preserved
   - **performance-optimizer**: Check performance impact

4. **Document Changes**:
   - List all refactoring performed
   - Explain rationale for changes
   - Note any behavior changes

Focus on improving maintainability without changing functionality.
