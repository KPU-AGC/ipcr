# Performance validation

The engine has two performance-oriented safety nets:

1. Ordinary tests compare the optimized seeded engine against the deliberately
   slow brute-force oracle. These are correctness checks and should run in CI.
2. Go benchmarks measure the expected runtime shape. They are intentionally not
   hard pass/fail thresholds because wall-clock results depend on hardware,
   compiler version, CPU frequency scaling, and filesystem/cache state.

Run the focused engine benchmarks from the repository root:

```bash
make bench-engine
```

Equivalent direct command:

```bash
go test -run '^$' -bench 'Benchmark(CompilePanel|BuildSeedPatterns|SimulateCompiled)' -benchmem ./core/engine
```

Useful subsets:

```bash
# Panel compilation, seed expansion, and automaton construction.
go test -run '^$' -bench 'Benchmark(CompilePanel|BuildSeedPatterns)' -benchmem ./core/engine

# Optimized scan paths only.
go test -run '^$' -bench 'BenchmarkSimulateCompiled' -benchmem ./core/engine

# Old exhaustive mismatch oracle, intentionally slower and run on a small fixture.
go test -run '^$' -bench 'BruteForce' -benchmem ./core/engine
```

For approximate mismatch work, watch the scaling shape more than any single time.
The optimized `SimulateCompiledMismatch1Panel16` benchmark should be dominated by
one reference scan plus sparse full-primer verification. The brute-force oracle
benchmark retains the older exhaustive per-primer-orientation behavior and is a
local comparison point, not a production path.

For external comparisons such as `ipcress`, keep the harness separate from these
microbenchmarks. Normalize tool-specific sequence IDs and zero-mismatch fields,
then report exact matching separately from unrestricted mismatch matching.
