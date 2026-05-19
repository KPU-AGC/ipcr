#!/usr/bin/env sh
set -eu

# Run focused engine benchmarks for the seeded/approximate-seed hot path.
# Environment knobs:
#   COUNT=10
#   BENCHTIME=500ms
#   BENCH='BenchmarkSimulateCompiled'

COUNT=${COUNT:-5}
BENCHTIME=${BENCHTIME:-200ms}
BENCH=${BENCH:-'Benchmark(CompilePanel|BuildSeedPatterns|SimulateCompiled|SimulateBatchBruteForce)'}

cd "$(dirname "$0")/.."

(
	cd core
	go test ./engine \
		-run '^$' \
		-bench "$BENCH" \
		-benchtime "$BENCHTIME" \
		-count "$COUNT" \
		-benchmem
)
