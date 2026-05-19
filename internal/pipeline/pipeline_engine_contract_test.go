// internal/pipeline/pipeline_engine_contract_test.go
package pipeline

import (
	"context"
	"ipcr-core/engine"
	"ipcr-core/primer"
	"os"
	"testing"
)

// Compile-time checks: the concrete engine satisfies all pipeline contracts.
var (
	_ Simulator                  = (*engine.Engine)(nil)
	_ CompiledSimulator          = (*engine.Engine)(nil)
	_ ScratchCompiledSimulator   = (*engine.Engine)(nil)
	_ StreamingCompiledSimulator = (*engine.Engine)(nil)
)

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

type countingCompiledEng struct {
	base          *engine.Engine
	compileCalls  int
	compiledCalls int
	batchCalls    int
}

func (e *countingCompiledEng) CompilePanel(pairs []primer.Pair) *engine.CompiledPanel {
	e.compileCalls++
	return e.base.CompilePanel(pairs)
}

func (e *countingCompiledEng) SimulateCompiled(seqID string, seq []byte, cp *engine.CompiledPanel) []engine.Product {
	e.compiledCalls++
	return e.base.SimulateCompiled(seqID, seq, cp)
}

func (e *countingCompiledEng) SimulateBatch(seqID string, seq []byte, pairs []primer.Pair) []engine.Product {
	e.batchCalls++
	return e.base.SimulateBatch(seqID, seq, pairs)
}

func TestForEachProduct_UsesCompiledSimulatorOnce(t *testing.T) {
	fn := "pipe_compiled.fa"
	if err := os.WriteFile(fn, []byte(">s1\nACGTACAAAAGGTACC\n>s2\nACGTACAAAAGGTACC\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(fn) }()

	pairs := []primer.Pair{{
		ID:         "x",
		Forward:    "ACGTAC",
		Reverse:    "GGTACC",
		MinProduct: 6,
		MaxProduct: 100,
	}}
	counting := &countingCompiledEng{base: engine.New(engine.Config{MinLen: 6, MaxLen: 100, SeedLen: 4})}

	var n int
	err := ForEachProduct(
		context.Background(),
		Config{Threads: 2, ChunkSize: 0, Overlap: 0, Circular: false, NeedSeq: true},
		[]string{fn},
		pairs,
		counting,
		func(p engine.Product) error {
			n++
			if p.SourceFile != fn {
				t.Fatalf("SourceFile = %q, want %q", p.SourceFile, fn)
			}
			if p.Seq == "" {
				t.Fatalf("expected Seq to be filled by streaming pipeline when NeedSeq=true")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("pipeline err: %v", err)
	}
	if n == 0 {
		t.Fatal("expected at least one product")
	}
	if counting.compileCalls != 1 {
		t.Fatalf("CompilePanel calls: got %d, want 1", counting.compileCalls)
	}
	if counting.compiledCalls != 2 {
		t.Fatalf("SimulateCompiled calls: got %d, want 2", counting.compiledCalls)
	}
	if counting.batchCalls != 0 {
		t.Fatalf("SimulateBatch fallback calls: got %d, want 0", counting.batchCalls)
	}
}

type countingScratchCompiledEng struct {
	base                 *engine.Engine
	compileCalls         int
	compiledCalls        int
	scratchCalls         int
	scratchCompiledCalls int
	streamCalls          int
	batchCalls           int
}

type countingStreamingCompiledEng struct {
	base                 *engine.Engine
	compileCalls         int
	compiledCalls        int
	scratchCalls         int
	scratchCompiledCalls int
	streamingCalls       int
	batchCalls           int
}

func (e *countingScratchCompiledEng) CompilePanel(pairs []primer.Pair) *engine.CompiledPanel {
	e.compileCalls++
	return e.base.CompilePanel(pairs)
}

func (e *countingScratchCompiledEng) SimulateCompiled(seqID string, seq []byte, cp *engine.CompiledPanel) []engine.Product {
	e.compiledCalls++
	return e.base.SimulateCompiled(seqID, seq, cp)
}

func (e *countingScratchCompiledEng) NewSimulationScratch(cp *engine.CompiledPanel) *engine.SimulationScratch {
	e.scratchCalls++
	return e.base.NewSimulationScratch(cp)
}

func (e *countingScratchCompiledEng) SimulateCompiledWithScratch(seqID string, seq []byte, cp *engine.CompiledPanel, scratch *engine.SimulationScratch) []engine.Product {
	e.scratchCompiledCalls++
	return e.base.SimulateCompiledWithScratch(seqID, seq, cp, scratch)
}

func (e *countingScratchCompiledEng) ForEachCompiledProduct(seqID string, seq []byte, cp *engine.CompiledPanel, scratch *engine.SimulationScratch, emit func(engine.Product) error) error {
	e.streamCalls++
	return e.base.ForEachCompiledProduct(seqID, seq, cp, scratch, emit)
}

func (e *countingScratchCompiledEng) SimulateBatch(seqID string, seq []byte, pairs []primer.Pair) []engine.Product {
	e.batchCalls++
	return e.base.SimulateBatch(seqID, seq, pairs)
}

func (e *countingStreamingCompiledEng) CompilePanel(pairs []primer.Pair) *engine.CompiledPanel {
	e.compileCalls++
	return e.base.CompilePanel(pairs)
}

func (e *countingStreamingCompiledEng) SimulateCompiled(seqID string, seq []byte, cp *engine.CompiledPanel) []engine.Product {
	e.compiledCalls++
	return e.base.SimulateCompiled(seqID, seq, cp)
}

func (e *countingStreamingCompiledEng) NewSimulationScratch(cp *engine.CompiledPanel) *engine.SimulationScratch {
	e.scratchCalls++
	return e.base.NewSimulationScratch(cp)
}

func (e *countingStreamingCompiledEng) SimulateCompiledWithScratch(seqID string, seq []byte, cp *engine.CompiledPanel, scratch *engine.SimulationScratch) []engine.Product {
	e.scratchCompiledCalls++
	return e.base.SimulateCompiledWithScratch(seqID, seq, cp, scratch)
}

func (e *countingStreamingCompiledEng) ForEachCompiledProduct(seqID string, seq []byte, cp *engine.CompiledPanel, scratch *engine.SimulationScratch, emit func(engine.Product) error) error {
	e.streamingCalls++
	return e.base.ForEachCompiledProduct(seqID, seq, cp, scratch, emit)
}

func (e *countingStreamingCompiledEng) SimulateBatch(seqID string, seq []byte, pairs []primer.Pair) []engine.Product {
	e.batchCalls++
	return e.base.SimulateBatch(seqID, seq, pairs)
}

func TestForEachProduct_UsesScratchCompiledSimulatorPerWorker(t *testing.T) {
	fn := "pipe_scratch_compiled.fa"
	if err := os.WriteFile(fn, []byte(">s1\nACGTACAAAAGGTACC\n>s2\nACGTACAAAAGGTACC\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(fn) }()

	pairs := []primer.Pair{{
		ID:         "x",
		Forward:    "ACGTAC",
		Reverse:    "GGTACC",
		MinProduct: 6,
		MaxProduct: 100,
	}}
	counting := &countingScratchCompiledEng{base: engine.New(engine.Config{MinLen: 6, MaxLen: 100, SeedLen: 4})}

	var n int
	err := ForEachProduct(
		context.Background(),
		Config{Threads: 2, ChunkSize: 0, Overlap: 0, Circular: false, NeedSeq: false},
		[]string{fn},
		pairs,
		counting,
		func(engine.Product) error {
			n++
			return nil
		},
	)
	if err != nil {
		t.Fatalf("pipeline err: %v", err)
	}
	if n == 0 {
		t.Fatal("expected at least one product")
	}
	if counting.compileCalls != 1 {
		t.Fatalf("CompilePanel calls: got %d, want 1", counting.compileCalls)
	}
	if counting.scratchCalls != 2 {
		t.Fatalf("NewSimulationScratch calls: got %d, want 2", counting.scratchCalls)
	}
	if counting.streamCalls != 2 {
		t.Fatalf("ForEachCompiledProduct calls: got %d, want 2", counting.streamCalls)
	}
	if counting.scratchCompiledCalls != 0 {
		t.Fatalf("SimulateCompiledWithScratch fallback calls: got %d, want 0", counting.scratchCompiledCalls)
	}
	if counting.compiledCalls != 0 {
		t.Fatalf("SimulateCompiled fallback calls: got %d, want 0", counting.compiledCalls)
	}
	if counting.batchCalls != 0 {
		t.Fatalf("SimulateBatch fallback calls: got %d, want 0", counting.batchCalls)
	}
}

func TestForEachProduct_UsesStreamingCompiledSimulatorPerWorker(t *testing.T) {
	fn := "pipe_streaming_compiled.fa"
	if err := os.WriteFile(fn, []byte(">s1\nACGTACAAAAGGTACC\n>s2\nACGTACAAAAGGTACC\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(fn) }()

	pairs := []primer.Pair{{
		ID:         "x",
		Forward:    "ACGTAC",
		Reverse:    "GGTACC",
		MinProduct: 6,
		MaxProduct: 100,
	}}
	counting := &countingStreamingCompiledEng{base: engine.New(engine.Config{MinLen: 6, MaxLen: 100, SeedLen: 4})}

	var n int
	err := ForEachProduct(
		context.Background(),
		Config{Threads: 2, ChunkSize: 0, Overlap: 0, Circular: false, NeedSeq: true},
		[]string{fn},
		pairs,
		counting,
		func(p engine.Product) error {
			n++
			if p.SourceFile != fn {
				t.Fatalf("SourceFile = %q, want %q", p.SourceFile, fn)
			}
			if p.Seq == "" {
				t.Fatalf("expected Seq to be filled by streaming pipeline")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("pipeline err: %v", err)
	}
	if n == 0 {
		t.Fatal("expected at least one product")
	}
	if counting.compileCalls != 1 {
		t.Fatalf("CompilePanel calls: got %d, want 1", counting.compileCalls)
	}
	if counting.scratchCalls != 2 {
		t.Fatalf("NewSimulationScratch calls: got %d, want 2", counting.scratchCalls)
	}
	if counting.streamingCalls != 2 {
		t.Fatalf("ForEachCompiledProduct calls: got %d, want 2", counting.streamingCalls)
	}
	if counting.scratchCompiledCalls != 0 {
		t.Fatalf("SimulateCompiledWithScratch fallback calls: got %d, want 0", counting.scratchCompiledCalls)
	}
	if counting.compiledCalls != 0 {
		t.Fatalf("SimulateCompiled fallback calls: got %d, want 0", counting.compiledCalls)
	}
	if counting.batchCalls != 0 {
		t.Fatalf("SimulateBatch fallback calls: got %d, want 0", counting.batchCalls)
	}
}
