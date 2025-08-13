#!/usr/bin/env python3
"""
JSON Validator Plugin for go-pre-commit
Validates and optionally formats JSON files
"""

import json
import sys
import os
from typing import Dict, List, Any

def validate_json_file(filepath: str, indent_size: int, sort_keys: bool) -> Dict[str, Any]:
    """Validate and optionally format a JSON file."""
    try:
        with open(filepath, 'r') as f:
            content = f.read()
            data = json.loads(content)

        # Check if formatting is needed
        formatted = json.dumps(data, indent=indent_size, sort_keys=sort_keys, ensure_ascii=False)
        if formatted != content.rstrip():
            # Formatting needed
            return {
                'valid': True,
                'needs_formatting': True,
                'formatted': formatted
            }

        return {
            'valid': True,
            'needs_formatting': False
        }

    except json.JSONDecodeError as e:
        return {
            'valid': False,
            'error': f"Invalid JSON at line {e.lineno}, column {e.colno}: {e.msg}"
        }
    except FileNotFoundError:
        return {
            'valid': False,
            'error': f"File not found: {filepath}"
        }
    except Exception as e:
        return {
            'valid': False,
            'error': str(e)
        }

def main():
    """Main entry point for the plugin."""
    # Read input from stdin
    try:
        input_data = json.loads(sys.stdin.read())
    except json.JSONDecodeError:
        response = {
            "success": False,
            "error": "Invalid input JSON",
            "suggestion": "Plugin expects valid JSON input via stdin"
        }
        print(json.dumps(response))
        sys.exit(1)

    # Extract command and files
    command = input_data.get('command', '')
    files = input_data.get('files', [])
    config = input_data.get('config', {})

    # Get configuration
    indent_size = int(os.environ.get('INDENT_SIZE', '2'))
    sort_keys = os.environ.get('SORT_KEYS', 'false').lower() == 'true'

    if command != 'check':
        response = {
            "success": False,
            "error": f"Unknown command: {command}",
            "suggestion": "Supported commands: check"
        }
        print(json.dumps(response))
        sys.exit(1)

    # Process files
    errors = []
    modified = []
    output_lines = []

    for filepath in files:
        if not filepath.endswith(('.json', '.jsonc')):
            continue

        result = validate_json_file(filepath, indent_size, sort_keys)

        if not result['valid']:
            errors.append(f"{filepath}: {result['error']}")
        elif result['needs_formatting']:
            output_lines.append(f"{filepath}: Needs formatting (indent={indent_size}, sort_keys={sort_keys})")
            modified.append(filepath)

    # Generate response
    if errors:
        response = {
            "success": False,
            "error": "\n".join(errors),
            "suggestion": "Fix JSON syntax errors in the listed files"
        }
        print(json.dumps(response))
        sys.exit(1)
    elif modified:
        response = {
            "success": False,
            "error": f"{len(modified)} file(s) need formatting",
            "suggestion": f"Run formatter with indent={indent_size} and sort_keys={sort_keys}",
            "modified": modified,
            "output": "\n".join(output_lines)
        }
        print(json.dumps(response))
        sys.exit(1)
    else:
        response = {
            "success": True,
            "output": f"All {len(files)} JSON file(s) are valid and properly formatted"
        }
        print(json.dumps(response))
        sys.exit(0)

if __name__ == "__main__":
    main()
