package engine

import (
	"testing"

	"ipcr-core/primer"
)

func TestMatchCollectorDeduplicatesAcceptedAndRejectedStarts(t *testing.T) {
	seq := []byte("ACGTAC")
	pat := []byte("ACGT")

	var c matchCollector
	c.addVerified(seq, 0, pat, 0, 0, 0, 0)
	c.addVerified(seq, 0, pat, 0, 0, 0, 0)
	if len(c.matches) != 1 {
		t.Fatalf("accepted duplicate start was not deduplicated: got %d matches", len(c.matches))
	}

	c.addVerified(seq, 1, pat, 0, 0, 0, 0)
	c.addVerified(seq, 1, pat, 0, 0, 0, 0)
	if len(c.matches) != 1 {
		t.Fatalf("rejected duplicate start changed matches: got %d", len(c.matches))
	}
	if len(c.starts) != 2 || c.visited != nil {
		t.Fatalf("sparse attempted starts not kept in slice: starts=%v visited=%v", c.starts, c.visited)
	}
}

func TestMatchCollectorHonorsHitCap(t *testing.T) {
	seq := []byte("AAAAAA")
	pat := []byte("AA")

	var c matchCollector
	c.addVerified(seq, 0, pat, 0, 0, 0, 1)
	c.addVerified(seq, 1, pat, 0, 0, 0, 1)
	if len(c.matches) != 1 {
		t.Fatalf("matches = %d, want 1", len(c.matches))
	}
	if c.matches[0].Pos != 0 {
		t.Fatalf("first capped match pos = %d, want 0", c.matches[0].Pos)
	}
}

func TestMatchCollectorPromotesStartsToVisitedMap(t *testing.T) {
	seq := []byte("AAAAAAAAAAAAAAAAAAAAAAAA")
	pat := []byte("AA")

	var c matchCollector
	for start := 0; start < matchCollectorLinearLimit; start++ {
		c.addVerified(seq, start, pat, 0, 0, 0, 0)
	}
	if c.visited != nil {
		t.Fatalf("visited map allocated before promotion threshold")
	}
	if len(c.starts) != matchCollectorLinearLimit {
		t.Fatalf("starts length = %d, want %d", len(c.starts), matchCollectorLinearLimit)
	}

	c.addVerified(seq, matchCollectorLinearLimit, pat, 0, 0, 0, 0)
	if c.visited == nil {
		t.Fatalf("visited map was not allocated after promotion threshold")
	}
	if len(c.starts) != 0 {
		t.Fatalf("starts slice length after promotion = %d, want 0", len(c.starts))
	}

	before := len(c.matches)
	c.addVerified(seq, 0, pat, 0, 0, 0, 0)
	if len(c.matches) != before {
		t.Fatalf("duplicate after promotion changed matches from %d to %d", before, len(c.matches))
	}
}

func TestMatchCollectorResetClearsSparseStartsAndVisitedMap(t *testing.T) {
	seq := []byte("AAAAAAAAAAAAAAAAAAAAAAAA")
	pat := []byte("AA")

	var sparse matchCollector
	sparse.addVerified(seq, 0, pat, 0, 0, 0, 0)
	sparse.reset()
	if len(sparse.starts) != 0 || len(sparse.matches) != 0 || sparse.visited != nil {
		t.Fatalf("sparse reset left state: starts=%v matches=%v visited=%v", sparse.starts, sparse.matches, sparse.visited)
	}

	var dense matchCollector
	for start := 0; start <= matchCollectorLinearLimit; start++ {
		dense.addVerified(seq, start, pat, 0, 0, 0, 0)
	}
	if dense.visited == nil {
		t.Fatal("expected dense collector to promote before reset")
	}
	dense.reset()
	if len(dense.starts) != 0 || len(dense.matches) != 0 || len(dense.visited) != 0 {
		t.Fatalf("dense reset left state: starts=%v matches=%v visited=%v", dense.starts, dense.matches, dense.visited)
	}
}

func TestSimulationScratchResetsCollectorsAcrossReuse(t *testing.T) {
	pairs := []primer.Pair{{ID: "x", Forward: "ACGTAC", Reverse: "GGTACC"}}
	eng := New(Config{MaxMM: 0, MinLen: 6, MaxLen: 60, SeedLen: 4})
	cp := eng.CompilePanel(pairs)
	scratch := eng.NewSimulationScratch(cp)

	seq1 := []byte("TTTACGTACAAAAGGTACCTTT")
	seq2 := []byte("TTTACGTACAAAACCCCCCCTTT")
	if got := eng.SimulateCompiledWithScratch("seq1", seq1, cp, scratch); len(got) == 0 {
		t.Fatal("expected products in first scratch use")
	}
	if got := eng.SimulateCompiledWithScratch("seq2", seq2, cp, scratch); len(got) != 0 {
		t.Fatalf("scratch reuse leaked previous matches: %+v", got)
	}
}
