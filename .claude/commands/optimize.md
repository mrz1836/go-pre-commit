---
allowed-tools: Task, Bash(go test -bench:*), Bash(go tool pprof:*), Read, Edit
argument-hint: [package-or-function]
description: Performance optimization workflow
model: opus
---

## ⚡ Performance Optimization

### Target
- Optimize: ${ARGUMENTS:-.}
- Current benchmarks: !`go test -bench=. ./... 2>&1 | grep -E "Benchmark|ns/op" | head -10`

### Optimization Process

Use the **performance-optimizer agent** to:

1. **Baseline Measurement**:
   - Run benchmarks with -benchmem
   - Generate CPU and memory profiles
   - Identify hot paths

2. **Analysis**:
   - Find performance bottlenecks
   - Detect excessive allocations
   - Identify inefficient algorithms
   - Check for unnecessary goroutines

3. **Optimization Techniques**:
   - **Memory**: Pre-allocate slices, use sync.Pool, reduce allocations
   - **CPU**: Optimize loops, reduce function calls, improve algorithms
   - **I/O**: Batch operations, use buffering, optimize file access
   - **Concurrency**: Proper worker pools, channel buffering

4. **Validation**:
   - Compare before/after benchmarks
   - Verify functionality preserved
   - Check race conditions
   - Measure actual improvement

5. **Documentation**:
   - Document optimization changes
   - Explain performance gains
   - Note any trade-offs

Provide benchmark comparison showing improvements.
