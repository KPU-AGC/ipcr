// core/engine/bruteforce.go
package engine

import "ipcr-core/primer"

// SimulateBatchBruteForce is a deliberately slow correctness oracle for the
// optimized SimulateBatch path. It scans every primer orientation with the
// current full-verification matcher, then reuses joinProducts so product
// construction, length filtering, circular handling, mismatch-index flipping,
// and optional site capture stay identical to the production path.
func (e *Engine) SimulateBatchBruteForce(seqID string, seq []byte, pairs []primer.Pair) []Product {
	if len(pairs) == 0 {
		return nil
	}

	per := make([]perPair, len(pairs))
	hc := e.cfg.HitCap
	tw := e.cfg.TerminalWindow

	for i := range pairs {
		fwdA := []byte(pairs[i].Forward)
		fwdB := []byte(pairs[i].Reverse)
		rcA := primer.RevComp(fwdA)
		rcB := primer.RevComp(fwdB)

		per[i].fwdA = primer.FindMatches(seq, fwdA, e.cfg.MaxMM, hc, tw)
		per[i].fwdB = primer.FindMatches(seq, fwdB, e.cfg.MaxMM, hc, tw)
		per[i].revA = filterLeftTW(primer.FindMatches(seq, rcA, e.cfg.MaxMM, hc, 0), tw)
		per[i].revB = filterLeftTW(primer.FindMatches(seq, rcB, e.cfg.MaxMM, hc, 0), tw)
	}

	var out []Product
	for i := range pairs {
		out = append(out, e.joinProducts(seqID, seq, pairs[i],
			per[i].fwdA, per[i].fwdB, per[i].revA, per[i].revB)...)
	}
	return out
}
