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

// rcToFwd converts reverse complement coordinates to forward strand.
func rcToFwd(seqLen int, m primer.Match) int {
	return seqLen - (m.Pos + m.Length)
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
	hc := e.cfg.HitCap
	tw := e.cfg.TerminalWindow

	fwdA := primer.FindMatches(seq, a, e.cfg.MaxMM, hc, tw)
	fwdB := primer.FindMatches(seq, b, e.cfg.MaxMM, hc, tw)
	rc := primer.RevComp(seq)
	revA := primer.FindMatches(rc, a, e.cfg.MaxMM, hc, tw)
	revB := primer.FindMatches(rc, b, e.cfg.MaxMM, hc, tw)

	var out []Product
	slen := len(seq)
	alen := len(a)
	blen := len(b)

	// A‑B forward
	for _, ma := range fwdA {
		for _, mbRC := range revB {
			bStart := rcToFwd(slen, mbRC)
			if bStart <= ma.Pos {
				continue
			}
			end := bStart + mbRC.Length
			length := end - ma.Pos
			if (minL != 0 && length < minL) || (maxL != 0 && length > maxL) {
				continue
			}
			// capture sites for pretty (primer orientation)
			fwdSite := ""
			revSite := ""
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
				RevMM:          mbRC.Mismatches,
				FwdMismatchIdx: ma.MismatchIdx,
				RevMismatchIdx: mbRC.MismatchIdx,
				FwdPrimer:      p.Forward,
				RevPrimer:      p.Reverse,
				FwdSite:        fwdSite,
				RevSite:        revSite,
			})
		}
	}

	// B‑A revcomp
	for _, mb := range fwdB {
		for _, maRC := range revA {
			aStart := rcToFwd(slen, maRC)
			if aStart <= mb.Pos {
				continue
			}
			end := aStart + maRC.Length
			length := end - mb.Pos
			if (minL != 0 && length < minL) || (maxL != 0 && length > maxL) {
				continue
			}
			// capture sites for pretty (primer orientation)
			fwdSite := ""
			revSite := ""
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
				RevMM:          maRC.Mismatches,
				FwdMismatchIdx: mb.MismatchIdx,
				RevMismatchIdx: maRC.MismatchIdx,
				FwdPrimer:      p.Reverse, // in revcomp, forward primer is B
				RevPrimer:      p.Forward, // reverse primer is A
				FwdSite:        fwdSite,
				RevSite:        revSite,
			})
		}
	}

	return out
}
