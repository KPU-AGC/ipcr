// internal/probe/annotate.go
package probe

import (
	"bytes"
	"strings"

	"ipcr/internal/primer"
)

type Annotation struct {
	Found   bool
	Strand  string // "+" or "-"
	Pos     int    // 0-based start within amplicon (plus orientation)
	MM      int
	Site    string // matched site (plus orientation)
}

func toUpperASCII(s string) string { return strings.ToUpper(s) }

// AnnotateAmplicon finds the best probe hit (fewest mismatches; tie -> leftmost).
// Searches for probe (5'→3') and its reverse complement.
func AnnotateAmplicon(amplicon string, probe string, maxMM int) Annotation {
	amp := []byte(toUpperASCII(amplicon))
	prb := []byte(toUpperASCII(probe))
	rc  := primer.RevComp(prb)

	best := Annotation{Found:false}

	score := func(hits []primer.Match, strand string, pat []byte) {
		if len(hits) == 0 { return }
		bestLocal := hits[0]
		for _, h := range hits[1:] {
			if h.Mismatches < bestLocal.Mismatches { bestLocal = h; continue }
			if h.Mismatches == bestLocal.Mismatches && h.Pos < bestLocal.Pos { bestLocal = h }
		}
		site := ""
		if end := bestLocal.Pos + len(pat); end <= len(amp) {
			site = string(amp[bestLocal.Pos:end])
		}
		candidate := Annotation{
			Found:  true,
			Strand: strand,
			Pos:    bestLocal.Pos,
			MM:     bestLocal.Mismatches,
			Site:   site,
		}
		// Keep the strictly better one (fewer mm), then leftmost
		replace := false
		if !best.Found { replace = true } else if candidate.MM < best.MM {
			replace = true
		} else if candidate.MM == best.MM && candidate.Pos < best.Pos {
			replace = true
		}
		if replace { best = candidate }
	}

	// Fast path: exact contains?
	if maxMM == 0 {
		if i := bytes.Index(amp, prb); i >= 0 {
			return Annotation{Found:true, Strand:"+", Pos:i, MM:0, Site:string(amp[i:i+len(prb)])}
		}
		if i := bytes.Index(amp, rc); i >= 0 {
			return Annotation{Found:true, Strand:"-", Pos:i, MM:0, Site:string(amp[i:i+len(rc)])}
		}
		return best
	}

	// Degenerate / mismatches allowed: reuse primer.FindMatches (no 3′ window)
	plusHits  := primer.FindMatches(amp, prb, maxMM, 0, 0)
	minusHits := primer.FindMatches(amp, rc,  maxMM, 0, 0)
	score(plusHits,  "+", prb)
	score(minusHits, "-", rc)
	return best
}
