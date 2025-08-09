---
allowed-tools: Read, Grep, Glob, Bash(go doc:*)
argument-hint: [function/package/feature]
description: Explain how code or features work
model: opus
---

## 📖 Explain Code or Feature

### Target
Explain: ${ARGUMENTS:-the overall project architecture}

### Analysis Process

Ultrathink deeply about the code structure, then:

1. **Locate relevant code**:
   - Find main implementation files
   - Identify related tests
   - Check documentation

2. **Analyze architecture**:
   - Purpose and responsibilities
   - Key components and their interactions
   - Data flow and dependencies
   - Design patterns used

3. **Explain clearly**:
   - High-level overview
   - Component breakdown
   - Step-by-step operation
   - Example usage scenarios
   - Edge cases and error handling

4. **Provide context**:
   - Why it was designed this way
   - Trade-offs made
   - Performance considerations
   - Future extensibility

Format the explanation for clarity with:
- Diagrams where helpful
- Code examples
- Practical use cases
