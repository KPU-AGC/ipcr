package pipeline

import (
	"context"
	"ipcr-core/engine"
	"ipcr-core/primer"
	"os"
	"testing"
)

func TestForEachProduct_NoChunking(t *testing.T) {
	// Make a tiny FASTA
	fn := "pipe_test.fa"
	defer func() { _ = os.Remove(fn) }()
	if err := os.WriteFile(fn, []byte(">s\nACGTACGTACGT\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	pairs := []primer.Pair{{
		ID: "x", Forward: "ACG", Reverse: "ACG", MinProduct: 0, MaxProduct: 1000,
	}}
	eng := engine.New(engine.Config{MaxLen: 1000})
	var n int
	err := ForEachProduct(context.Background(), Config{
		Threads: 1, ChunkSize: 0, Overlap: 0, Circular: false, NeedSeq: true,
	}, []string{fn}, pairs, eng, func(p engine.Product) error {
		n++
		return nil
	})
	if err != nil {
		t.Fatalf("pipeline err: %v", err)
	}
	if n == 0 {
		t.Fatal("expected at least one product")
	}
}

func TestForEachProduct_PreservesReferenceCasingInSeq(t *testing.T) {
	fn := "pipe_test_softmask.fa"
	defer func() { _ = os.Remove(fn) }()
	wantSeq := "ACgtACGTacgt"
	if err := os.WriteFile(fn, []byte(">s\n"+wantSeq+"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	pairs := []primer.Pair{{ID: "x", Forward: "ACGT", Reverse: "ACGT", MinProduct: 0, MaxProduct: 1000}}
	eng := engine.New(engine.Config{MaxLen: 1000, AllowSoftmask: true})

	var got []string
	err := ForEachProduct(context.Background(), Config{
		Threads: 1, ChunkSize: 0, Overlap: 0, Circular: false, NeedSeq: true,
	}, []string{fn}, pairs, eng, func(p engine.Product) error {
		got = append(got, p.Seq)
		return nil
	})
	if err != nil {
		t.Fatalf("pipeline err: %v", err)
	}

	found := false
	for _, s := range got {
		if s == wantSeq {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected an emitted product sequence equal to %q, got %v", wantSeq, got)
	}
}
