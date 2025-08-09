---
allowed-tools: Task, Bash(go list:*), Bash(govulncheck:*), Bash(gitleaks:*)
description: Security audit with vulnerability scanning
model: sonnet
---

## 🔒 Security Audit

### Current Security Status
- Dependencies: !`go list -m all | wc -l` total modules
- Go version: !`go version`

### Comprehensive Security Audit

Use the **dependency-auditor agent** to perform:

1. **Vulnerability Scanning**:
   - Run govulncheck for Go vulnerabilities
   - Execute nancy for dependency CVEs
   - Use gitleaks for secret detection

2. **Dependency Analysis**:
   - Check for outdated dependencies
   - Identify unused dependencies
   - Review license compliance

3. **Code Security Review**:
   - SQL injection risks
   - Command injection vulnerabilities
   - Path traversal issues
   - Insecure random number generation
   - Hardcoded credentials

4. **Security Best Practices**:
   - Verify no secrets in code
   - Check for proper input validation
   - Ensure secure error handling
   - Validate authentication/authorization

5. **Generate Report**:
   - List all vulnerabilities by severity
   - Provide remediation steps
   - Update NANCY_EXCLUDES if needed
   - Document any accepted risks

Output a comprehensive security report with actionable fixes.
