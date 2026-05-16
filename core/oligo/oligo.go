// internal/oligo/oligo.go
package oligo

import (
	"ipcr-core/primer"
	"strings"
)

type Hit struct {
	Found  bool
	Strand string // "+" or "-"
	Pos    int
	MM     int
	Site   string
}

// BestHit returns the best (fewest mismatches, then leftmost) hit of probe on
// either strand of amplicon, allowing up to maxMM mismatches.
func BestHit(amplicon, probe string, maxMM int) Hit {
	amp := strings.ToUpper(amplicon)
	prb := strings.ToUpper(probe)
	prbB := []byte(prb)
	rcB := primer.RevComp(prbB)

	// Exact match fast-path. Keep it only for strict A/C/G/T probes; degenerate
	// probes must go through the IUPAC-aware matcher even when maxMM is zero.
	if maxMM == 0 && isStrictACGT(prb) {
		if i := strings.Index(amp, prb); i >= 0 {
			return Hit{Found: true, Strand: "+", Pos: i, MM: 0, Site: amp[i : i+len(prb)]}
		}
		rc := string(rcB)
		if i := strings.Index(amp, rc); i >= 0 {
			return Hit{Found: true, Strand: "-", Pos: i, MM: 0, Site: amp[i : i+len(rc)]}
		}
		return Hit{}
	}

	plus := primer.FindMatches([]byte(amp), prbB, maxMM, 0, 0)
	minus := primer.FindMatches([]byte(amp), rcB, maxMM, 0, 0)

	best := Hit{}
	selectBest := func(pos, mm int, strand string, patLen int) {
		site := ""
		if end := pos + patLen; end <= len(amp) {
			site = amp[pos:end]
		}
		c := Hit{Found: true, Strand: strand, Pos: pos, MM: mm, Site: site}
		if !best.Found || c.MM < best.MM || (c.MM == best.MM && c.Pos < best.Pos) {
			best = c
		}
	}
	if len(plus) > 0 {
		bestLocal := plus[0]
		for _, h := range plus[1:] {
			if h.Mismatches < bestLocal.Mismatches || (h.Mismatches == bestLocal.Mismatches && h.Pos < bestLocal.Pos) {
				bestLocal = h
			}
		}
		selectBest(bestLocal.Pos, bestLocal.Mismatches, "+", len(prb))
	}
	if len(minus) > 0 {
		bestLocal := minus[0]
		for _, h := range minus[1:] {
			if h.Mismatches < bestLocal.Mismatches || (h.Mismatches == bestLocal.Mismatches && h.Pos < bestLocal.Pos) {
				bestLocal = h
			}
		}
		selectBest(bestLocal.Pos, bestLocal.Mismatches, "-", len(rcB))
	}
	return best
}

func isStrictACGT(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 'A', 'C', 'G', 'T':
		default:
			return false
		}
	}
	return true
}
