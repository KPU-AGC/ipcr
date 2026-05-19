package engine

import (
	"fmt"
	"ipcr-core/primer"
	"testing"
)

var (
	benchmarkProductsSink     []Product
	benchmarkCompiledSink     *CompiledPanel
	benchmarkSeedPatternsSink []SeedPattern
)

type engineBenchFixture struct {
	seq   []byte
	pairs []primer.Pair
}

func makeEngineBenchFixture(pairCount, genomeLen int, mutateForward, referenceN bool) engineBenchFixture {
	const productLen = 180
	const primerLen = 20

	if pairCount < 1 {
		pairCount = 1
	}
	minLen := 128
	neededLen := 256 + pairCount*256 + productLen
	if genomeLen < neededLen {
		genomeLen = neededLen
	}

	seq := benchDNA(genomeLen, 0x5eed1234)
	pairs := make([]primer.Pair, 0, pairCount)
	for i := 0; i < pairCount; i++ {
		fwd := benchPrimer(i*2, primerLen)
		rev := benchPrimer(i*2+1, primerLen)
		pair := primer.Pair{
			ID:         fmt.Sprintf("bench_%03d", i),
			Forward:    fwd,
			Reverse:    rev,
			MinProduct: minLen,
			MaxProduct: productLen + 32,
		}
		pairs = append(pairs, pair)

		start := 128 + i*256
		plantedFwd := []byte(fwd)
		if mutateForward {
			// With a 20-mer primer and SeedLen=12, the engine normally chooses a
			// right-hand seed. Put the mismatch inside that seed span so benchmark
			// runs exercise approximate seed-neighborhood generation.
			plantedFwd[10] = differentBase(plantedFwd[10])
		}
		if referenceN {
			// Reference N is a hard mismatch. Placing it inside the seed span exercises
			// non-ACGT halo recovery rather than treating N as a wildcard.
			plantedFwd[11] = 'N'
		}
		copy(seq[start:], plantedFwd)
		copy(seq[start+productLen-len(rev):], primer.RevComp([]byte(rev)))
	}

	return engineBenchFixture{seq: seq, pairs: pairs}
}

func benchDNA(n int, seed uint32) []byte {
	seq := make([]byte, n)
	x := seed
	alphabet := [4]byte{'A', 'C', 'G', 'T'}
	for i := range seq {
		x = x*1664525 + 1013904223
		seq[i] = alphabet[(x>>30)&3]
	}
	return seq
}

func benchPrimer(idx, n int) string {
	buf := make([]byte, n)
	x := uint32(0x9e3779b9 ^ (idx * 0x45d9f3b))
	alphabet := [4]byte{'A', 'C', 'G', 'T'}
	for i := range buf {
		x = x*1103515245 + 12345 + uint32(i*97)
		buf[i] = alphabet[(x>>29)&3]
	}

	// Avoid pathological low-complexity seeds in deterministic fixtures.
	buf[0] = alphabet[idx&3]
	buf[1] = alphabet[(idx+1)&3]
	buf[2] = alphabet[(idx+2)&3]
	buf[n-1] = alphabet[(idx+3)&3]
	return string(buf)
}

func differentBase(b byte) byte {
	switch b {
	case 'A':
		return 'C'
	case 'C':
		return 'G'
	case 'G':
		return 'T'
	default:
		return 'A'
	}
}

func BenchmarkCompilePanelExact64(b *testing.B) {
	fixture := makeEngineBenchFixture(64, 100000, false, false)
	eng := New(Config{MaxMM: 0, TerminalWindow: 0, MinLen: 100, MaxLen: 240, SeedLen: 12})
	b.ReportAllocs()

	var cp *CompiledPanel
	for i := 0; i < b.N; i++ {
		cp = eng.CompilePanel(fixture.pairs)
	}
	benchmarkCompiledSink = cp
}

func BenchmarkCompilePanelMismatch1Panel64(b *testing.B) {
	fixture := makeEngineBenchFixture(64, 100000, true, false)
	eng := New(Config{MaxMM: 1, TerminalWindow: 0, MinLen: 100, MaxLen: 240, SeedLen: 12})
	b.ReportAllocs()

	var cp *CompiledPanel
	for i := 0; i < b.N; i++ {
		cp = eng.CompilePanel(fixture.pairs)
	}
	benchmarkCompiledSink = cp
}

func BenchmarkCompilePanelMismatch2Panel16(b *testing.B) {
	fixture := makeEngineBenchFixture(16, 100000, true, false)
	eng := New(Config{MaxMM: 2, TerminalWindow: 0, MinLen: 100, MaxLen: 240, SeedLen: 12})
	b.ReportAllocs()

	var cp *CompiledPanel
	for i := 0; i < b.N; i++ {
		cp = eng.CompilePanel(fixture.pairs)
	}
	benchmarkCompiledSink = cp
}

func BenchmarkBuildSeedPatternsMismatch1Panel64(b *testing.B) {
	fixture := makeEngineBenchFixture(64, 100000, true, false)
	b.ReportAllocs()

	var patterns []SeedPattern
	for i := 0; i < b.N; i++ {
		patterns, _ = buildSeedPatterns(fixture.pairs, 12, 0, 1)
	}
	benchmarkSeedPatternsSink = patterns
}

func BenchmarkSimulateCompiledExactPanel64(b *testing.B) {
	fixture := makeEngineBenchFixture(64, 250000, false, false)
	eng := New(Config{MaxMM: 0, TerminalWindow: 0, MinLen: 100, MaxLen: 240, SeedLen: 12})
	cp := eng.CompilePanel(fixture.pairs)
	b.ReportAllocs()
	b.SetBytes(int64(len(fixture.seq)))
	b.ResetTimer()

	var products []Product
	for i := 0; i < b.N; i++ {
		products = eng.SimulateCompiled("bench", fixture.seq, cp)
	}
	benchmarkProductsSink = products
}

func BenchmarkSimulateCompiledMismatch1Panel16(b *testing.B) {
	fixture := makeEngineBenchFixture(16, 250000, true, false)
	eng := New(Config{MaxMM: 1, TerminalWindow: 0, MinLen: 100, MaxLen: 240, SeedLen: 12})
	cp := eng.CompilePanel(fixture.pairs)
	b.ReportAllocs()
	b.SetBytes(int64(len(fixture.seq)))
	b.ResetTimer()

	var products []Product
	for i := 0; i < b.N; i++ {
		products = eng.SimulateCompiled("bench", fixture.seq, cp)
	}
	benchmarkProductsSink = products
}

func BenchmarkSimulateCompiledMismatch2Panel16(b *testing.B) {
	fixture := makeEngineBenchFixture(16, 250000, true, false)
	eng := New(Config{MaxMM: 2, TerminalWindow: 0, MinLen: 100, MaxLen: 240, SeedLen: 12})
	cp := eng.CompilePanel(fixture.pairs)
	b.ReportAllocs()
	b.SetBytes(int64(len(fixture.seq)))
	b.ResetTimer()

	var products []Product
	for i := 0; i < b.N; i++ {
		products = eng.SimulateCompiled("bench", fixture.seq, cp)
	}
	benchmarkProductsSink = products
}

func BenchmarkSimulateCompiledMismatch1Panel16WithReferenceN(b *testing.B) {
	fixture := makeEngineBenchFixture(16, 250000, false, true)
	eng := New(Config{MaxMM: 1, TerminalWindow: 0, MinLen: 100, MaxLen: 240, SeedLen: 12})
	cp := eng.CompilePanel(fixture.pairs)
	b.ReportAllocs()
	b.SetBytes(int64(len(fixture.seq)))
	b.ResetTimer()

	var products []Product
	for i := 0; i < b.N; i++ {
		products = eng.SimulateCompiled("bench", fixture.seq, cp)
	}
	benchmarkProductsSink = products
}

func BenchmarkSimulateBatchBruteForceMismatch1SmallPanel(b *testing.B) {
	// This intentionally exercises the old exhaustive mismatch oracle. Keep the
	// fixture small so it remains usable as a local comparison benchmark.
	fixture := makeEngineBenchFixture(4, 20000, true, false)
	eng := New(Config{MaxMM: 1, TerminalWindow: 0, MinLen: 100, MaxLen: 240, SeedLen: 12})
	b.ReportAllocs()
	b.SetBytes(int64(len(fixture.seq)))
	b.ResetTimer()

	var products []Product
	for i := 0; i < b.N; i++ {
		products = eng.SimulateBatchBruteForce("bench", fixture.seq, fixture.pairs)
	}
	benchmarkProductsSink = products
}
