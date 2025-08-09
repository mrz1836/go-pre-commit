---
allowed-tools: Task, Read, Grep, Glob
argument-hint: [package-or-file]
description: Validate Go standards compliance
claude-sonnet-4-0
---

## 📏 Validate Go Standards

### Target
- Validate: ${ARGUMENTS:-.}
- Files: !`find ${ARGUMENTS:-.} -name "*.go" -type f | wc -l` Go files

### Standards Validation

Use the **go-standards-enforcer agent** to check:

1. **Context-First Design**:
   - All cancellable operations have context.Context as first param
   - No context stored in structs
   - Proper context cancellation handling

2. **Interface Design**:
   - Small, focused interfaces
   - Accept interfaces, return concrete types
   - Proper -er suffix naming

3. **Error Handling**:
   - All errors checked
   - Errors wrapped with context
   - Using errors.Is() for comparisons

4. **Goroutine Discipline**:
   - Clear lifecycle management
   - Context-based cancellation
   - Proper synchronization

5. **No Global State**:
   - No package-level mutable variables
   - Dependency injection used
   - No init() functions

6. **Module Hygiene**:
   - Clean go.mod
   - Minimal dependencies
   - Proper versioning

Report all violations with specific fixes and AGENTS.md references.
