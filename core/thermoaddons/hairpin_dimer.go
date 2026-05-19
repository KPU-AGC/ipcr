// core/thermoaddons/hairpin_dimer.go
package thermoaddons

import (
	"math"

	"ipcr-core/thermo"
)

// HairpinPenalty returns a bounded °C-equivalent penalty for the strongest
// nearest-neighbor hairpin stem found in a 5'→3' single-stranded DNA segment.
// It is retained as a compatibility wrapper; new callers should use
// thermo.BestHairpin when they need ΔG/Tm components.
func HairpinPenalty(seq5to3 string) float64 {
	return HairpinPenaltyWithConditions(seq5to3, thermo.DefaultConditions())
}

func HairpinPenaltyWithConditions(seq5to3 string, cond thermo.Conditions) float64 {
	res, ok, err := thermo.BestHairpin(seq5to3, thermo.DefaultStructureOptions(cond))
	if err != nil || !ok || res.DeltaGAtAnnealKcal >= 0 {
		return 0
	}
	pen := -1.5 * res.DeltaGAtAnnealKcal
	if math.IsNaN(pen) || math.IsInf(pen, 0) || pen < 0 {
		return 0
	}
	if pen > 6.0 {
		return 6.0
	}
	return pen
}

func BestHairpin(seq5to3 string, cond thermo.Conditions) (thermo.StructureResult, bool, error) {
	return thermo.BestHairpin(seq5to3, thermo.DefaultStructureOptions(cond))
}

func BestSelfDimer(seq5to3 string, cond thermo.Conditions) (thermo.StructureResult, bool, error) {
	return thermo.BestSelfDimer(seq5to3, thermo.DefaultStructureOptions(cond))
}

func BestCrossDimer(a5to3, b5to3 string, cond thermo.Conditions) (thermo.StructureResult, bool, error) {
	return thermo.BestCrossDimer(a5to3, b5to3, thermo.DefaultStructureOptions(cond))
}
