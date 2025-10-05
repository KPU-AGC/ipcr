package common

import (
	"ipcr-core/primer"
	"testing"
)

func TestAddSelfPairsUnique(t *testing.T) {
	in := []primer.Pair{
		{ID: "p1", Forward: "AAA", Reverse: "CCC"},
		{ID: "p2", Forward: "AAA", Reverse: "GGG"}, // forward duplicate
		{ID: "p3", Forward: "TTT", Reverse: "CCC"}, // reverse duplicate
	}
	out := AddSelfPairsUnique(in)

	// Expect original 3 + unique A:self for AAA, TTT + unique B:self for CCC, GGG => +4
	if len(out) != 7 {
		t.Fatalf("want 7 pairs (3 originals + 4 self), got %d", len(out))
	}
	// Basic sanity: last 4 are self-pairs
	for i := 3; i < len(out); i++ {
		if out[i].Forward != out[i].Reverse {
			t.Fatalf("self pair not equal at %d: %+v", i, out[i])
		}
	}
}
