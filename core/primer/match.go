// internal/primer/match.go
package primer

import "bytes"

/* ----------------------- types --------------------- */

type Match struct {
	Pos         int
	Mismatches  int
	Length      int
	MismatchIdx []int // 0‑based positions in primer (5'→3') that mismatched
}

/* ---------------------- helpers -------------------- */

func isUnambiguous(p []byte) bool {
	for _, c := range p {
		if c != 'A' && c != 'C' && c != 'G' && c != 'T' {
			return false
		}
	}
	return true
}

/* --------------------------- FindMatches (cap) -------------------------- */

// capHits == 0  ➜ unlimited
// terminalWindow: N bases at the primer 3' end where mismatches are disallowed (0=allow)
func FindMatches(seq, primer []byte, maxMM, capHits int, terminalWindow int) []Match {
	pl := len(primer)
	if pl == 0 || len(seq) < pl {
		return nil
	}

	// Exact‑match fast path: SIMD'd bytes.Index jump scanning.
	// Safe with any terminalWindow because mismatches=0.
	if maxMM == 0 && isUnambiguous(primer) {
		out := make([]Match, 0, 8)
		for i := 0; ; {
			j := bytes.Index(seq[i:], primer)
			if j < 0 {
				break
			}
			pos := i + j
			out = append(out, Match{Pos: pos, Mismatches: 0, Length: pl})
			if capHits > 0 && len(out) >= capHits {
				break
			}
			i = pos + 1
		}
		return out
	}

	end := len(seq) - pl
	out := make([]Match, 0, 8)

	// cutoff index: any mismatch with j >= cutoff is disallowed
	cutoff := pl - terminalWindow
	if terminalWindow <= 0 {
		cutoff = pl + 1 // effectively disable the check
	}
	if cutoff < 0 {
		cutoff = 0
	}

window:
	for pos := 0; pos <= end; pos++ {
		mm := 0
		var idx []int
		for j := 0; j < pl; j++ {
			if !BaseMatch(seq[pos+j], primer[j]) {
				// Reject if within 3' terminal window
				if j >= cutoff {
					continue window
				}
				mm++
				idx = append(idx, j)
				if mm > maxMM {
					continue window
				}
			}
		}
		out = append(out, Match{Pos: pos, Mismatches: mm, Length: pl, MismatchIdx: idx})
		if capHits > 0 && len(out) >= capHits {
			break // early stop to cap memory
		}
	}
	return out
}
