// core/thermoaddons/conditions.go
package thermoaddons

import (
	"fmt"
	"math"
	"os"
	"strings"
)

// Conditions is a lightweight holder for commonly tuned wet-lab knobs.
type Conditions struct {
	AnnealC           float64 // °C
	NaM               float64 // monovalent cations, mol/L
	MgM               float64 // magnesium, mol/L
	PrimerTotalM      float64 // total primer concentration, mol/L
	SelfComplementary bool
}

// ParseConc parses "50mM", "250nM", "3uM" → mol/L.
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

// EffectiveMonovalent returns a single “Na+-equivalent” to feed into salt
// corrections. By default, we *do not* add Mg2+ (keeps current behavior),
// but you can enable an Owczarzy-lite transform via env:
//
//   IPCR_MG_EQ=owczarzy-lite  →  Na_eff = Na + 3.8*sqrt(Mg)
//
// This keeps us conservative and avoids silently changing users’ results.
// You can swap the transform later without touching thermodynamic tables.
func EffectiveMonovalent(naM, mgM float64) float64 {
	mode := strings.TrimSpace(strings.ToLower(os.Getenv("IPCR_MG_EQ")))
	if mode == "owczarzy-lite" && mgM > 0 {
		return naM + 3.8*math.Sqrt(mgM)
	}
	return naM
}
