#!/bin/bash

# Multi-Validator Composite Plugin
# Runs multiple validation steps in sequence

set -e

# Read JSON input
INPUT=$(cat)
COMMAND=$(echo "$INPUT" | grep -o '"command":"[^"]*' | cut -d'"' -f4)
FILES=$(echo "$INPUT" | grep -o '"files":\[[^]]*\]' | sed 's/"files":\[//' | sed 's/\]//' | sed 's/"//g' | sed 's/,/ /g')

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

# Configuration
ENABLE_SPELLING=${ENABLE_SPELLING:-true}
ENABLE_LINKS=${ENABLE_LINKS:-true}
ENABLE_COMPLEXITY=${ENABLE_COMPLEXITY:-true}

# Track overall results
TOTAL_ISSUES=0
ALL_ERRORS=""
ALL_OUTPUT=""

# Step 1: Check file sizes
echo "Step 1: Checking file sizes..." >&2
for file in $FILES; do
    if [ -f "$file" ]; then
        SIZE=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "0")
        if [ "$SIZE" -gt 1048576 ]; then # 1MB
            ALL_OUTPUT="${ALL_OUTPUT}$file: File too large ($(($SIZE / 1024))KB)\n"
            TOTAL_ISSUES=$((TOTAL_ISSUES + 1))
        fi
    fi
done

# Step 2: Check for common spelling mistakes (if enabled)
if [ "$ENABLE_SPELLING" = "true" ]; then
    echo "Step 2: Checking spelling..." >&2
    COMMON_TYPOS="teh|recieve|occured|seperate|untill|wich"
    for file in $FILES; do
        if [[ "$file" == *.md || "$file" == *.txt || "$file" == *.go ]]; then
            if grep -qE "$COMMON_TYPOS" "$file" 2>/dev/null; then
                ALL_OUTPUT="${ALL_OUTPUT}$file: Contains common spelling mistakes\n"
                TOTAL_ISSUES=$((TOTAL_ISSUES + 1))
            fi
        fi
    done
fi

# Step 3: Check for broken links in markdown (if enabled)
if [ "$ENABLE_LINKS" = "true" ]; then
    echo "Step 3: Checking links in markdown files..." >&2
    for file in $FILES; do
        if [[ "$file" == *.md ]]; then
            # Simple check for obviously broken links
            if grep -E '\[.*\]\(\s*\)' "$file" 2>/dev/null | grep -v "^\s*#" > /dev/null; then
                ALL_OUTPUT="${ALL_OUTPUT}$file: Contains empty markdown links\n"
                TOTAL_ISSUES=$((TOTAL_ISSUES + 1))
            fi
        fi
    done
fi

# Step 4: Check code complexity (if enabled)
if [ "$ENABLE_COMPLEXITY" = "true" ]; then
    echo "Step 4: Checking code complexity..." >&2
    for file in $FILES; do
        if [[ "$file" == *.go ]]; then
            # Simple complexity check: functions with too many lines
            if [ -f "$file" ]; then
                # Count lines in functions (simplified)
                FUNC_LINES=$(awk '/^func / {count=0; next} /^}/ {if(count>50) print count; count=0; next} {count++}' "$file" 2>/dev/null | head -1)
                if [ -n "$FUNC_LINES" ]; then
                    ALL_OUTPUT="${ALL_OUTPUT}$file: Contains complex functions (>50 lines)\n"
                    TOTAL_ISSUES=$((TOTAL_ISSUES + 1))
                fi
            fi
        fi
    done
fi

# Generate final response
if [ $TOTAL_ISSUES -gt 0 ]; then
    cat <<EOF
{
  "success": false,
  "error": "Found $TOTAL_ISSUES validation issue(s)",
  "suggestion": "Review and fix the reported issues",
  "output": "$(echo -e "$ALL_OUTPUT" | sed 's/$/\\n/' | tr -d '\n' | sed 's/\\n$//')"
}
EOF
    exit 1
else
    STEPS_RUN=""
    [ "$ENABLE_SPELLING" = "true" ] && STEPS_RUN="${STEPS_RUN}spelling, "
    [ "$ENABLE_LINKS" = "true" ] && STEPS_RUN="${STEPS_RUN}links, "
    [ "$ENABLE_COMPLEXITY" = "true" ] && STEPS_RUN="${STEPS_RUN}complexity, "
    STEPS_RUN=${STEPS_RUN%, }

    cat <<EOF
{
  "success": true,
  "output": "All validation checks passed (${STEPS_RUN})"
}
EOF
    exit 0
fi
