---
allowed-tools: Task, Read, Edit, MultiEdit, Grep, Glob
argument-hint: [package-or-directory]
description: Remove duplicate code and improve structure
model: sonnet
---

## 🧹 Clean Code and Remove Duplicates

### Analysis Scope
- Target: ${ARGUMENTS:-.}
- Files to analyze: !`find ${ARGUMENTS:-.} -name "*.go" -type f | head -20`

### Cleaning Tasks

Ultrathink about code quality, then:

1. **Identify duplicate code patterns**:
   - Similar functions that could be generalized
   - Repeated logic that could be extracted
   - Copy-pasted code blocks

2. **Refactor duplicates**:
   - Extract common functionality into shared functions
   - Create interfaces for similar behaviors
   - Use generics where appropriate (Go 1.18+)

3. **Improve structure**:
   - Group related functions
   - Ensure proper package organization
   - Remove dead code

4. **Validate changes** with:
   - **go-standards-enforcer agent**: Ensure refactoring follows standards
   - **go-tester agent**: Verify tests still pass

Focus on maintaining functionality while improving maintainability.
