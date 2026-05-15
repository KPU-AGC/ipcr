package thermo

import (
	"fmt"
	"strings"
)

const (
	IUPACThermoPolicyStrict    = "strict"
	IUPACThermoPolicyWorst     = "worst"
	IUPACThermoPolicyBest      = "best"
	IUPACThermoPolicyMean      = "mean"
	IUPACThermoPolicyEnumerate = "enumerate"
)

func ParseIUPACThermoPolicy(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return IUPACThermoPolicyWorst, nil
	}
	switch s {
	case IUPACThermoPolicyStrict, IUPACThermoPolicyWorst, IUPACThermoPolicyBest, IUPACThermoPolicyMean, IUPACThermoPolicyEnumerate:
		return s, nil
	default:
		return "", fmt.Errorf("unknown IUPAC thermo policy %q (expected strict | worst | best | mean | enumerate)", s)
	}
}

func IsStrictACGT(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 'A', 'C', 'G', 'T', 'a', 'c', 'g', 't':
		default:
			return false
		}
	}
	return true
}

func ExpandIUPAC(seq string, max int) ([]string, bool, error) {
	seq = strings.ToUpper(strings.TrimSpace(seq))
	if seq == "" {
		return nil, false, fmt.Errorf("empty IUPAC sequence")
	}
	out := []string{""}
	capped := false
	for i := 0; i < len(seq); i++ {
		bases, ok := iupacBases(seq[i])
		if !ok {
			return nil, false, fmt.Errorf("unsupported IUPAC base %q at position %d", seq[i], i)
		}
		next := make([]string, 0, len(out)*len(bases))
		for _, prefix := range out {
			for _, b := range bases {
				if max > 0 && len(next) >= max {
					capped = true
					break
				}
				next = append(next, prefix+string(b))
			}
			if capped && max > 0 && len(next) >= max {
				break
			}
		}
		out = next
	}
	return out, capped, nil
}

func iupacBases(b byte) ([]byte, bool) {
	switch b {
	case 'A':
		return []byte{'A'}, true
	case 'C':
		return []byte{'C'}, true
	case 'G':
		return []byte{'G'}, true
	case 'T':
		return []byte{'T'}, true
	case 'R':
		return []byte{'A', 'G'}, true
	case 'Y':
		return []byte{'C', 'T'}, true
	case 'S':
		return []byte{'C', 'G'}, true
	case 'W':
		return []byte{'A', 'T'}, true
	case 'K':
		return []byte{'G', 'T'}, true
	case 'M':
		return []byte{'A', 'C'}, true
	case 'B':
		return []byte{'C', 'G', 'T'}, true
	case 'D':
		return []byte{'A', 'G', 'T'}, true
	case 'H':
		return []byte{'A', 'C', 'T'}, true
	case 'V':
		return []byte{'A', 'C', 'G'}, true
	case 'N':
		return []byte{'A', 'C', 'G', 'T'}, true
	default:
		return nil, false
	}
}
