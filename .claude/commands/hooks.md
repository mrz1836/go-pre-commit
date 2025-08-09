---
allowed-tools: Task, Bash(go-pre-commit:*), Read, Edit
description: Fix pre-commit hook issues
model: haiku
---

## 🪝 Fix Pre-commit Hook Issues

### Hook Status
- Installation: !`go-pre-commit status 2>&1 | head -10`
- Git hooks: !`ls -la .git/hooks/pre-commit 2>/dev/null || echo "No pre-commit hook installed"`

### Hook Troubleshooting

Use the **hook-specialist agent** to:

1. **Diagnose Issues**:
   - Check installation status
   - Verify hook configuration
   - Test hook execution
   - Review .env.shared settings

2. **Common Fixes**:
   - Reinstall hooks if missing
   - Fix configuration issues
   - Resolve Make target problems
   - Adjust parallel workers
   - Fix timeout issues

3. **Test Hook Functionality**:
   - Create test changes
   - Verify checks run
   - Confirm auto-staging works
   - Test specific checks

4. **Performance Tuning**:
   - Optimize parallel execution
   - Adjust timeouts
   - Enable/disable specific checks
   - Configure fail-fast mode

5. **Integration**:
   - Ensure CI compatibility
   - Verify Make targets exist
   - Check environment variables

Provide specific fixes for any hook-related issues found.
