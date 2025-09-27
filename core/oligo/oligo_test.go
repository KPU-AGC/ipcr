package oligo

import "testing"

func TestBestHit(t *testing.T) {
	h := BestHit("ACGTACGTACGT", "GTAC", 0)
	if !h.Found || h.Pos != 2 || h.MM != 0 || h.Strand != "+" {
		t.Fatalf("unexpected hit: %+v", h)
	}
	h2 := BestHit("ACGTACGTACGT", "GTGC", 1) // RC=GCAC; allow 1 mm
	if !h2.Found {
		t.Fatalf("expected a hit on RC with mismatches")
	}
}
