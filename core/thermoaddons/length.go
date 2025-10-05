package thermoaddons

import "math"

func LengthPenalty(bp int, knee int, steepness float64, maxPenalty float64) float64 {
	if bp <= knee {
		return 0
	}
	x := float64(bp)
	k := steepness
	if k <= 0 {
		k = 0.003
	}
	return maxPenalty / (1.0 + math.Exp(-k*(x-float64(knee))))
}
