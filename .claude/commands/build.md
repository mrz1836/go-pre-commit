---
allowed-tools: Task, Bash(magex:*), Read, Edit
argument-hint: [target]
description: Fix MAGE-X and build issues
model: haiku
---

## 🔨 Fix Build Issues

### Build Status
- MAGE-X commands: !`magex -l 2>/dev/null | head -10 || echo "MAGE-X not configured"`
- Build test: !`magex build 2>&1 | tail -5`

### Build Troubleshooting

Use the **magex-expert** agent to:

1. **Diagnose Build Issues**:
   - Identify failing commands: ${ARGUMENTS:-all}
   - Check .mage.yaml syntax
   - Verify dependencies
   - Review configuration

2. **Common Fixes**:
   - Fix YAML formatting issues
   - Correct build configuration
   - Update dependency versions
   - Fix path issues
   - Resolve command conflicts

3. **Optimize MAGE-X Config**:
   - Improve build performance
   - Configure caching
   - Optimize command execution
   - Clean up unused commands

4. **Add New Commands** (if needed):
   - Create well-documented commands
   - Follow project patterns
   - Include descriptions
   - Test thoroughly

5. **Integration**:
   - Ensure CI compatibility
   - Verify all required commands exist
   - Check cross-platform compatibility

Provide specific fixes for build issues.
