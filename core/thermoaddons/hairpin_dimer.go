// core/thermoaddons/hairpin_dimer.go
package thermoaddons

// HairpinPenalty returns a small, bounded °C penalty for simple hairpins in a
// 5'→3' single-stranded DNA segment. It’s intentionally conservative and O(n^2):
// - stem >= 4, loop >= 3
// - penalty ~ 0.25 °C per stem base beyond 3, capped at 2.0 °C
func HairpinPenalty(seq5to3 string) float64 {
	b := []byte(seq5to3)
	n := len(b)
	bestStem := 0

	comp := func(x byte) byte {
		switch x {
		case 'A', 'a':
			return 'T'
		case 'T', 't', 'U', 'u':
			return 'A'
		case 'C', 'c':
			return 'G'
		case 'G', 'g':
			return 'C'
		default:
			return 0
		}
	}

	for i := 0; i < n; i++ {
		for j := i + 7; j < n; j++ { // loop >=3 → j-(i+1) >= 3 → j >= i+4; plus 3 more for stem room
			// grow stem outwards from (i) and (j)
			li, rj := i, j
			stem := 0
			for li >= 0 && rj < n && comp(b[li]) == b[rj] {
				stem++
				li--
				rj++
			}
			if stem > bestStem {
				bestStem = stem
			}
		}
	}
	if bestStem < 4 {
		return 0
	}
	pen := 0.25 * float64(bestStem-3)
	if pen > 2.0 {
		pen = 2.0
	}
	return pen
}
