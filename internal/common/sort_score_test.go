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

func TestSortProductsTotalOrderUsesSourceAndGlobalChunkCoords(t *testing.T) {
	ps := []engine.Product{
		{SourceFile: "b.fa", SequenceID: "s", Start: 0, End: 8, Length: 8, Type: "forward", ExperimentID: "x"},
		{SourceFile: "a.fa", SequenceID: "s:10-20", Start: 2, End: 8, Length: 6, Type: "forward", ExperimentID: "x"},
		{SourceFile: "a.fa", SequenceID: "s", Start: 4, End: 10, Length: 6, Type: "forward", ExperimentID: "x"},
	}
	SortProducts(ps)
	if ps[0].SourceFile != "a.fa" || ps[0].SequenceID != "s" || ps[0].Start != 4 {
		t.Fatalf("expected a.fa global start 4 first, got %+v", ps[0])
	}
	if ps[1].SourceFile != "a.fa" || ps[1].SequenceID != "s:10-20" {
		t.Fatalf("expected chunked a.fa product second by global start, got %+v", ps[1])
	}
	if ps[2].SourceFile != "b.fa" {
		t.Fatalf("expected b.fa product last, got %+v", ps[2])
	}
}
