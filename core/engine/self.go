// core/engine/self.go
package engine

import "ipcr-core/primer"

// SimulateSelf scans a set of single oligos against the reference and emits
// products where each oligo acts as both forward and reverse (A × rc(A)).
// Implementation detail: we reuse the batch engine by converting oligos to
// self-pairs (Forward == Reverse) and delegating to SimulateBatch.
func (e *Engine) SimulateSelf(seqID string, seq []byte, oligos []primer.Oligo) []Product {
	if len(oligos) == 0 {
		return nil
	}
	pairs := primer.SelfPairs(oligos)
	return e.SimulateBatch(seqID, seq, pairs)
}
