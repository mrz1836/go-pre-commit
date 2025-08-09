---
allowed-tools: Task, Bash(go test:*), Bash(go tool pprof:*), Write
argument-hint: [cpu|mem|trace] [package]
description: Profile performance issues with pprof
model: sonnet
---

## 🔬 Performance Profiling

### Profile Type: ${ARGUMENTS:-cpu}

### Profiling Process

Use the **performance-optimizer agent** for deep profiling:

1. **CPU Profiling**:
   ```bash
   go test -cpuprofile=cpu.prof -bench=. ${ARGUMENTS##* }
   go tool pprof cpu.prof
   ```
   - Identify hot functions
   - Find CPU bottlenecks
   - Analyze call graphs

2. **Memory Profiling**:
   ```bash
   go test -memprofile=mem.prof -bench=. ${ARGUMENTS##* }
   go tool pprof -alloc_space mem.prof
   ```
   - Find memory leaks
   - Identify allocation hotspots
   - Analyze heap usage

3. **Execution Trace**:
   ```bash
   go test -trace=trace.out -bench=. ${ARGUMENTS##* }
   go tool trace trace.out
   ```
   - Visualize execution flow
   - Find goroutine contention
   - Identify blocking operations

4. **Analysis Tools**:
   - `top`: Show top functions
   - `list`: Show source code with costs
   - `web`: Generate call graph
   - `peek`: Show callers/callees

5. **Optimization Focus**:
   - Functions consuming most CPU/memory
   - Unnecessary allocations
   - Inefficient algorithms
   - Goroutine overhead

6. **Report**:
   - Top 10 hot functions
   - Memory allocation patterns
   - Optimization recommendations
   - Before/after comparison

Generate profile visualization and save analysis.
