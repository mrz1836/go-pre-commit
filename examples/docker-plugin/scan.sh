#!/bin/bash

# Security Scanner Plugin using Docker
# Runs security scans in an isolated container

set -e

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    cat <<EOF
{
  "success": false,
  "error": "Docker is not installed or not in PATH",
  "suggestion": "Install Docker from https://docs.docker.com/get-docker/"
}
EOF
    exit 1
fi

# Read JSON input
INPUT=$(cat)
COMMAND=$(echo "$INPUT" | grep -o '"command":"[^"]*' | cut -d'"' -f4)

if [ "$COMMAND" != "check" ]; then
    cat <<EOF
{
  "success": false,
  "error": "Unknown command: $COMMAND",
  "suggestion": "Supported commands: check"
}
EOF
    exit 1
fi

# Get configuration
SCAN_TYPE=${SCAN_TYPE:-basic}
FAIL_ON_HIGH=${FAIL_ON_HIGH:-true}

# Build Docker image if not exists
DOCKER_IMAGE="go-pre-commit-security-scanner:latest"
if [[ "$(docker images -q $DOCKER_IMAGE 2> /dev/null)" == "" ]]; then
    # Create Dockerfile inline
    cat > /tmp/scanner.Dockerfile <<'DOCKERFILE'
FROM alpine:latest
RUN apk add --no-cache bash grep sed
WORKDIR /scan
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
DOCKERFILE

    # Create entrypoint script
    cat > /tmp/entrypoint.sh <<'ENTRYPOINT'
#!/bin/bash
# Simple security scanner simulation
echo "Running security scan..."

# Simulate scanning for common issues
ISSUES=0
HIGH_SEVERITY=0

# Check for hardcoded secrets (simulation)
if grep -r "password\s*=\s*['\"]" /scan 2>/dev/null | grep -v "password\s*=\s*['\"].*\${" > /dev/null; then
    echo "CRITICAL: Hardcoded passwords detected"
    HIGH_SEVERITY=$((HIGH_SEVERITY + 1))
    ISSUES=$((ISSUES + 1))
fi

if grep -r "api[_-]key\s*=\s*['\"]" /scan 2>/dev/null | grep -v "api[_-]key\s*=\s*['\"].*\${" > /dev/null; then
    echo "HIGH: Hardcoded API keys detected"
    HIGH_SEVERITY=$((HIGH_SEVERITY + 1))
    ISSUES=$((ISSUES + 1))
fi

# Check for SQL injection risks (basic check)
if grep -r "query.*+.*user_input" /scan 2>/dev/null > /dev/null; then
    echo "MEDIUM: Potential SQL injection vulnerability"
    ISSUES=$((ISSUES + 1))
fi

# Output results
if [ $HIGH_SEVERITY -gt 0 ]; then
    echo "Found $HIGH_SEVERITY high severity issue(s)"
    exit 2
elif [ $ISSUES -gt 0 ]; then
    echo "Found $ISSUES issue(s)"
    exit 1
else
    echo "No security issues found"
    exit 0
fi
ENTRYPOINT

    # Build image
    docker build -f /tmp/scanner.Dockerfile -t $DOCKER_IMAGE /tmp 2>/dev/null
fi

# Run security scan in Docker
SCAN_OUTPUT=$(docker run --rm -v "$(pwd):/scan:ro" $DOCKER_IMAGE 2>&1)
SCAN_EXIT_CODE=$?

# Parse results and generate response
if [ $SCAN_EXIT_CODE -eq 2 ] && [ "$FAIL_ON_HIGH" = "true" ]; then
    cat <<EOF
{
  "success": false,
  "error": "High severity security issues detected",
  "suggestion": "Fix critical security issues before committing",
  "output": "$SCAN_OUTPUT"
}
EOF
    exit 1
elif [ $SCAN_EXIT_CODE -ne 0 ]; then
    cat <<EOF
{
  "success": false,
  "error": "Security issues detected",
  "suggestion": "Review and fix security issues",
  "output": "$SCAN_OUTPUT"
}
EOF
    exit 1
else
    cat <<EOF
{
  "success": true,
  "output": "$SCAN_OUTPUT"
}
EOF
    exit 0
fi
