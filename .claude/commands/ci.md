---
allowed-tools: Task, Bash(gh:*), Read, Edit, Grep
argument-hint: [workflow-name-or-run-id]
description: Diagnose and fix CI/CD issues with ci-guardian
model: sonnet
---

## 🔧 Diagnose CI Issues

### CI Status
- Recent runs: !`gh run list --limit 5 2>/dev/null || echo "GitHub CLI not configured"`
- Workflow files: !`ls -la .github/workflows/*.yml | tail -5`

### Diagnostic Process

Use the **ci-guardian agent** to:

1. **Analyze failing workflows**: ${ARGUMENTS:-all}
2. **Identify root causes**:
   - Test failures
   - Linting violations
   - Security scan issues
   - Timeout problems
   - Configuration errors

3. **Coordinate fixes** with specialized agents:
   - go-tester for test failures
   - go-formatter for linting issues
   - dependency-auditor for security scans
   - makefile-expert for build problems

4. **Optimize performance**:
   - Review caching strategy
   - Check parallel execution
   - Validate runner selection

5. **Verify fixes**:
   - Test locally when possible
   - Validate workflow syntax
   - Check environment variables in .env.shared

Provide specific fixes for identified issues.
