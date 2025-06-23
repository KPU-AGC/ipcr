package engine

import "ipcress-go/internal/primer"

// Config is unchanged
type Config struct {
	MaxMM       int
	Disallow3MM bool
	MinLen      int
	MaxLen      int
}

type Engine struct{ cfg Config }

func New(c Config) *Engine { return &Engine{cfg: c} }

/* -------------------------------------------------------------------------- */
/*                           reverse‑coord helper                             */
/* -------------------------------------------------------------------------- */

// rcToFwd converts a match position on the reverse‑complement back
// to the forward‑strand coordinate of its first base.
func rcToFwd(seqLen int, rc primer.Match) int {
	return seqLen - (rc.Pos + rc.Length)
}

/* -------------------------------------------------------------------------- */
/*                               Simulate                                     */
/* -------------------------------------------------------------------------- */

func (e *Engine) Simulate(seqID string, seq []byte, p primer.Pair) []Product {
	minL := p.MinProduct
	maxL := p.MaxProduct
	if minL == 0 {
		minL = e.cfg.MinLen
	}
	if maxL == 0 {
		maxL = e.cfg.MaxLen
	}

	// Pre‑compute primer byte slices
	a := []byte(p.Forward)
	b := []byte(p.Reverse)

	// Matches on both strands
	fwdA := primer.FindMatches(seq, a, e.cfg.MaxMM, e.cfg.Disallow3MM)
	fwdB := primer.FindMatches(seq, b, e.cfg.MaxMM, e.cfg.Disallow3MM)

	rc := primer.RevComp(seq)
	revA := primer.FindMatches(rc, a, e.cfg.MaxMM, e.cfg.Disallow3MM)
	revB := primer.FindMatches(rc, b, e.cfg.MaxMM, e.cfg.Disallow3MM)

	var out []Product
	seqLen := len(seq)

	/* --------------------------- A‑B  (“forward”) --------------------------- */
	for _, ma := range fwdA {
		for _, mbRC := range revB {
			bStart := rcToFwd(seqLen, mbRC)
			if bStart <= ma.Pos {
				continue // must be downstream
			}
			end := bStart + mbRC.Length
			length := end - ma.Pos
			if (minL != 0 && length < minL) || (maxL != 0 && length > maxL) {
				continue
			}
			out = append(out, Product{
				ExperimentID: p.ID,
				SequenceID:   seqID,
				Start:        ma.Pos,
				End:          end,
				Length:       length,
				Type:         "forward",
				FwdMatch:     ma,
				RevMatch: primer.Match{
					Pos:        bStart,
					Mismatches: mbRC.Mismatches,
					Length:     mbRC.Length,
				},
			})
		}
	}

	/* --------------------------- B‑A (“revcomp”) ---------------------------- */
	for _, mb := range fwdB {
		for _, maRC := range revA {
			aStart := rcToFwd(seqLen, maRC)
			if aStart <= mb.Pos {
				continue
			}
			end := aStart + maRC.Length
			length := end - mb.Pos
			if (minL != 0 && length < minL) || (maxL != 0 && length > maxL) {
				continue
			}
			out = append(out, Product{
				ExperimentID: p.ID,
				SequenceID:   seqID,
				Start:        mb.Pos,
				End:          end,
				Length:       length,
				Type:         "revcomp",
				FwdMatch:     mb, // note: forward hit is primer B
				RevMatch: primer.Match{
					Pos:        aStart,
					Mismatches: maRC.Mismatches,
					Length:     maRC.Length,
				},
			})
		}
	}

	return out
}
