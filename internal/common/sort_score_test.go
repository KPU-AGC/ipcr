package common

import (
	"ipcr-core/engine"
	"testing"
)

func TestSortProductsByScore(t *testing.T) {
	ps := []engine.Product{
		{SequenceID: "s", Start: 2, End: 5, Score: 0.1},
		{SequenceID: "s", Start: 1, End: 4, Score: 3.0},
		{SequenceID: "s", Start: 3, End: 6, Score: 3.0}, // tie on score → coord order fallback
	}
	SortProductsByScore(ps)
	if ps[0].Score != 3.0 || ps[1].Score != 3.0 || ps[2].Score != 0.1 {
		t.Fatalf("unexpected order: %+v", ps)
	}
	// For the tie, Start=1 should come before Start=3
	if ps[0].Start != 1 || ps[1].Start != 3 {
		t.Fatalf("tie-break by coord failed: start got %d then %d", ps[0].Start, ps[1].Start)
	}
}
