# 🔌 go-pre-commit Plugin Examples

This directory contains example plugins demonstrating how to create custom pre-commit hooks for `go-pre-commit`.

## 📚 Plugin Types

### 1. **Shell Script Plugin** (`shell-plugin/`)
Simple shell script that checks for TODO comments in code.

### 2. **Python Plugin** (`python-plugin/`)
Python script that validates JSON files for proper formatting.

### 3. **Go Plugin** (`go-plugin/`)
Compiled Go binary that checks for license headers in source files.

### 4. **Docker Plugin** (`docker-plugin/`)
Docker-based plugin for running security scans in an isolated environment.

### 5. **Composite Plugin** (`composite-plugin/`)
Multi-step plugin that combines different tools for comprehensive checks.

## 🚀 Quick Start

### Using an Example Plugin

1. Copy the plugin directory to your project's `.pre-commit-plugins/` folder:
```bash
cp -r examples/shell-plugin .pre-commit-plugins/
```

2. Enable plugins in your configuration:
```bash
# Add to .env.custom to override defaults, or modify .env.base directly
GO_PRE_COMMIT_ENABLE_PLUGINS=true
```

3. Run pre-commit checks:
```bash
go-pre-commit run
```

## 📝 Creating Your Own Plugin

### Plugin Structure

Every plugin needs:
1. A manifest file (`plugin.yaml` or `plugin.json`)
2. An executable (script or binary)
3. Optional configuration files

### Manifest Format

```yaml
name: my-custom-check
version: 1.0.0
description: Description of what your plugin does
executable: ./check.sh
file_patterns:
  - "*.go"
  - "*.js"
timeout: 30s
category: linting
requires_files: true
environment:
  MY_SETTING: "${GO_PRE_COMMIT_MY_SETTING}"
```

### Plugin Protocol

Plugins communicate via JSON over stdin/stdout:

**Input (stdin):**
```json
{
  "command": "check",
  "files": ["file1.go", "file2.go"],
  "config": {
    "key": "value"
  }
}
```

**Output (stdout):**
```json
{
  "success": true,
  "error": "",
  "suggestion": "",
  "modified": [],
  "output": "Check completed successfully"
}
```

### Exit Codes

- `0`: Success
- `1`: Check failed (but not an error)
- `2+`: Error occurred

## 🧪 Testing Your Plugin

Test your plugin locally:

```bash
# Direct execution
echo '{"command":"check","files":["test.go"]}' | ./my-plugin/check.sh

# Via go-pre-commit
go-pre-commit run --only my-custom-check
```

## 📚 Best Practices

1. **Keep it Fast**: Plugins should complete within 30 seconds
2. **Be Specific**: Use file patterns to only process relevant files
3. **Clear Messages**: Provide helpful error messages and suggestions
4. **Idempotent**: Running the check multiple times should produce same results
5. **Exit Gracefully**: Handle errors and timeouts properly
6. **Document Well**: Include clear documentation and examples

## 🔒 Security Considerations

- Plugins run with the same permissions as `go-pre-commit`
- Use `read_only: true` in manifest if plugin doesn't need write access
- Validate all inputs before processing
- Avoid executing arbitrary commands from user input

## 📖 Examples Index

| Plugin | Language | Description | Complexity |
|--------|----------|-------------|------------|
| [shell-plugin](shell-plugin/) | Shell | TODO comment checker | Simple |
| [python-plugin](python-plugin/) | Python | JSON validator | Medium |
| [go-plugin](go-plugin/) | Go | License header checker | Medium |
| [docker-plugin](docker-plugin/) | Docker | Security scanner | Advanced |
| [composite-plugin](composite-plugin/) | Mixed | Multi-step validator | Advanced |

## 🤝 Contributing

Have a useful plugin? Consider:
1. Adding it as an example here
2. Publishing it as a standalone repository
3. Contributing it to the core if widely useful

## 📄 License

All example plugins are MIT licensed and free to use as templates for your own plugins.
