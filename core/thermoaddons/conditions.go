package thermoaddons

import (
	"fmt"
	"math"
	"strings"
)

type Conditions struct {
	AnnealC           float64
	NaM               float64
	MgM               float64
	PrimerTotalM      float64
	SelfComplementary bool
}

func ParseConc(s string) (float64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	unit := ""
	val := 0.0
	_, err := fmt.Sscanf(s, "%f%s", &val, &unit)
	if err != nil {
		return 0, fmt.Errorf("invalid conc %q: %w", s, err)
	}
	switch unit {
	case "m", "":
		return val, nil
	case "mm":
		return val * 1e-3, nil
	case "um", "μm":
		return val * 1e-6, nil
	case "nm":
		return val * 1e-9, nil
	default:
		return 0, fmt.Errorf("unknown unit %q in %q", unit, s)
	}
}

// Simple effective monovalent estimate; you can swap Owczarzy later.
func EffectiveMonovalent(naM, mgM float64) float64 {
	return naM + 120.0*math.Sqrt(math.Max(0, mgM))
}
