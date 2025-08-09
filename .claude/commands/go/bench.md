---
allowed-tools: Task, Bash(go test -bench:*), Bash(benchstat:*), Write
argument-hint: [package-or-function]
description: Run and analyze benchmarks
claude-sonnet-4-0
---

## 📊 Benchmark Analysis

### Target: ${ARGUMENTS:-.}

### Benchmark Workflow

Use the **performance-optimizer agent** with benchmark focus:

1. **Run Benchmarks**:
   ```bash
   go test -bench=${ARGUMENTS:-.} -benchmem -count=10 -run=^$ ./...
   ```

2. **Collect Metrics**:
   - Operations per second
   - Memory allocations per op
   - Bytes allocated per op
   - CPU profile data

3. **Compare Results**:
   - Run baseline benchmarks
   - Make improvements
   - Run new benchmarks
   - Use benchstat for comparison

4. **Analysis**:
   - Identify slowest operations
   - Find allocation hotspots
   - Detect performance regressions
   - Compare with previous runs

5. **Optimization Opportunities**:
   - Functions with high ns/op
   - Excessive allocations
   - Large memory usage
   - Inefficient algorithms

6. **Report**:
   - Benchmark results table
   - Performance trends
   - Optimization recommendations
   - Historical comparison

Save results to `bench-results-$(date +%Y%m%d).txt`
