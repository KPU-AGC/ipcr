// internal/pipeline/pipeline_engine_contract_test.go
package pipeline

import (
	"context"
	"ipcr-core/engine"
	"ipcr-core/primer"
	"os"
	"testing"
)

// Compile-time check: the concrete engine satisfies the minimal contract.
var _ Simulator = (*engine.Engine)(nil)

// fake engine implementing the Simulator interface
type fakeEng struct{}

func (fakeEng) SimulateBatch(seqID string, seq []byte, pairs []primer.Pair) []engine.Product {
	return []engine.Product{{
		ExperimentID: "x", SequenceID: seqID,
		Start: 1, End: 3, Length: 2, Type: "forward",
	}}
}

func TestForEachProduct_UsesSimulatorAndFillsSeq(t *testing.T) {
	fn := "pipe_fake.fa"
	if err := os.WriteFile(fn, []byte(">s\nACGT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(fn) }()

	var n int
	err := ForEachProduct(
		context.Background(),
		Config{Threads: 1, ChunkSize: 0, Overlap: 0, Circular: false, NeedSeq: true},
		[]string{fn},
		[]primer.Pair{{ID: "x"}},
		fakeEng{},
		func(p engine.Product) error {
			n++
			if p.Seq == "" {
				t.Fatalf("expected Seq to be filled by pipeline when NeedSeq=true")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("pipeline err: %v", err)
	}
	if n != 1 {
		t.Fatalf("want 1 product, got %d", n)
	}
}
