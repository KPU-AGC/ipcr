// internal/engine/engine_test.go
package engine

import (
	"bytes"
	"testing"

	"ipcr-core/primer"
)

// Minimal simulation: should find one full-length product
func TestSimulateMinimal(t *testing.T) {
	seq := []byte("ACGTACGTACGT")
	pair := primer.Pair{
		ID:      "test",
		Forward: "ACG",
		Reverse: "ACG",
	}

	eng := New(Config{MaxMM: 0, TerminalWindow: 0})
	got := eng.Simulate("dummySeq", seq, pair)

	if len(got) == 0 {
		t.Fatal("expected at least one product")
	}

	first := got[0]
	if first.Start != 0 || first.End != 12 || first.Length != 12 {
		t.Errorf("unexpected product coords: %+v, want Start=0 End=12 Length=12", first)
	}

	wantSeq := seq[first.Start:first.End]
	if !bytes.Equal(wantSeq, primer.RevComp(primer.RevComp(wantSeq))) {
		t.Error("round-trip revcomp failed")
	}
}

// Should filter product lengths correctly and set type
func TestLengthFilteringAndType(t *testing.T) {
	seq := []byte("ACGTACGTACGT")
	pair := primer.Pair{
		ID:         "t",
		Forward:    "ACG",
		Reverse:    "ACG",
		MinProduct: 10,
		MaxProduct: 12,
	}

	eng := New(Config{MaxMM: 0, TerminalWindow: 0})
	hits := eng.Simulate("seq", seq, pair)

	if len(hits) == 0 {
		t.Fatal("expected product within bounds")
	}
	for _, p := range hits {
		if p.Length < 10 || p.Length > 12 {
			t.Errorf("product length %d outside bounds", p.Length)
		}
	}
}

// Should return no products outside bounds
func TestLengthOutOfRange(t *testing.T) {
	seq := []byte("ACGTACGTACGT")
	pair := primer.Pair{
		ID:         "t2",
		Forward:    "ACG",
		Reverse:    "ACG",
		MinProduct: 5,
		MaxProduct: 7,
	}

	eng := New(Config{MaxMM: 0, TerminalWindow: 0})
	hits := eng.Simulate("seq", seq, pair)

	if len(hits) != 0 {
		t.Fatalf("expected zero products, got %d", len(hits))
	}
}

// Should detect at least one revcomp product
func TestRevcompProduct(t *testing.T) {
	seq := []byte("TTTACGACGTAAA")
	pair := primer.Pair{
		ID:      "rev",
		Forward: "ACG",
		Reverse: "TTT",
	}
	eng := New(Config{})
	hits := eng.Simulate("s", seq, pair)

	found := false
	for _, h := range hits {
		if h.Type == "revcomp" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected at least one revcomp product, got %+v", hits)
	}
}

// Circular template should allow wrap-around amplicon when reverse site is left of forward site.
func TestCircularAmplicon(t *testing.T) {
	seq := []byte("TGACAAG") // 7 bp circular
	pair := primer.Pair{ID: "p1", Forward: "AG", Reverse: "TC"}

	// Linear mode: no amplicon for this configuration.
	engLin := New(Config{Circular: false, MaxMM: 0, MinLen: 0, MaxLen: 0})
	hitsLin := engLin.Simulate("seq1", seq, pair)
	if len(hitsLin) != 0 {
		t.Errorf("expected no amplicon in linear mode, got %d", len(hitsLin))
	}

	// Circular mode: should produce one wrap-around product.
	engCirc := New(Config{Circular: true, MaxMM: 0, MinLen: 0, MaxLen: 0})
	hitsCirc := engCirc.Simulate("seq1", seq, pair)
	if len(hitsCirc) != 1 {
		t.Fatalf("expected 1 amplicon in circular mode, got %d", len(hitsCirc))
	}
	prod := hitsCirc[0]
	if prod.Start <= prod.End {
		t.Errorf("expected wrap-around coordinates (Start > End), got Start=%d End=%d", prod.Start, prod.End)
	}
	expectedLen := len(seq) - prod.Start + prod.End
	if prod.Length != expectedLen {
		t.Errorf("expected product length %d, got %d", expectedLen, prod.Length)
	}
}
