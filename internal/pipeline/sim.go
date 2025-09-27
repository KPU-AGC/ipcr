// internal/pipeline/sim.go
package pipeline

import (
	"ipcr/internal/engine"
	"ipcr/internal/primer"
)

// Simulator is the minimal capability the pipeline needs.
// Any engine (including fakes in tests) can satisfy this.
type Simulator interface {
	SimulateBatch(seqID string, seq []byte, pairs []primer.Pair) []engine.Product
}
