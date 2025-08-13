---
allowed-tools: Task, Bash(git:*), Read, Edit, Bash(make:*)
argument-hint: <version>
description: Prepare and validate release
claude-sonnet-4-0
---

## 📦 Prepare Release

### Release Version: $ARGUMENTS

### Release Preparation

Use the **release-coordinator agent** to:

1. **Pre-Release Validation**:
   - All tests passing
   - Security scans clean
   - Documentation updated
   - No uncommitted changes

2. **Version Determination**:
   - Analyze changes since last release
   - Determine version type (major/minor/patch)
   - Validate against semantic versioning

3. **Update Metadata**:
   - Prepare CHANGELOG.md
   - Update version references in docs

4. **Release Checklist**:
   - [ ] Tests passing (go-tester)
   - [ ] Linting clean (go-formatter)
   - [ ] Security scans passed (dependency-auditor)
   - [ ] Documentation current (doc-maintainer)
   - [ ] PR merged to master
   - [ ] goreleaser config valid

5. **Generate Release**:
   - Create release notes
   - List breaking changes
   - Highlight new features
   - Document bug fixes

6. **Post-Release Tasks**:
   - Verify GitHub release
   - Check pkg.go.dev indexing
   - Test installation methods

Note: Only maintainers can create actual tags. This prepares everything needed.
