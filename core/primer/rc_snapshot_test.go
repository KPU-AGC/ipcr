package primer

import "testing"

// Snapshot: RevComp over the full ambiguity alphabet + ACGT.
// We assert the exact bytes produced by the current complement table.
func TestComplementTable_Snapshot(t *testing.T) {
	in := []byte("RYSWKMBDHVNACGT")
	// This is the reverse-complement the current table produces.
	want := []byte("ACGTNBDHVKMWSRY")

	got := RevComp(in)
	if string(got) != string(want) {
		t.Fatalf("complement table changed:\n got  %s\n want %s", got, want)
	}
}
