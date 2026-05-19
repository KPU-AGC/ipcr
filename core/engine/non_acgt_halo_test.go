package engine

import (
	"ipcr-core/primer"
	"reflect"
	"testing"
)

func TestNonACGTRangesTreatLowercaseACGTAsCanonical(t *testing.T) {
	got := nonACGTRanges([]byte("aaNaaRRtt"))
	want := []seqRange{{start: 2, end: 3}, {start: 5, end: 7}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranges = %#v, want %#v", got, want)
	}
}

func TestForEachNonACGTHaloStartMergesOverlappingHalos(t *testing.T) {
	ranges := []seqRange{{start: 5, end: 6}, {start: 7, end: 8}}
	var got []int
	forEachNonACGTHaloStart(20, 6, ranges, func(start int) {
		got = append(got, start)
	})

	// A 6-mer overlapping the first reset byte can start at 0..5; one overlapping
	// the second can start at 2..7. These halos overlap and should be emitted once.
	want := []int{0, 1, 2, 3, 4, 5, 6, 7}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("halo starts = %#v, want %#v", got, want)
	}
}

func TestSimulateCompiledUsesNonACGTHaloForSeededOrientation(t *testing.T) {
	pairs := []primer.Pair{{
		ID:      "reference_n_inside_seed",
		Forward: "ACGTAC",
		Reverse: "GGTACC",
	}}
	seq := []byte("TTTACNTACAAAAGGTACCTTT")

	eng := New(Config{MaxMM: 1, TerminalWindow: 0, MinLen: 1, MaxLen: 100, SeedLen: 6})
	cp := eng.CompilePanel(pairs)
	if !compiledHas(cp.Have, 0, 'A') {
		t.Fatalf("expected forward primer to be seeded: %+v", cp.SeedPatterns)
	}

	got := eng.SimulateCompiled("seq", seq, cp)
	want := eng.SimulateBatchBruteForce("seq", seq, pairs)
	assertProductMultisetEqual(t, got, want)
	if len(got) == 0 {
		t.Fatal("expected product with reference N accepted as one mismatch")
	}
}
