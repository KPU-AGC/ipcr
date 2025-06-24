package primer

/* ----------------------- existing types (unchanged) --------------------- */

type Match struct {
	Pos        int
	Mismatches int
	Length     int
}

/* --------------------------- FindMatches (cap) -------------------------- */

// capHits == 0  ➜ unlimited
func FindMatches(seq, primer []byte, maxMM, capHits int, disallow3 bool) []Match {
	pl := len(primer)
	if pl == 0 || len(seq) < pl {
		return nil
	}
	end := len(seq) - pl
	out := make([]Match, 0, 8)

window:
	for pos := 0; pos <= end; pos++ {
		mm := 0
		for j := 0; j < pl; j++ {
			if !BaseMatch(seq[pos+j], primer[j]) {
				if disallow3 && j == pl-1 {
					continue window
				}
				mm++
				if mm > maxMM {
					continue window
				}
			}
		}
		out = append(out, Match{Pos: pos, Mismatches: mm, Length: pl})
		if capHits > 0 && len(out) >= capHits {
			break // early stop to cap memory
		}
	}
	return out
}
