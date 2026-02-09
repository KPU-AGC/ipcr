package engine

import (
	"ipcr-core/primer"
	"sort"
)

// Config holds global simulation options.
type Config struct {
	MaxMM          int
	TerminalWindow int // N bases at primer 3' end where mismatches are disallowed (0=allow)
	MinLen         int
	MaxLen         int
	HitCap         int
	NeedSites      bool // only compute FwdSite/RevSite for pretty text
	SeedLen        int  // seed length for multi-pattern scan (0=auto/full-length as implemented in seed.go)
	Circular       bool // treat templates as circular if true
	AllowSoftmask  bool // allow matching within soft-masked (lowercase) reference regions
}

// Engine runs in-silico PCR against a byte slice.
type Engine struct{ cfg Config }

func New(cfg Config) *Engine { return &Engine{cfg: cfg} }

// Simulate scans one sequence against one primer pair and returns amplicons.
func (e *Engine) Simulate(seqID string, seq []byte, pair primer.Pair) []Product {
	return e.SimulateBatch(seqID, seq, []primer.Pair{pair})
}

// SimulateBatch scans one sequence against many primer pairs.
func (e *Engine) SimulateBatch(seqID string, seq []byte, pairs []primer.Pair) []Product {
	if len(pairs) == 0 || len(seq) == 0 {
		return nil
	}

	// Precompute forward and reverse-complement primer byte slices.
	fwdABytes := make([][]byte, len(pairs))
	fwdBBytes := make([][]byte, len(pairs))
	rcA := make([][]byte, len(pairs))
	rcB := make([][]byte, len(pairs))
	for i, p := range pairs {
		fwdABytes[i] = []byte(p.Forward)
		fwdBBytes[i] = []byte(p.Reverse)
		rcA[i] = primer.RevComp(fwdABytes[i])
		rcB[i] = primer.RevComp(fwdBBytes[i])
	}

	// Seed prep
	seeds, have := BuildSeeds(pairs, e.cfg.SeedLen, e.cfg.TerminalWindow)
	nodes, _ := buildAC(seeds)

	// Per-pair match buckets
	type perPair struct {
		fwdA []primer.Match
		fwdB []primer.Match
		revA []primer.Match // rc(A) on template (i.e., binds reverse strand)
		revB []primer.Match // rc(B)
	}
	per := make([]perPair, len(pairs))

	// Dedup: avoid verifying same start multiple times per pair/orientation.
	seenA := make([]map[int]struct{}, len(pairs))
	seenB := make([]map[int]struct{}, len(pairs))
	seena := make([]map[int]struct{}, len(pairs))
	seenb := make([]map[int]struct{}, len(pairs))
	for i := range pairs {
		seenA[i], seenB[i], seena[i], seenb[i] = make(map[int]struct{}), make(map[int]struct{}), make(map[int]struct{}), make(map[int]struct{})
	}

	// Verify around seed hits
	var hits []hit
	if e.cfg.AllowSoftmask {
		hits = scanACSoftmask(seq, nodes, seeds)
	} else {
		hits = scanAC(seq, nodes, seeds)
	}
	maxMM := e.cfg.MaxMM
	tw := e.cfg.TerminalWindow

	for _, h := range hits {
		s := seeds[h.SeedIdx]
		start := h.Pos - len(s.Pat) + 1 - s.SeedOffset
		if start < 0 {
			continue
		}
		// Gate on terminal window mismatch rules in verifyAt:
		// forward-oriented primers protect rightTW; reverse-oriented protect leftTW.
		switch s.Which {
		case 'A':
			if _, ok := seenA[s.PairIdx][start]; ok {
				continue
			}
			if m, ok := verifyAt(seq, start, fwdABytes[s.PairIdx], maxMM, 0, tw, e.cfg.AllowSoftmask); ok {
				per[s.PairIdx].fwdA = append(per[s.PairIdx].fwdA, m)
				seenA[s.PairIdx][start] = struct{}{}
			}
		case 'B':
			if _, ok := seenB[s.PairIdx][start]; ok {
				continue
			}
			if m, ok := verifyAt(seq, start, fwdBBytes[s.PairIdx], maxMM, 0, tw, e.cfg.AllowSoftmask); ok {
				per[s.PairIdx].fwdB = append(per[s.PairIdx].fwdB, m)
				seenB[s.PairIdx][start] = struct{}{}
			}
		case 'a':
			if _, ok := seena[s.PairIdx][start]; ok {
				continue
			}
			if m, ok := verifyAt(seq, start, rcA[s.PairIdx], maxMM, tw, 0, e.cfg.AllowSoftmask); ok {
				per[s.PairIdx].revA = append(per[s.PairIdx].revA, m)
				seena[s.PairIdx][start] = struct{}{}
			}
		case 'b':
			if _, ok := seenb[s.PairIdx][start]; ok {
				continue
			}
			if m, ok := verifyAt(seq, start, rcB[s.PairIdx], maxMM, tw, 0, e.cfg.AllowSoftmask); ok {
				per[s.PairIdx].revB = append(per[s.PairIdx].revB, m)
				seenb[s.PairIdx][start] = struct{}{}
			}
		}
	}

	// Fallback for orientations lacking seeds (ambiguous primers etc.)
	find := primer.FindMatches
	if e.cfg.AllowSoftmask {
		find = primer.FindMatchesSoftmask
	}
	for i := range pairs {
		hc := e.cfg.HitCap
		if !have[i]['A'] {
			per[i].fwdA = find(seq, fwdABytes[i], e.cfg.MaxMM, hc, tw)
		}
		if !have[i]['B'] {
			per[i].fwdB = find(seq, fwdBBytes[i], e.cfg.MaxMM, hc, tw)
		}
		if !have[i]['a'] {
			per[i].revA = filterLeftTW(find(seq, rcA[i], e.cfg.MaxMM, hc, 0), tw)
		}
		if !have[i]['b'] {
			per[i].revB = filterLeftTW(find(seq, rcB[i], e.cfg.MaxMM, hc, 0), tw)
		}
	}

	// Join matches into products (A × rc(B), B × rc(A)).
	out := make([]Product, 0, 64)
	for i, p := range pairs {
		out = append(out, e.joinProducts(seqID, seq, p, per[i].fwdA, per[i].revB, "forward")...)
		out = append(out, e.joinProducts(seqID, seq, p, per[i].fwdB, per[i].revA, "revcomp")...)
	}
	return out
}

func filterLeftTW(ms []primer.Match, tw int) []primer.Match {
	if tw <= 0 {
		return ms
	}
	out := make([]primer.Match, 0, len(ms))
	for _, m := range ms {
		ok := true
		for _, j := range m.MismatchIdx {
			if j < tw {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, m)
		}
	}
	return out
}

type startLen struct {
	Pos int
	Len int
}

type placedMatch struct {
	startLen
	MM    int
	MMIdx []int
}

func toPlaced(ms []primer.Match) []placedMatch {
	out := make([]placedMatch, len(ms))
	for i := range ms {
		out[i] = placedMatch{
			startLen: startLen{Pos: ms[i].Pos, Len: ms[i].Length},
			MM:      ms[i].Mismatches,
			MMIdx:   ms[i].MismatchIdx,
		}
	}
	return out
}

// joinProducts combines fwd matches and rev matches into amplicons.
// typ: "forward" for A×rc(B), "revcomp" for B×rc(A).
func (e *Engine) joinProducts(seqID string, seq []byte, pair primer.Pair, fwd []primer.Match, rev []primer.Match, typ string) []Product {
	if len(fwd) == 0 || len(rev) == 0 {
		return nil
	}

	// determine min/max length: per-pair override, else engine global.
	minL := pair.MinProduct
	maxL := pair.MaxProduct
	if minL == 0 {
		minL = e.cfg.MinLen
	}
	if maxL == 0 {
		maxL = e.cfg.MaxLen
	}

	// Convert to starts + mismatch metadata
	aStarts := toPlaced(fwd)
	bStarts := toPlaced(rev)

	// sort bStarts by position asc (we will scan backwards for each a)
	sort.Slice(bStarts, func(i, j int) bool { return bStarts[i].Pos < bStarts[j].Pos })

	out := make([]Product, 0, 16)

	for _, ma := range aStarts {
		alen := ma.Len

		// earliest allowed b start
		lo := ma.Pos + minL - bStarts[0].Len
		if minL <= 0 {
			lo = ma.Pos + 1
		}
		// latest allowed b start
		hi := bStarts[len(bStarts)-1].Pos
		if maxL > 0 {
			hi = ma.Pos + maxL - bStarts[0].Len
		}
		if lo < ma.Pos+1 {
			lo = ma.Pos + 1
		}

		idxLo := sort.Search(len(bStarts), func(i int) bool { return bStarts[i].Pos >= lo })
		idxHi := sort.Search(len(bStarts), func(i int) bool { return bStarts[i].Pos > hi })
		if idxLo >= idxHi {
			continue
		}

		for k := idxHi - 1; k >= idxLo; k-- {
			mb := bStarts[k]
			bStart := mb.Pos
			blen := mb.Len

			start := ma.Pos
			end := bStart + blen

			length := end - start
			if length < 0 {
				continue
			}

			if e.cfg.Circular && end > len(seq) {
				wEnd := end - len(seq)
				length = (len(seq) - start) + wEnd
				end = wEnd
			} else if end > len(seq) {
				continue
			}

			prod := Product{
				SequenceID:   seqID,
				ExperimentID: pair.ID,
				Start:        start,
				End:          end,
				Length:       length,
				Type:         typ,

				FwdMM:          ma.MM,
				RevMM:          mb.MM,
				FwdMismatchIdx: append([]int(nil), ma.MMIdx...),
				RevMismatchIdx: append([]int(nil), mb.MMIdx...),

				// Set for pretty mode below
				FwdPrimer: pair.Forward,
				RevPrimer: pair.Reverse,
			}

			if e.cfg.NeedSites {
				prod.FwdSite = string(seq[ma.Pos : ma.Pos+alen])
				prod.RevSite = string(primer.RevComp(seq[bStart : bStart+blen]))
			}

			out = append(out, prod)
		}
	}

	return out
}
