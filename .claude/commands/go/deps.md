---
allowed-tools: Task, Bash(go list:*), Bash(go mod:*), Edit
argument-hint: [add|update|remove|analyze] [package]
description: Manage Go dependencies efficiently
model: haiku
---

## 📦 Dependency Management

### Action: ${ARGUMENTS:-analyze}

### Dependency Operations

Use the **dependency-auditor agent** for comprehensive dependency management:

1. **Analyze Dependencies**:
   - List all direct dependencies
   - Show dependency tree
   - Find unused dependencies
   - Check for updates available

2. **Add Dependency**:
   ```bash
   go get [package]
   go mod tidy
   ```
   - Verify license compatibility
   - Check security status
   - Review transitive dependencies

3. **Update Dependencies**:
   - Update specific: `go get -u [package]`
   - Update all: `go get -u ./...`
   - Update minor/patch only: `go get -u=patch ./...`
   - Run tests after updates
   - Check for breaking changes

4. **Remove Dependencies**:
   - Remove from code
   - Run `go mod tidy`
   - Verify no breakage

5. **Security Check**:
   - Run govulncheck
   - Check known CVEs
   - Review licenses

6. **Optimization**:
   - Remove unused dependencies
   - Replace heavy dependencies
   - Minimize dependency tree

Report dependency changes and security status.
