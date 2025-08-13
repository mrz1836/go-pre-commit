#!/bin/bash

# TODO/FIXME/HACK Comment Checker Plugin for go-pre-commit
# This plugin checks for TODO, FIXME, and HACK comments in code files

set -e

# Read JSON input from stdin
INPUT=$(cat)

# Parse JSON using basic shell tools (for portability)
# In production, use jq if available
COMMAND=$(echo "$INPUT" | grep -o '"command":"[^"]*' | cut -d'"' -f4)
FILES=$(echo "$INPUT" | grep -o '"files":\[[^]]*\]' | sed 's/"files":\[//' | sed 's/\]//' | sed 's/"//g' | sed 's/,/ /g')

# Get configuration from environment
ALLOW_TODOS=${ALLOW_TODOS:-false}
MAX_TODOS=${MAX_TODOS:-10}

# Initialize counters
TOTAL_TODOS=0
FOUND_ISSUES=false
ERROR_MSG=""
OUTPUT=""

# Function to check a single file
check_file() {
    local file="$1"
    local count=0

    if [ ! -f "$file" ]; then
        return 0
    fi

    # Search for TODO, FIXME, HACK comments
    while IFS= read -r line; do
        if echo "$line" | grep -qE 'TODO|FIXME|HACK'; then
            LINE_NUM=$(echo "$line" | cut -d: -f1)
            CONTENT=$(echo "$line" | cut -d: -f2-)
            OUTPUT="${OUTPUT}${file}:${LINE_NUM}: ${CONTENT}\n"
            count=$((count + 1))
            TOTAL_TODOS=$((TOTAL_TODOS + 1))
        fi
    done < <(grep -nE 'TODO|FIXME|HACK' "$file" 2>/dev/null || true)

    return 0
}

# Process command
case "$COMMAND" in
    "check")
        # Check each file
        for file in $FILES; do
            check_file "$file"
        done

        # Determine success based on configuration
        if [ "$ALLOW_TODOS" = "false" ] && [ $TOTAL_TODOS -gt 0 ]; then
            FOUND_ISSUES=true
            ERROR_MSG="Found $TOTAL_TODOS TODO/FIXME/HACK comments that need to be resolved"
        elif [ $TOTAL_TODOS -gt $MAX_TODOS ]; then
            FOUND_ISSUES=true
            ERROR_MSG="Found $TOTAL_TODOS TODO/FIXME/HACK comments, exceeding limit of $MAX_TODOS"
        fi

        # Generate JSON response
        if [ "$FOUND_ISSUES" = true ]; then
            cat <<EOF
{
  "success": false,
  "error": "$ERROR_MSG",
  "suggestion": "Resolve TODO/FIXME/HACK comments or set GO_PRE_COMMIT_ALLOW_TODOS=true",
  "output": "$(echo -e "$OUTPUT" | sed 's/$/\\n/' | tr -d '\n' | sed 's/\\n$//')"
}
EOF
            exit 1
        else
            if [ $TOTAL_TODOS -gt 0 ]; then
                cat <<EOF
{
  "success": true,
  "output": "Found $TOTAL_TODOS TODO/FIXME/HACK comments (within limit of $MAX_TODOS)"
}
EOF
            else
                cat <<EOF
{
  "success": true,
  "output": "No TODO/FIXME/HACK comments found"
}
EOF
            fi
        fi
        ;;
    *)
        cat <<EOF
{
  "success": false,
  "error": "Unknown command: $COMMAND",
  "suggestion": "Supported commands: check"
}
EOF
        exit 1
        ;;
esac
