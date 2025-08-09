---
allowed-tools: Task, Bash(git:*), Edit, Bash(make test:*)
argument-hint: <issue-description>
description: Emergency hotfix workflow
model: opus
---

## 🚨 Emergency Hotfix

### Issue: $ARGUMENTS

### Hotfix Process

Execute rapid fix workflow with validation:

1. **Setup Hotfix Branch**:
   - Create branch: `hotfix/$ARGUMENTS`
   - Branch from latest production tag
   - Document issue clearly

2. **Rapid Diagnosis** (parallel with Task):
   - **code-reviewer**: Identify root cause
   - **go-tester**: Create failing test for bug
   - **dependency-auditor**: Check if security-related

3. **Implement Fix**:
   - Minimal change to fix issue
   - Avoid refactoring or improvements
   - Focus only on the bug

4. **Fast Validation** (parallel):
   - **go-tester**: Verify fix works
   - **go-formatter**: Quick format check
   - **go-standards-enforcer**: Ensure no violations
   - Run regression tests

5. **Expedited Review**:
   - **code-reviewer**: Security and correctness check
   - **ci-guardian**: Ensure CI passes
   - Document fix clearly

6. **Release Preparation**:
   - **release-coordinator**: Prepare patch release
   - Update CHANGELOG with fix
   - Create release notes

7. **Deployment Checklist**:
   - [ ] Bug reproduced
   - [ ] Fix implemented
   - [ ] Tests passing
   - [ ] No side effects
   - [ ] CI green
   - [ ] Ready for immediate release

Priority: Speed with safety. Fix must be minimal and thoroughly tested.
