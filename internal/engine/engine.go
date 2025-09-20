// internal/engine/engine.go
package engine

import (
	"ipcr/internal/primer"
)

// Config holds PCR simulation parameters.
type Config struct {
	MaxMM          int
	TerminalWindow int // N bases at primer 3' end where mismatches are disallowed (0=allow)
	MinLen         int
	MaxLen         int
	HitCap         int
}

// Engine runs PCR simulations with given config.
type Engine struct {
	cfg Config
}

// New creates a new Engine.
func New(c Config) *Engine {
	return &Engine{cfg: c}
}

// SetHitCap updates the hit cap after creation.
func (e *Engine) SetHitCap(n int) {
	e.cfg.HitCap = n
}

// Simulate finds PCR products for the given sequence and primer pair.
func (e *Engine) Simulate(seqID string, seq []byte, p primer.Pair) []Product {
	minL := p.MinProduct
	maxL := p.MaxProduct
	if minL == 0 {
		minL = e.cfg.MinLen
	}
	if maxL == 0 {
		maxL = e.cfg.MaxLen
	}

	a := []byte(p.Forward)
	b := []byte(p.Reverse)
	ra := primer.RevComp(a) // scan RC primers on the forward genome
	rb := primer.RevComp(b)

	hc := e.cfg.HitCap
	tw := e.cfg.TerminalWindow

	// Forward‑strand scans with original primers: 3' window applies on the right end (built into FindMatches).
	fwdA := primer.FindMatches(seq, a, e.cfg.MaxMM, hc, tw)
	fwdB := primer.FindMatches(seq, b, e.cfg.MaxMM, hc, tw)

	// Forward‑strand scans with RC primers: enforce the 3' window of the ORIGINAL primer,
	// which corresponds to the LEFT end in RC‑primer space. We therefore scan with tw=0 and
	// post‑filter by rejecting any match with a mismatch index < tw.
	revAraw := primer.FindMatches(seq, ra, e.cfg.MaxMM, hc, 0)
	revBraw := primer.FindMatches(seq, rb, e.cfg.MaxMM, hc, 0)

	filterLeftTW := func(ms []primer.Match, tw int) []primer.Match {
		if tw <= 0 {
			return ms
		}
		out := make([]primer.Match, 0, len(ms))
	outer:
		for _, m := range ms {
			for _, j := range m.MismatchIdx {
				if j < tw { // left‑side (RC space) lies within the original primer's 3' terminal window
					continue outer
				}
			}
			out = append(out, m)
		}
		return out
	}
	revA := filterLeftTW(revAraw, tw) // RC(A) with original A 3' policy
	revB := filterLeftTW(revBraw, tw) // RC(B) with original B 3' policy

	alen := len(a)
	blen := len(b)

	// flip converts mismatch indices from RC‑primer space back to the original primer orientation.
	flip := func(n int, idx []int) []int {
		if len(idx) == 0 {
			return nil
		}
		out := make([]int, len(idx))
		for i, v := range idx {
			out[i] = n - 1 - v
		}
		return out
	}

	var out []Product

	// A‑B forward: join fwdA with revB (RC(B) on forward genome).
	// Iterate revB in descending genomic order to preserve previous output ordering (keeps tests stable).
	for _, ma := range fwdA {
		for j := len(revB) - 1; j >= 0; j-- {
			mb := revB[j]
			bStart := mb.Pos
			if bStart <= ma.Pos {
				continue
			}
			end := bStart + blen
			length := end - ma.Pos
			if (minL != 0 && length < minL) || (maxL != 0 && length > maxL) {
				continue
			}

			var fwdSite, revSite string
			if ma.Pos+alen <= len(seq) {
				fwdSite = string(seq[ma.Pos : ma.Pos+alen])
			}
			if bStart+blen <= len(seq) {
				revSite = string(primer.RevComp(seq[bStart : bStart+blen]))
			}

			out = append(out, Product{
				ExperimentID:   p.ID,
				SequenceID:     seqID,
				Start:          ma.Pos,
				End:            end,
				Length:         length,
				Type:           "forward",
				FwdMM:          ma.Mismatches,
				RevMM:          mb.Mismatches,
				FwdMismatchIdx: ma.MismatchIdx,           // already in A orientation
				RevMismatchIdx: flip(blen, mb.MismatchIdx), // flip RC(B) → B orientation
				FwdPrimer:      p.Forward,
				RevPrimer:      p.Reverse,
				FwdSite:        fwdSite,
				RevSite:        revSite,
			})
		}
	}

	// B‑A revcomp: join fwdB with revA (RC(A) on forward genome).
	// Iterate revA in descending genomic order to mirror previous behavior.
	for _, mb := range fwdB {
		for j := len(revA) - 1; j >= 0; j-- {
			ma := revA[j]
			aStart := ma.Pos
			if aStart <= mb.Pos {
				continue
			}
			end := aStart + alen
			length := end - mb.Pos
			if (minL != 0 && length < minL) || (maxL != 0 && length > maxL) {
				continue
			}

			var fwdSite, revSite string
			if mb.Pos+blen <= len(seq) {
				fwdSite = string(seq[mb.Pos : mb.Pos+blen])
			}
			if aStart+alen <= len(seq) {
				revSite = string(primer.RevComp(seq[aStart : aStart+alen]))
			}

			out = append(out, Product{
				ExperimentID:   p.ID,
				SequenceID:     seqID,
				Start:          mb.Pos,
				End:            end,
				Length:         length,
				Type:           "revcomp",
				FwdMM:          mb.Mismatches,
				RevMM:          ma.Mismatches,
				FwdMismatchIdx: mb.MismatchIdx,           // already in B orientation
				RevMismatchIdx: flip(alen, ma.MismatchIdx), // flip RC(A) → A orientation
				FwdPrimer:      p.Reverse, // in revcomp, forward primer is B
				RevPrimer:      p.Forward, // reverse primer is A
				FwdSite:        fwdSite,
				RevSite:        revSite,
			})
		}
	}

	return out
}
