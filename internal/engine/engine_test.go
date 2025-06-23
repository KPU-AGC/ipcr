// internal/engine/engine_test.go
package engine

import (
	"bytes"
	"ipcress-go/internal/primer"
	"testing"
)

func TestSimulateMinimal(t *testing.T) {
	seq := []byte("ACGTACGTACGT")
	pair := primer.Pair{
		ID:      "test",
		Forward: "ACG",
		Reverse: "ACG",
	}

	eng := New(Config{MaxMM: 0, Disallow3MM: false})
	got := eng.Simulate("dummySeq", seq, pair)

	if len(got) == 0 {
		t.Fatalf("expected at least one product")
	}

	first := got[0]
	if first.Start != 0 || first.End != 12 || first.Length != 12 {
		t.Errorf("product coords %+v, want Start=0 End=12 Length=12", first)
	}

	wantSeq := seq[first.Start:first.End]
	rcProduct := primer.RevComp(seq[first.Start:first.End])
	if !bytes.Equal(wantSeq, primer.RevComp(rcProduct)) {
		t.Errorf("round-trip revcomp failed")
	}
}

func TestLengthFilteringAndType(t *testing.T) {
	seq := []byte("ACGTACGTACGT")
	pair := primer.Pair{ID: "t", Forward: "ACG", Reverse: "ACG",
		MinProduct: 10, MaxProduct: 12}

	eng := New(Config{MaxMM: 0, Disallow3MM: false})
	hits := eng.Simulate("seq", seq, pair)

	if len(hits) == 0 {
		t.Fatalf("expected product within bounds")
	}
	for _, p := range hits {
		if p.Length < 10 || p.Length > 12 {
			t.Errorf("product length %d outside bounds", p.Length)
		}
	}
}

func TestLengthOutOfRange(t *testing.T) {
	seq := []byte("ACGTACGTACGT")
	pair := primer.Pair{ID: "t2", Forward: "ACG", Reverse: "ACG",
		MinProduct: 5, MaxProduct: 7}

	eng := New(Config{MaxMM: 0, Disallow3MM: false})
	hits := eng.Simulate("seq", seq, pair)

	if len(hits) != 0 {
		t.Fatalf("expected zero products, got %d", len(hits))
	}
}

func TestRevcompProduct(t *testing.T) {
	seq := []byte("TTTACGACGTAAA") // A primer = "ACG", B primer = "TTT"
	pair := primer.Pair{
		ID:      "rev",
		Forward: "ACG", // A
		Reverse: "TTT", // B
	}
	eng := New(Config{})
	hits := eng.Simulate("s", seq, pair)

	found := false
	for _, h := range hits {
		if h.Type == "revcomp" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected at least one revcomp product, got %+v", hits)
	}
}
