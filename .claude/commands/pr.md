---
allowed-tools: Task, Bash(git:*), Read, Edit
argument-hint: [pr-title]
description: Complete pull request workflow with pr-orchestrator
claude-sonnet-4-0
---

## 🚀 Create Pull Request

### Current Branch Status
- Branch: !`git branch --show-current`
- Changes: !`git diff --stat master...HEAD 2>/dev/null || git diff --stat`
- Commits: !`git log --oneline master...HEAD 2>/dev/null | head -10`

### PR Workflow

Use the **pr-orchestrator agent** to:

1. **Validate branch naming** follows conventions (feat/, fix/, docs/, etc.)
2. **Run validation checks** in parallel:
   - Format code (go-formatter)
   - Run tests (go-tester)
   - Check standards (go-standards-enforcer)
   - Review changes (code-reviewer)

3. **Prepare PR**:
   - Create descriptive title: ${ARGUMENTS:-Auto-generated PR title}
   - Write comprehensive description (What/Why/Testing/Impact)
   - Apply appropriate labels
   - Ensure CI readiness

4. **Final checks**:
   - No merge conflicts
   - All tests passing
   - Documentation updated
   - Changelog prepared (if needed)

The pr-orchestrator will coordinate all necessary agents for a complete PR.
