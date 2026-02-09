package engine

import (
	"ipcr-core/primer"
	"testing"
)

func TestSoftmaskToggle_LowercaseReference(t *testing.T) {
	seq := []byte("acgtacgtacgt")
	pair := primer.Pair{ID: "p", Forward: "ACG", Reverse: "ACG"}

	engDefault := New(Config{MaxMM: 0, TerminalWindow: 0, AllowSoftmask: false})
	if got := engDefault.Simulate("s", seq, pair); len(got) != 0 {
		t.Fatalf("default mode: expected 0 products, got %d", len(got))
	}

	engSoft := New(Config{MaxMM: 0, TerminalWindow: 0, AllowSoftmask: true})
	got := engSoft.Simulate("s", seq, pair)
	if len(got) == 0 {
		t.Fatalf("allow-softmask mode: expected at least 1 product, got 0")
	}
	first := got[0]
	if first.Start != 0 || first.End != 12 || first.Length != 12 {
		t.Fatalf("allow-softmask mode: unexpected first product coords: %+v", first)
	}
}

func TestSoftmaskToggle_MixedCaseSites(t *testing.T) {
	// All candidate primer binding sites are mixed-case (soft-masked).
	seq := []byte("ACgtACgtACgt")
	pair := primer.Pair{ID: "p", Forward: "ACGT", Reverse: "ACGT"}

	engDefault := New(Config{MaxMM: 0, TerminalWindow: 0, AllowSoftmask: false})
	if got := engDefault.Simulate("s", seq, pair); len(got) != 0 {
		t.Fatalf("default mode: expected 0 products, got %d", len(got))
	}

	engSoft := New(Config{MaxMM: 0, TerminalWindow: 0, AllowSoftmask: true})
	if got := engSoft.Simulate("s", seq, pair); len(got) == 0 {
		t.Fatalf("allow-softmask mode: expected products, got 0")
	}
}

func TestSeedScannerSoftmask(t *testing.T) {
	// Case-sensitive scanner should not see lowercase.
	seeds := []Seed{{PairIdx: 0, Which: 'A', Pat: []byte("ACG"), PrimerLen: 3, SeedOffset: 0}}
	nodes, _ := buildAC(seeds)

	seq := []byte("ttacgtt") // "acg" starts at 2, ends at 4
	if hits := scanAC(seq, nodes, seeds); len(hits) != 0 {
		t.Fatalf("scanAC: expected 0 hits in lowercase region, got %d", len(hits))
	}

	hits := scanACSoftmask(seq, nodes, seeds)
	if len(hits) != 1 {
		t.Fatalf("scanACSoftmask: expected 1 hit, got %d", len(hits))
	}
	if hits[0].Pos != 4 || hits[0].SeedIdx != 0 {
		t.Fatalf("scanACSoftmask: unexpected hit: %+v", hits[0])
	}
}

func TestOutputPreservation_Sites(t *testing.T) {
	// Ensure emitted binding-site strings preserve the original reference casing.
	seq := []byte("ACgtACGTacgt")
	pair := primer.Pair{ID: "p", Forward: "ACGT", Reverse: "ACGT"}

	eng := New(Config{MaxMM: 0, TerminalWindow: 0, NeedSites: true, AllowSoftmask: true})
	got := eng.Simulate("s", seq, pair)
	if len(got) == 0 {
		t.Fatalf("expected at least 1 product")
	}

	first := got[0]
	if first.FwdSite != "ACgt" {
		t.Fatalf("FwdSite casing changed: got %q, want %q", first.FwdSite, "ACgt")
	}
	if first.RevSite != "acgt" {
		t.Fatalf("RevSite casing changed: got %q, want %q", first.RevSite, "acgt")
	}
}
