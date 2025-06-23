// internal/primer/match.go
package primer

type Match struct {
	Pos        int // 0‑based offset in genome sequence
	Mismatches int // number of non‑matching positions
	Length     int // primer length (cached for convenience)
}

// FindMatches returns every position in `seq` where `primer` can bind
// with <= maxMM mismatches.
//
// * `seq`   – uppercase genomic sequence
// * `primer`– uppercase primer (5'→3')
// * `maxMM` – maximum allowed mismatches
// * `disallow3MM` – if true, the 3' terminal base must match exactly
func FindMatches(seq, primer []byte, maxMM int, disallow3MM bool) []Match {
	pl := len(primer)
	if pl == 0 || len(seq) < pl {
		return nil
	}
	end := len(seq) - pl
	out := make([]Match, 0, 4)

window:
	for pos := 0; pos <= end; pos++ {
		mm := 0
		for j := 0; j < pl; j++ {
			b := seq[pos+j]
			if !BaseMatch(b, primer[j]) {
				if disallow3MM && j == pl-1 {
					continue window
				}
				mm++
				if mm > maxMM {
					continue window
				}
			}
		}
		out = append(out, Match{Pos: pos, Mismatches: mm, Length: pl})
	}
	return out
}
