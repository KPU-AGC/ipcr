// core/engine/engine.go
package engine

import (
	"ipcr-core/primer"
	"sort"
)

// Config holds PCR simulation parameters.
type Config struct {
	MaxMM          int
	TerminalWindow int // N bases at primer 3' end where mismatches are disallowed (0=allow)
	MinLen         int
	MaxLen         int
	HitCap         int
	NeedSites      bool // only compute FwdSite/RevSite for pretty text
	SeedLen        int  // seed length for multi-pattern scan (0=auto/full-length as implemented in seed.go)
	Circular       bool // treat templates as circular if true
}

// Engine runs PCR simulations with given config.
type Engine struct {
	cfg Config
}

// New creates a new Engine.
func New(c Config) *Engine { return &Engine{cfg: c} }

// SetHitCap updates the hit cap after creation.
func (e *Engine) SetHitCap(n int) { e.cfg.HitCap = n }

// -------------------- Existing single-pair path (now delegates) ------------
func (e *Engine) Simulate(seqID string, seq []byte, p primer.Pair) []Product {
	// Single source of truth: reuse the seeded batch path for one pair.
	return e.SimulateBatch(seqID, seq, []primer.Pair{p})
}

// -------------------- Seeded batch path (all pairs per chunk) --------------

type perPair struct {
	fwdA []primer.Match
	fwdB []primer.Match
	revA []primer.Match // rc(A) verified on forward genome
	revB []primer.Match // rc(B) verified on forward genome
}

// SimulateBatch scans seeds for all pairs in one pass, verifies candidates,
// then joins per pair to produce Products.
func (e *Engine) SimulateBatch(seqID string, seq []byte, pairs []primer.Pair) []Product {
	return e.SimulateCompiled(seqID, seq, e.CompilePanel(pairs))
}

func filterLeftTW(ms []primer.Match, tw int) []primer.Match {
	if tw <= 0 {
		return ms
	}
	out := make([]primer.Match, 0, len(ms))
outer:
	for _, m := range ms {
		for _, j := range m.MismatchIdx {
			if j < tw {
				continue outer
			}
		}
		out = append(out, m)
	}
	return out
}

func sortMatchesByPos(ms []primer.Match) []primer.Match {
	if matchesSortedByPos(ms) {
		return ms
	}
	sort.SliceStable(ms, func(i, j int) bool { return ms[i].Pos < ms[j].Pos })
	return ms
}

func matchesSortedByPos(ms []primer.Match) bool {
	for i := 1; i < len(ms); i++ {
		if ms[i].Pos < ms[i-1].Pos {
			return false
		}
	}
	return true
}

func lowerBoundMatchPos(ms []primer.Match, pos int) int {
	return sort.Search(len(ms), func(i int) bool { return ms[i].Pos >= pos })
}

func upperBoundMatchPos(ms []primer.Match, pos int) int {
	return sort.Search(len(ms), func(i int) bool { return ms[i].Pos > pos })
}

// -------------------------- Join & helpers ----------------------------------

func (e *Engine) joinProducts(seqID string, seq []byte, p primer.Pair,
	fwdA, fwdB, revA, revB []primer.Match,
) []Product {
	var out []Product
	_ = e.forEachJoinedProduct(seqID, seq, p, fwdA, fwdB, revA, revB, func(product Product) error {
		out = append(out, product)
		return nil
	})
	return out
}

func (e *Engine) forEachJoinedProduct(seqID string, seq []byte, p primer.Pair,
	fwdA, fwdB, revA, revB []primer.Match,
	emit func(Product) error,
) error {
	// Resolve product-length bounds (pair overrides engine cfg; 0 = unbounded)
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
	alen := len(a)
	blen := len(b)

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

	if emit == nil {
		return nil
	}

	// --- A (fwd) × rc(B) => "forward"
	revB = sortMatchesByPos(revB)
	for _, ma := range fwdA {
		last := len(seq) - blen
		lo := ma.Pos + 1 // strictly to the right
		if minL > 0 {
			lo = ma.Pos + minL - blen
			if lo <= ma.Pos {
				lo = ma.Pos + 1
			}
		}
		if lo < 0 {
			lo = 0
		}
		hi := last
		if maxL > 0 {
			hi = ma.Pos + maxL - blen
			if hi > last {
				hi = last
			}
		}
		if hi >= lo {
			iMin := lowerBoundMatchPos(revB, lo)
			iMax := upperBoundMatchPos(revB, hi) - 1
			if iMin <= iMax {
				// Descending: farthest-right first (restores legacy test expectations)
				for j := iMax; j >= iMin; j-- {
					mb := revB[j]
					bStart := mb.Pos
					end := bStart + blen
					length := end - ma.Pos
					if (minL != 0 && length < minL) || (maxL != 0 && length > maxL) {
						continue
					}

					var fwdSite, revSite string
					if e.cfg.NeedSites {
						if ma.Pos+alen <= len(seq) {
							fwdSite = string(seq[ma.Pos : ma.Pos+alen])
						}
						if bStart+blen <= len(seq) {
							revSite = string(primer.RevComp(seq[bStart : bStart+blen]))
						}
					}
					if err := emit(Product{
						ExperimentID:   p.ID,
						SequenceID:     seqID,
						Start:          ma.Pos,
						End:            end,
						Length:         length,
						Type:           "forward",
						FwdMM:          ma.Mismatches,
						RevMM:          mb.Mismatches,
						FwdMismatchIdx: ma.MismatchIdx,
						RevMismatchIdx: flip(blen, mb.MismatchIdx),
						FwdPrimer:      p.Forward,
						RevPrimer:      p.Reverse,
						FwdSite:        fwdSite,
						RevSite:        revSite,
					}); err != nil {
						return err
					}
				}
			}
		}

		// Circular wrap-around: allow rev match before forward match
		if e.cfg.Circular {
			// segment from forward to end: X; need remainder on the left to meet min/max
			X := len(seq) - ma.Pos
			loWrap := 0
			if minL > 0 {
				needed := minL - X - blen
				if needed < 0 {
					needed = 0
				}
				loWrap = needed
			}
			hiWrap := ma.Pos - 1
			if maxL > 0 {
				allowed := maxL - X - blen
				if allowed < hiWrap {
					hiWrap = allowed
				}
			}
			if hiWrap >= loWrap {
				iMinW := lowerBoundMatchPos(revB, loWrap)
				iMaxW := upperBoundMatchPos(revB, hiWrap) - 1
				for j := iMaxW; j >= iMinW; j-- {
					mb := revB[j]
					bStart := mb.Pos
					if bStart >= ma.Pos {
						continue
					}
					end := bStart + blen
					length := (len(seq) - ma.Pos) + end
					if (minL != 0 && length < minL) || (maxL != 0 && length > maxL) {
						continue
					}

					var fwdSite, revSite string
					if e.cfg.NeedSites {
						if ma.Pos+alen <= len(seq) {
							fwdSite = string(seq[ma.Pos : ma.Pos+alen])
						}
						if end <= len(seq) {
							revSite = string(primer.RevComp(seq[bStart:end]))
						}
					}
					if err := emit(Product{
						ExperimentID:   p.ID,
						SequenceID:     seqID,
						Start:          ma.Pos,
						End:            end,
						Length:         length,
						Type:           "forward",
						FwdMM:          ma.Mismatches,
						RevMM:          mb.Mismatches,
						FwdMismatchIdx: ma.MismatchIdx,
						RevMismatchIdx: flip(blen, mb.MismatchIdx),
						FwdPrimer:      p.Forward,
						RevPrimer:      p.Reverse,
						FwdSite:        fwdSite,
						RevSite:        revSite,
					}); err != nil {
						return err
					}
				}
			}
		}
	}

	// --- B (fwd) × rc(A) => "revcomp"
	revA = sortMatchesByPos(revA)
	for _, mb := range fwdB {
		last := len(seq) - alen
		lo := mb.Pos + 1
		if minL > 0 {
			lo = mb.Pos + minL - alen
			if lo <= mb.Pos {
				lo = mb.Pos + 1
			}
		}
		if lo < 0 {
			lo = 0
		}
		hi := last
		if maxL > 0 {
			hi = mb.Pos + maxL - alen
			if hi > last {
				hi = last
			}
		}
		if hi >= lo {
			iMin := lowerBoundMatchPos(revA, lo)
			iMax := upperBoundMatchPos(revA, hi) - 1
			if iMin <= iMax {
				for j := iMax; j >= iMin; j-- {
					ma := revA[j]
					aStart := ma.Pos
					end := aStart + alen
					length := end - mb.Pos
					if (minL != 0 && length < minL) || (maxL != 0 && length > maxL) {
						continue
					}

					var fwdSite, revSite string
					if e.cfg.NeedSites {
						if mb.Pos+blen <= len(seq) {
							fwdSite = string(seq[mb.Pos : mb.Pos+blen])
						}
						if aStart+alen <= len(seq) {
							revSite = string(primer.RevComp(seq[aStart : aStart+alen]))
						}
					}
					if err := emit(Product{
						ExperimentID:   p.ID,
						SequenceID:     seqID,
						Start:          mb.Pos,
						End:            end,
						Length:         length,
						Type:           "revcomp",
						FwdMM:          mb.Mismatches,
						RevMM:          ma.Mismatches,
						FwdMismatchIdx: mb.MismatchIdx,
						RevMismatchIdx: flip(alen, ma.MismatchIdx),
						FwdPrimer:      p.Reverse,
						RevPrimer:      p.Forward,
						FwdSite:        fwdSite,
						RevSite:        revSite,
					}); err != nil {
						return err
					}
				}
			}
		}

		// Circular wrap-around: allow rc(A) before B forward match
		if e.cfg.Circular {
			X := len(seq) - mb.Pos
			loWrap := 0
			if minL > 0 {
				needed := minL - X - alen
				if needed < 0 {
					needed = 0
				}
				loWrap = needed
			}
			hiWrap := mb.Pos - 1
			if maxL > 0 {
				allowed := maxL - X - alen
				if allowed < hiWrap {
					hiWrap = allowed
				}
			}
			if hiWrap >= loWrap {
				iMinW := lowerBoundMatchPos(revA, loWrap)
				iMaxW := upperBoundMatchPos(revA, hiWrap) - 1
				for j := iMaxW; j >= iMinW; j-- {
					ma := revA[j]
					aStart := ma.Pos
					if aStart >= mb.Pos {
						continue
					}
					end := aStart + alen
					length := (len(seq) - mb.Pos) + end
					if (minL != 0 && length < minL) || (maxL != 0 && length > maxL) {
						continue
					}

					var fwdSite, revSite string
					if e.cfg.NeedSites {
						if mb.Pos+blen <= len(seq) {
							fwdSite = string(seq[mb.Pos : mb.Pos+blen])
						}
						if end <= len(seq) {
							revSite = string(primer.RevComp(seq[aStart:end]))
						}
					}
					if err := emit(Product{
						ExperimentID:   p.ID,
						SequenceID:     seqID,
						Start:          mb.Pos,
						End:            end,
						Length:         length,
						Type:           "revcomp",
						FwdMM:          mb.Mismatches,
						RevMM:          ma.Mismatches,
						FwdMismatchIdx: mb.MismatchIdx,
						RevMismatchIdx: flip(alen, ma.MismatchIdx),
						FwdPrimer:      p.Reverse,
						RevPrimer:      p.Forward,
						FwdSite:        fwdSite,
						RevSite:        revSite,
					}); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
