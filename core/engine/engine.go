// core/engine/engine.go  (REPLACE)
package engine

import (
	"sort"

	"ipcr-core/primer"
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

// -------------------- Existing single-pair path (kept) ----------------------
func (e *Engine) Simulate(seqID string, seq []byte, p primer.Pair) []Product {

	a := []byte(p.Forward)
	b := []byte(p.Reverse)
	ra := primer.RevComp(a)
	rb := primer.RevComp(b)

	hc := e.cfg.HitCap
	tw := e.cfg.TerminalWindow

	fwdA := primer.FindMatches(seq, a, e.cfg.MaxMM, hc, tw)
	fwdB := primer.FindMatches(seq, b, e.cfg.MaxMM, hc, tw)
	revAraw := primer.FindMatches(seq, ra, e.cfg.MaxMM, hc, 0)
	revBraw := primer.FindMatches(seq, rb, e.cfg.MaxMM, hc, 0)

	revA := filterLeftTW(revAraw, tw)
	revB := filterLeftTW(revBraw, tw)

	return e.joinProducts(seqID, seq, p, fwdA, fwdB, revA, revB)
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
	if len(pairs) == 0 {
		return nil
	}

	// Build seeds (exact A/C/G/T only) and AC automaton
	seeds, have := BuildSeeds(pairs, e.cfg.SeedLen, e.cfg.TerminalWindow)
	nodes, _ := buildAC(seeds)

	per := make([]perPair, len(pairs))

	// Precompute primers once per pair (hoisted RC)
	fwdABytes := make([][]byte, len(pairs))
	fwdBBytes := make([][]byte, len(pairs))
	rcA := make([][]byte, len(pairs))
	rcB := make([][]byte, len(pairs))
	for i := range pairs {
		fwdABytes[i] = []byte(pairs[i].Forward)
		fwdBBytes[i] = []byte(pairs[i].Reverse)
		rcA[i] = primer.RevComp(fwdABytes[i])
		rcB[i] = primer.RevComp(fwdBBytes[i])
	}

	// per-orientation dedup (start positions)
	seenA := make([]map[int]struct{}, len(pairs))
	seenB := make([]map[int]struct{}, len(pairs))
	seena := make([]map[int]struct{}, len(pairs))
	seenb := make([]map[int]struct{}, len(pairs))
	for i := range pairs {
		seenA[i], seenB[i] = make(map[int]struct{}), make(map[int]struct{})
		seena[i], seenb[i] = make(map[int]struct{}), make(map[int]struct{})
	}

	// Verify around seed hits
	hits := scanAC(seq, nodes, seeds)
	maxMM := e.cfg.MaxMM
	tw := e.cfg.TerminalWindow

	for _, h := range hits {
		s := seeds[h.SeedIdx]
		switch s.Which {
		case 'A':
			// AC reports 'i' as the index of the last byte of the seed.
			// For a 3'-anchored suffix seed (length = len(s.Pat)), the primer start is:
			//   i - (len(seed)-1) - (primerLen - len(seed))  ==  i - SeedOffset - (len(seed)-1)
			start := h.Pos - s.SeedOffset - (len(s.Pat) - 1)
			if _, dup := seenA[s.PairIdx][start]; dup {
				break
			}
			if m, ok := verifyAt(seq, start, fwdABytes[s.PairIdx], maxMM, 0, tw); ok {
				seenA[s.PairIdx][start] = struct{}{}
				if e.cfg.HitCap == 0 || len(per[s.PairIdx].fwdA) < e.cfg.HitCap {
					per[s.PairIdx].fwdA = append(per[s.PairIdx].fwdA, m)
				}
			}
		case 'B':
			start := h.Pos - s.SeedOffset - (len(s.Pat) - 1)
			if _, dup := seenB[s.PairIdx][start]; dup {
				break
			}
			if m, ok := verifyAt(seq, start, fwdBBytes[s.PairIdx], maxMM, 0, tw); ok {
				seenB[s.PairIdx][start] = struct{}{}
				if e.cfg.HitCap == 0 || len(per[s.PairIdx].fwdB) < e.cfg.HitCap {
					per[s.PairIdx].fwdB = append(per[s.PairIdx].fwdB, m)
				}
			}
		 case 'a': // rc(A) prefix seed: primer start is i-(len(seed)-1)
			start := h.Pos - (len(s.Pat) - 1)
			if _, dup := seena[s.PairIdx][start]; dup {
				break
			}
			if m, ok := verifyAt(seq, start, rcA[s.PairIdx], maxMM, tw, 0); ok {
				seena[s.PairIdx][start] = struct{}{}
				if e.cfg.HitCap == 0 || len(per[s.PairIdx].revA) < e.cfg.HitCap {
					per[s.PairIdx].revA = append(per[s.PairIdx].revA, m)
				}
			}
		case 'b': // rc(B) prefix seed
			start := h.Pos - (len(s.Pat) - 1)
			if _, dup := seenb[s.PairIdx][start]; dup {
				break
			}
			if m, ok := verifyAt(seq, start, rcB[s.PairIdx], maxMM, tw, 0); ok {
				seenb[s.PairIdx][start] = struct{}{}
				if e.cfg.HitCap == 0 || len(per[s.PairIdx].revB) < e.cfg.HitCap {
					per[s.PairIdx].revB = append(per[s.PairIdx].revB, m)
				}
			}
		}
	}

	// Fallback for orientations lacking seeds (ambiguous primers etc.)
	for i := range pairs {
		hc := e.cfg.HitCap
		if !have[i]['A'] {
			per[i].fwdA = primer.FindMatches(seq, fwdABytes[i], e.cfg.MaxMM, hc, tw)
		}
		if !have[i]['B'] {
			per[i].fwdB = primer.FindMatches(seq, fwdBBytes[i], e.cfg.MaxMM, hc, tw)
		}
		if !have[i]['a'] {
			raw := primer.FindMatches(seq, rcA[i], e.cfg.MaxMM, hc, 0)
			per[i].revA = filterLeftTW(raw, tw)
		}
		if !have[i]['b'] {
			raw := primer.FindMatches(seq, rcB[i], e.cfg.MaxMM, hc, 0)
			per[i].revB = filterLeftTW(raw, tw)
		}
	}

	// Join per pair
	var out []Product
	for i := range pairs {
		out = append(out, e.joinProducts(seqID, seq, pairs[i],
			per[i].fwdA, per[i].fwdB, per[i].revA, per[i].revB)...)
	}
	return out
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

// -------------------------- Join & helpers ----------------------------------

func (e *Engine) joinProducts(seqID string, seq []byte, p primer.Pair,
	fwdA, fwdB, revA, revB []primer.Match,
) []Product {
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

	var out []Product

	// --- A (fwd) × rc(B) => "forward"
	bStarts := make([]int, len(revB))
	for i, m := range revB {
		bStarts[i] = m.Pos
	}
	if !sort.IntsAreSorted(bStarts) {
		sort.Ints(bStarts)
		sort.SliceStable(revB, func(i, j int) bool { return revB[i].Pos < revB[j].Pos })
	}
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
			iMin := sort.SearchInts(bStarts, lo)
			iMax := sort.Search(len(bStarts), func(i int) bool { return bStarts[i] > hi }) - 1
			if iMin < 0 {
				iMin = 0
			}
			if iMax >= len(bStarts) {
				iMax = len(bStarts) - 1
			}
			if iMin <= iMax {
				// Descending: farthest-right first (restores legacy test expectations)
				for j := iMax; j >= iMin; j-- {
					bStart := bStarts[j]
					mb := revB[j]
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
					out = append(out, Product{
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
					})
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
				iMinW := sort.SearchInts(bStarts, loWrap)
				iMaxW := sort.Search(len(bStarts), func(i int) bool { return bStarts[i] > hiWrap }) - 1
				if iMinW < 0 {
					iMinW = 0
				}
				if iMaxW >= len(bStarts) {
					iMaxW = len(bStarts) - 1
				}
				for j := iMaxW; j >= iMinW; j-- {
					bStart := bStarts[j]
					if bStart >= ma.Pos {
						continue
					}
					mb := revB[j]
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
					out = append(out, Product{
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
					})
				}
			}
		}
	}

	// --- B (fwd) × rc(A) => "revcomp"
	aStarts := make([]int, len(revA))
	for i, m := range revA {
		aStarts[i] = m.Pos
	}
	if !sort.IntsAreSorted(aStarts) {
		sort.Ints(aStarts)
		sort.SliceStable(revA, func(i, j int) bool { return revA[i].Pos < revA[j].Pos })
	}
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
			iMin := sort.SearchInts(aStarts, lo)
			iMax := sort.Search(len(aStarts), func(i int) bool { return aStarts[i] > hi }) - 1
			if iMin < 0 {
				iMin = 0
			}
			if iMax >= len(aStarts) {
				iMax = len(aStarts) - 1
			}
			if iMin <= iMax {
				for j := iMax; j >= iMin; j-- {
					aStart := aStarts[j]
					ma := revA[j]
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
					out = append(out, Product{
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
					})
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
				iMinW := sort.SearchInts(aStarts, loWrap)
				iMaxW := sort.Search(len(aStarts), func(i int) bool { return aStarts[i] > hiWrap }) - 1
				if iMinW < 0 {
					iMinW = 0
				}
				if iMaxW >= len(aStarts) {
					iMaxW = len(aStarts) - 1
				}
				for j := iMaxW; j >= iMinW; j-- {
					aStart := aStarts[j]
					if aStart >= mb.Pos {
						continue
					}
					ma := revA[j]
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
					out = append(out, Product{
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
					})
				}
			}
		}
	}
	return out
}
