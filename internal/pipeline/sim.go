// internal/pipeline/sim.go
package pipeline

import (
	"ipcr-core/engine"
	"ipcr-core/primer"
)

// Simulator is the minimal capability the pipeline needs.
// Any engine (including fakes in tests) can satisfy this.
type Simulator interface {
	SimulateBatch(seqID string, seq []byte, pairs []primer.Pair) []engine.Product
}

// CompiledSimulator is the optional fast-path contract used by the real engine.
// When available, the pipeline compiles the primer panel once and shares it
// read-only across workers.
type CompiledSimulator interface {
	Simulator
	CompilePanel(pairs []primer.Pair) *engine.CompiledPanel
	SimulateCompiled(seqID string, seq []byte, cp *engine.CompiledPanel) []engine.Product
}

// ScratchCompiledSimulator is an optional compiled fast path that lets each
// worker reuse per-sequence buffers across records/chunks. Scratch values are
// worker-local and must not be shared between goroutines.
type ScratchCompiledSimulator interface {
	CompiledSimulator
	NewSimulationScratch(cp *engine.CompiledPanel) *engine.SimulationScratch
	SimulateCompiledWithScratch(seqID string, seq []byte, cp *engine.CompiledPanel, scratch *engine.SimulationScratch) []engine.Product
}

// StreamingCompiledSimulator is an optional compiled fast path that emits
// products directly instead of returning a per-record/per-chunk slice. Scratch
// values are worker-local and must not be shared between goroutines.
type StreamingCompiledSimulator interface {
	ScratchCompiledSimulator
	ForEachCompiledProduct(seqID string, seq []byte, cp *engine.CompiledPanel, scratch *engine.SimulationScratch, emit func(engine.Product) error) error
}
