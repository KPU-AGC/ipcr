package thermoaddons

import "math"

func HairpinPenalty(seq5to3 string) float64 {
	b := []byte(seq5to3)
	n := len(b)
	maxStem := 0
	max3Prox := 0
	for i := 0; i < n; i++ {
		for j := i + 3; j < n; j++ {
			k := 0
			for i+k < j-k && j+k < n && i-k >= 0 {
				if !isWC(b[i+k], b[j-k]) {
					break
				}
				k++
			}
			if k >= 3 {
				if k > maxStem {
					maxStem = k
					max3Prox = (n - 1) - (j - k)
				} else if k == maxStem {
					p := (n - 1) - (j - k)
					if p > max3Prox {
						max3Prox = p
					}
				}
			}
		}
	}
	if maxStem == 0 {
		return 0
	}
	return float64(maxStem) * (1.0 + math.Min(float64(max3Prox)/8.0, 1.0))
}
func DimerPenalty(primerA, primerB string) float64 {
	a := []byte(primerA)
	b := []byte(primerB)
	win := 8
	if len(a) < win || len(b) < win {
		win = min(len(a), len(b))
	}
	if win < 3 {
		return 0
	}
	maxRun := 0
	for i := 0; i < win; i++ {
		if isWC(a[len(a)-1-i], b[len(b)-1-i]) {
			maxRun++
		} else {
			break
		}
	}
	if maxRun < 3 {
		return 0
	}
	return float64(maxRun*maxRun) * 0.8
}
func isWC(p, t byte) bool {
	switch p {
	case 'A', 'a':
		return t == 'T' || t == 't'
	case 'T', 't':
		return t == 'A' || t == 'a'
	case 'C', 'c':
		return t == 'G' || t == 'g'
	case 'G', 'g':
		return t == 'C' || t == 'c'
	default:
		return false
	}
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
