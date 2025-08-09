---
allowed-tools: Task, Bash(make test:*), Bash(make lint:*), Bash(go test:*), Bash(git status:*)
description: Full validation suite with parallel agent execution
claude-sonnet-4-0
---

## ✅ Full Project Validation

### Pre-validation Status
- Git status: !`git status --short`
- Build status: !`make build 2>&1 | tail -5`

### Comprehensive Validation Suite

Run these agents **in parallel** using the Task tool for complete validation:

1. **go-standards-enforcer**: Validate Go standards compliance
2. **go-tester**: Run all tests with race detection and coverage
3. **go-formatter**: Check formatting and linting
4. **dependency-auditor**: Security scans (govulncheck, nancy, gitleaks)
5. **hook-specialist**: Verify pre-commit hooks functionality
6. **ci-guardian**: Validate CI/CD configuration

### Expected Results
- ✅ All tests passing with 90%+ coverage
- ✅ No linting violations
- ✅ No security vulnerabilities
- ✅ Standards compliant
- ✅ Hooks properly configured
- ✅ CI ready

Report any issues found with specific fixes required.
