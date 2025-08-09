---
allowed-tools: Task, Bash(make:*), Read, Edit
argument-hint: [target]
description: Fix Makefile and build issues
model: haiku
---

## 🔨 Fix Build Issues

### Build Status
- Make targets: !`make help 2>/dev/null | head -10 || echo "Make not configured"`
- Build test: !`make build 2>&1 | tail -5`

### Build Troubleshooting

Use the **makefile-expert agent** to:

1. **Diagnose Build Issues**:
   - Identify failing targets: ${ARGUMENTS:-all}
   - Check Make syntax errors
   - Verify dependencies
   - Review variable definitions

2. **Common Fixes**:
   - Fix tab vs space issues
   - Correct variable expansion
   - Add missing PHONY declarations
   - Fix path issues
   - Resolve circular dependencies

3. **Optimize Makefile**:
   - Improve parallel execution
   - Add caching where appropriate
   - Optimize dependency resolution
   - Clean up redundant targets

4. **Add New Targets** (if needed):
   - Create well-documented targets
   - Follow project patterns
   - Include help text
   - Test thoroughly

5. **Integration**:
   - Ensure CI compatibility
   - Verify all required targets exist
   - Check cross-platform compatibility

Provide specific fixes for build issues.
