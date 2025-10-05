package thermoaddons

import (
	"errors"
	"math"
	"strings"
)

func TmNearestNeighbor(primer5to3 string, cond Conditions) (tmC, dH, dS float64, _ error) {
	s := strings.ToUpper(strings.TrimSpace(primer5to3))
	if len(s) < 2 {
		return 0, 0, 0, errors.New("sequence too short")
	}
	dH = initDH
	dS = initDS
	for i := 0; i < len(s)-1; i++ {
		dh, okH := nnDH[s[i:i+2]]
		ds, okS := nnDS[s[i:i+2]]
		if !okH || !okS {
			return 0, 0, 0, errors.New("invalid base (need A/C/G/T)")
		}
		dH += dh
		dS += ds
	}
	if cond.SelfComplementary {
		dS += symmetryDS
	}
	naEff := EffectiveMonovalent(cond.NaM, cond.MgM)
	if naEff <= 0 {
		naEff = 1e-6
	}
	dS += 0.368 * float64(len(s)-1) * math.Log(naEff)

	ct := math.Max(cond.PrimerTotalM, 1e-12)
	cfactor := 4.0
	if cond.SelfComplementary {
		cfactor = 1.0
	}
	den := dS + Rcal*math.Log(ct/cfactor)
	tmK := (dH*1000.0)/den + 273.15
	return tmK - 273.15, dH, dS, nil
}

func DeltaGAt(dHkcal, dScal float64, tempC float64) float64 {
	tK := tempC + 273.15
	return dHkcal - (tK * dScal / 1000.0)
}
