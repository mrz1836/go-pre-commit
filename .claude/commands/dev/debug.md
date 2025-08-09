---
allowed-tools: Task, Read, Bash(go test:*), Bash(go run:*), Grep
argument-hint: <issue-description>
description: Debug complex issues with systematic approach
model: opus
---

## 🔍 Debug Complex Issue

### Issue: $ARGUMENTS

### Systematic Debugging Process

Ultrathink about the problem, then debug systematically:

1. **Information Gathering**:
   - Error messages and stack traces
   - Recent changes that might be related
   - Environment and configuration
   - Reproduction steps

2. **Multi-Agent Analysis** (parallel with Task):
   - **code-reviewer**: Analyze code for logical errors
   - **go-tester**: Check test coverage of affected area
   - **go-standards-enforcer**: Look for standards violations
   - **performance-optimizer**: Check for performance issues

3. **Hypothesis Formation**:
   - List possible causes
   - Rank by probability
   - Design tests for each hypothesis

4. **Systematic Testing**:
   - Add debug logging at key points
   - Use debugger or print statements
   - Isolate the problem area
   - Create minimal reproduction

5. **Root Cause Analysis**:
   - Identify the exact cause
   - Understand why it happened
   - Determine impact scope

6. **Solution Development**:
   - Design proper fix
   - Consider edge cases
   - Prevent similar issues

7. **Validation**:
   - Verify fix resolves issue
   - Check for regressions
   - Add tests to prevent recurrence

8. **Documentation**:
   - Document the issue and fix
   - Add comments if code is complex
   - Update troubleshooting guide

Provide detailed analysis with root cause and recommended fix.
