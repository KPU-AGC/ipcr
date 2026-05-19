package thermoaddons

import (
	"errors"
	"strings"

	"ipcr-core/thermo"
)

func TmNearestNeighbor(primer5to3 string, cond Conditions) (tmC, dH, dS float64, _ error) {
	s := strings.ToUpper(strings.TrimSpace(primer5to3))
	if len(s) < 2 {
		return 0, 0, 0, errors.New("sequence too short")
	}
	target3to5, ok := complement3to5(s)
	if !ok {
		return 0, 0, 0, errors.New("invalid base (need A/C/G/T)")
	}

	cond = cond.WithDefaults()
	if cond.SaltModel == "" {
		cond.SaltModel = thermo.SaltModelMonovalent
	}
	if isSelfComplementary(s) {
		cond.SelfComplementary = true
	}
	res, err := thermo.Tm(s, target3to5, cond.TmInput())
	if err != nil {
		return 0, 0, 0, err
	}
	return res.TmC, res.DH_kcal, res.DS_Na, nil
}

func DeltaGAt(dHkcal, dScal, tempC float64) float64 {
	tK := tempC + 273.15
	return dHkcal - (tK * dScal / 1000.0)
}

func complement3to5(s string) (string, bool) {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 'A':
			out[i] = 'T'
		case 'C':
			out[i] = 'G'
		case 'G':
			out[i] = 'C'
		case 'T':
			out[i] = 'A'
		default:
			return "", false
		}
	}
	return string(out), true
}

func reverseComplement(s string) (string, bool) {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 'A':
			out[len(s)-1-i] = 'T'
		case 'C':
			out[len(s)-1-i] = 'G'
		case 'G':
			out[len(s)-1-i] = 'C'
		case 'T':
			out[len(s)-1-i] = 'A'
		default:
			return "", false
		}
	}
	return string(out), true
}

func isSelfComplementary(s string) bool {
	rc, ok := reverseComplement(s)
	return ok && rc == s
}
