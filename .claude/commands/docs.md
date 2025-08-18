---
allowed-tools: Task, Read, Edit, MultiEdit, Bash(git diff:*), Grep
argument-hint: [feature-or-file]
description: Update documentation for new or modified features
claude-sonnet-4-0
---

## 📚 Update Documentation

### Context
- Recent changes: !`git diff --name-only master...HEAD 2>/dev/null || git diff --name-only`
- Target: $ARGUMENTS

### Documentation Tasks

Use the **doc-maintainer agent** to:

1. **Update README.md** if new features or configuration changes
2. **Update code comments** for modified functions (godoc style)
3. **Verify documentation accuracy** - does the documented feature still exist?
4. **Update configuration docs** if .env.base/.env.custom or mage file changed
5. **Ensure Markdown formatting** follows AGENTS.md standards
6. **Update examples** to match current implementation

Additionally, check for:
- Outdated installation instructions
- Deprecated features still documented
- Missing documentation for new functionality
- Broken internal links or references

Maintain consistent tone and comprehensive coverage.
