package engine

import (
	"ipcr-core/primer"
	"testing"
)

func TestApproximateSeedRegressionGates(t *testing.T) {
	fixture := makeEngineBenchFixture(16, 20000, true, false)
	eng := New(Config{MaxMM: 1, TerminalWindow: 0, MinLen: 100, MaxLen: 240, SeedLen: 12})
	cp := eng.CompilePanel(fixture.pairs)

	if len(cp.SeedPatterns) == 0 || len(cp.Automaton.nodes) <= 1 {
		t.Fatalf("expected compiled approximate seed automaton, patterns=%d nodes=%d", len(cp.SeedPatterns), len(cp.Automaton.nodes))
	}
	for i := range fixture.pairs {
		for _, which := range []byte{'A', 'B', 'a', 'b'} {
			if !compiledHas(cp.Have, i, which) {
				t.Fatalf("expected pair %d orientation %c to be seeded; have=%v", i, which, cp.Have[i])
			}
		}
	}
}

func TestApproximateSeedVariantExplosionFallsBackPerOrientation(t *testing.T) {
	pairs := []primer.Pair{
		{ID: "variant_cap", Forward: "NNNNNNNNNNNN", Reverse: "ACGTACGTACGT"},
		{ID: "ordinary", Forward: "ACGTACGTACGT", Reverse: "TGCATGCATGCA"},
	}

	eng := New(Config{MaxMM: 2, TerminalWindow: 0, MinLen: 1, MaxLen: 100, SeedLen: 12})
	cp := eng.CompilePanel(pairs)

	for _, which := range []byte{'A', 'a'} {
		if compiledHas(cp.Have, 0, which) {
			t.Fatalf("expected highly degenerate orientation %c to fall back after variant cap; have=%v", which, cp.Have[0])
		}
	}
	for _, which := range []byte{'B', 'b'} {
		if !compiledHas(cp.Have, 0, which) {
			t.Fatalf("expected non-degenerate orientation %c to remain seeded; have=%v", which, cp.Have[0])
		}
	}
	for _, which := range []byte{'A', 'B', 'a', 'b'} {
		if !compiledHas(cp.Have, 1, which) {
			t.Fatalf("variant cap disabled ordinary pair orientation %c; have=%v", which, cp.Have[1])
		}
	}
}

func TestApproximateSeedBenchmarkWorkloadMatchesBruteForce(t *testing.T) {
	fixture := makeEngineBenchFixture(12, 20000, true, true)
	eng := New(Config{MaxMM: 2, TerminalWindow: 0, MinLen: 100, MaxLen: 240, SeedLen: 12})
	cp := eng.CompilePanel(fixture.pairs)

	got := eng.SimulateCompiled("bench", fixture.seq, cp)
	want := eng.SimulateBatchBruteForce("bench", fixture.seq, fixture.pairs)
	assertProductMultisetEqual(t, got, want)
}
