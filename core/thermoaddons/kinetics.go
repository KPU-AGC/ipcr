package thermoaddons

import "math"

func ExtensionProb(marginC, alpha float64) float64 {
	return 1.0 / (1.0 + math.Exp(-alpha*marginC))
}
func Logit(p float64) float64 {
	if p <= 1e-9 {
		return -20
	}
	if p >= 1-1e-9 {
		return 20
	}
	return math.Log(p / (1.0 - p))
}
