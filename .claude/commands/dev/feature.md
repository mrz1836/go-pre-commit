---
allowed-tools: Task, Bash(git:*), Write, Edit
argument-hint: <feature-name>
description: Start new feature development workflow
claude-sonnet-4-0
---

## 🚀 Start New Feature Development

### Feature: $ARGUMENTS

### Feature Development Workflow

Orchestrate the complete feature development process:

1. **Setup Feature Branch**:
   - Create branch: `feat/$ARGUMENTS`
   - Ensure clean working directory
   - Pull latest from master

2. **Planning Phase** (use Task for parallel execution):
   - **doc-maintainer**: Create initial documentation plan
   - **go-standards-enforcer**: Review design for standards compliance
   - Generate PRD if complex feature

3. **Implementation Phase**:
   - Write implementation code
   - Follow TDD approach with **go-tester**
   - Ensure standards with **go-standards-enforcer**
   - Optimize if needed with **performance-optimizer**

4. **Validation Phase** (parallel):
   - **go-tester**: Write comprehensive tests
   - **go-formatter**: Format and lint code
   - **code-reviewer**: Review implementation
   - **doc-maintainer**: Update documentation

5. **Integration Phase**:
   - **hook-specialist**: Test with pre-commit hooks
   - **ci-guardian**: Verify CI compatibility
   - **pr-orchestrator**: Prepare for PR

6. **Checklist**:
   - [ ] Feature implemented
   - [ ] Tests written (90%+ coverage)
   - [ ] Documentation updated
   - [ ] Standards validated
   - [ ] Performance acceptable
   - [ ] PR ready

Create tracking issue and initial structure for the feature.
