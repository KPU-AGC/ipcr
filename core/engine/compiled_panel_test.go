// core/engine/compiled_panel_test.go
package engine

import (
	"errors"
	"ipcr-core/primer"
	"testing"
)

var errForEachCompiledProductTest = errors.New("stop streaming")

func TestSimulateCompiledMatchesBruteForceOracle(t *testing.T) {
	pairs := []primer.Pair{
		{
			ID:         "exact",
			Forward:    "ACGTAC",
			Reverse:    "GGTACC",
			MinProduct: 6,
			MaxProduct: 60,
		},
		{
			ID:         "ambiguous",
			Forward:    "ACRTAC",
			Reverse:    "GGNACC",
			MinProduct: 6,
			MaxProduct: 60,
		},
	}
	seqs := map[string][]byte{
		"forward": []byte("TTTACGTACAAAAGGTACCTTT"),
		"revcomp": []byte("TTTGGTACCAAAAGTACGTTTT"),
		"mixed":   []byte("TTTTCGTACAAAAGGTACCTTTGGNACC"),
	}

	for _, cfg := range []Config{
		{MaxMM: 0, TerminalWindow: 0, MinLen: 6, MaxLen: 60, SeedLen: 4, NeedSites: true},
		{MaxMM: 1, TerminalWindow: 0, MinLen: 6, MaxLen: 60, SeedLen: 4, NeedSites: true},
		{MaxMM: 1, TerminalWindow: 3, MinLen: 6, MaxLen: 60, SeedLen: 4, NeedSites: true},
		{MaxMM: 2, TerminalWindow: 3, MinLen: 6, MaxLen: 60, SeedLen: 4, NeedSites: true},
	} {
		eng := New(cfg)
		cp := eng.CompilePanel(pairs)
		for seqID, seq := range seqs {
			got := eng.SimulateCompiled(seqID, seq, cp)
			want := eng.SimulateBatchBruteForce(seqID, seq, pairs)
			assertProductMultisetEqual(t, got, want)
		}
	}
}

func TestSimulateCompiledWithScratchMatchesBruteForceAcrossReuse(t *testing.T) {
	pairs := []primer.Pair{{
		ID:         "reuse",
		Forward:    "ACGTAC",
		Reverse:    "GGTACC",
		MinProduct: 6,
		MaxProduct: 60,
	}}
	eng := New(Config{MaxMM: 1, TerminalWindow: 0, MinLen: 6, MaxLen: 60, SeedLen: 4})
	cp := eng.CompilePanel(pairs)
	scratch := eng.NewSimulationScratch(cp)

	seqs := map[string][]byte{
		"first":  []byte("TTTACGTACAAAAGGTACCTTT"),
		"second": []byte("TTTACGTATAAAAGGTACCTTT"),
		"third":  []byte("TTTGGTACCAAAAGTACGTTTT"),
	}
	for seqID, seq := range seqs {
		got := eng.SimulateCompiledWithScratch(seqID, seq, cp, scratch)
		want := eng.SimulateBatchBruteForce(seqID, seq, pairs)
		assertProductMultisetEqual(t, got, want)
	}
}

func TestForEachCompiledProductMatchesSimulateCompiledAcrossReuse(t *testing.T) {
	pairs := []primer.Pair{{
		ID:         "stream",
		Forward:    "ACGTAC",
		Reverse:    "GGTACC",
		MinProduct: 6,
		MaxProduct: 60,
	}}
	eng := New(Config{MaxMM: 1, TerminalWindow: 0, MinLen: 6, MaxLen: 60, SeedLen: 4})
	cp := eng.CompilePanel(pairs)
	scratch := eng.NewSimulationScratch(cp)

	seqs := map[string][]byte{
		"first":  []byte("TTTACGTACAAAAGGTACCTTT"),
		"second": []byte("TTTACGTATAAAAGGTACCTTT"),
		"third":  []byte("TTTGGTACCAAAAGTACGTTTT"),
	}
	for seqID, seq := range seqs {
		var got []Product
		if err := eng.ForEachCompiledProduct(seqID, seq, cp, scratch, func(p Product) error {
			got = append(got, p)
			return nil
		}); err != nil {
			t.Fatalf("ForEachCompiledProduct error: %v", err)
		}
		want := eng.SimulateCompiled(seqID, seq, cp)
		assertProductMultisetEqual(t, got, want)
	}
}

func TestForEachCompiledProductMatchesSimulateCompiled(t *testing.T) {
	pairs := []primer.Pair{{
		ID:         "stream",
		Forward:    "ACGTAC",
		Reverse:    "GGTACC",
		MinProduct: 6,
		MaxProduct: 60,
	}}
	eng := New(Config{MaxMM: 1, TerminalWindow: 0, MinLen: 6, MaxLen: 60, SeedLen: 4})
	cp := eng.CompilePanel(pairs)
	scratch := eng.NewSimulationScratch(cp)
	seq := []byte("TTTACGTATAAAAGGTACCTTT")

	var got []Product
	err := eng.ForEachCompiledProduct("seq", seq, cp, scratch, func(p Product) error {
		got = append(got, p)
		return nil
	})
	if err != nil {
		t.Fatalf("ForEachCompiledProduct error: %v", err)
	}
	want := eng.SimulateCompiled("seq", seq, cp)
	assertProductMultisetEqual(t, got, want)
}

func TestForEachCompiledProductPropagatesEmitError(t *testing.T) {
	pairs := []primer.Pair{{
		ID:         "stream_error",
		Forward:    "ACGTAC",
		Reverse:    "GGTACC",
		MinProduct: 6,
		MaxProduct: 60,
	}}
	eng := New(Config{MaxMM: 0, TerminalWindow: 0, MinLen: 6, MaxLen: 60, SeedLen: 4})
	cp := eng.CompilePanel(pairs)
	seq := []byte("TTTACGTACAAAAGGTACCTTT")

	errSentinel := errForEachCompiledProductTest
	err := eng.ForEachCompiledProduct("seq", seq, cp, nil, func(Product) error {
		return errSentinel
	})
	if err != errSentinel {
		t.Fatalf("ForEachCompiledProduct error = %v, want %v", err, errSentinel)
	}
}

func TestCompilePanelSnapshotsPairsAndConfig(t *testing.T) {
	pairs := []primer.Pair{{
		ID:         "snapshot",
		Forward:    "ACGTAC",
		Reverse:    "GGTACC",
		MinProduct: 6,
		MaxProduct: 60,
	}}
	eng := New(Config{MaxMM: 0, TerminalWindow: 0, MinLen: 6, MaxLen: 60, SeedLen: 4})
	cp := eng.CompilePanel(pairs)

	pairs[0].ID = "mutated"
	pairs[0].Forward = "AAAAAA"
	eng.SetHitCap(1)

	seq := []byte("TTTACGTACAAAAGGTACCTTT")
	got := eng.SimulateCompiled("seq", seq, cp)
	want := New(cp.Cfg).SimulateBatchBruteForce("seq", seq, cp.Pairs)
	assertProductMultisetEqual(t, got, want)
	if len(got) == 0 {
		t.Fatal("expected products from snapshotted compiled panel")
	}
	for _, p := range got {
		if p.ExperimentID != "snapshot" {
			t.Fatalf("compiled panel did not snapshot pair metadata: got %q", p.ExperimentID)
		}
	}
}

func TestCompilePanelDeduplicatesSeedPatternsAndDispatchesPayloads(t *testing.T) {
	pairs := []primer.Pair{
		{
			ID:         "p1",
			Forward:    "ACGTAC",
			Reverse:    "GGTACC",
			MinProduct: 6,
			MaxProduct: 60,
		},
		{
			ID:         "p2",
			Forward:    "ACGTAC",
			Reverse:    "GGTACC",
			MinProduct: 6,
			MaxProduct: 60,
		},
	}

	eng := New(Config{MaxMM: 0, TerminalWindow: 0, MinLen: 6, MaxLen: 60, SeedLen: 4})
	cp := eng.CompilePanel(pairs)
	flatSeeds, _ := buildSeeds(pairs, 4, 0, 0)

	if len(cp.SeedPatterns) >= len(flatSeeds) {
		t.Fatalf("expected deduplicated seed patterns, got patterns=%d flattened=%d", len(cp.SeedPatterns), len(flatSeeds))
	}

	shared := false
	for _, pattern := range cp.SeedPatterns {
		if len(pattern.Payloads) > 1 {
			shared = true
			break
		}
	}
	if !shared {
		t.Fatalf("expected at least one seed pattern with multiple payloads: %+v", cp.SeedPatterns)
	}

	got := eng.SimulateCompiled("seq", []byte("TTTACGTACAAAAGGTACCTTT"), cp)
	seen := map[string]bool{}
	for _, product := range got {
		seen[product.ExperimentID] = true
	}
	for _, id := range []string{"p1", "p2"} {
		if !seen[id] {
			t.Fatalf("compiled seed-pattern payload dispatch missed experiment %q; products=%+v", id, got)
		}
	}
}

func TestCompilePanelStoresPrimerOrientationsInByteSlab(t *testing.T) {
	pairs := []primer.Pair{
		{ID: "one", Forward: "ACGTAC", Reverse: "GGTACC"},
		{ID: "two", Forward: "TTGCAA", Reverse: "AACCGG"},
	}
	eng := New(Config{MaxMM: 0, TerminalWindow: 0, MinLen: 1, MaxLen: 100, SeedLen: 4})
	cp := eng.CompilePanel(pairs)

	if len(cp.fwdA) != len(pairs) || len(cp.fwdB) != len(pairs) || len(cp.rcA) != len(pairs) || len(cp.rcB) != len(pairs) {
		t.Fatalf("orientation span lengths do not match pair count")
	}

	wantBytes := 0
	for i, p := range pairs {
		if got, want := string(cp.fwdASeq(i)), p.Forward; got != want {
			t.Fatalf("fwdA[%d] = %q, want %q", i, got, want)
		}
		if got, want := string(cp.fwdBSeq(i)), p.Reverse; got != want {
			t.Fatalf("fwdB[%d] = %q, want %q", i, got, want)
		}
		if got, want := string(cp.rcASeq(i)), string(primer.RevComp([]byte(p.Forward))); got != want {
			t.Fatalf("rcA[%d] = %q, want %q", i, got, want)
		}
		if got, want := string(cp.rcBSeq(i)), string(primer.RevComp([]byte(p.Reverse))); got != want {
			t.Fatalf("rcB[%d] = %q, want %q", i, got, want)
		}
		wantBytes += len(p.Forward) + len(p.Reverse) + len(p.Forward) + len(p.Reverse)
	}
	if got := len(cp.primerBytes); got != wantBytes {
		t.Fatalf("primerBytes length = %d, want %d", got, wantBytes)
	}
}
