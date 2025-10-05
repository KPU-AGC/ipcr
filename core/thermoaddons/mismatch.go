package thermoaddons

import "unicode"

var pairDeltaTm = map[[2]byte]float64{
	{'G', 'T'}: 2.0, {'T', 'G'}: 2.0,
	{'A', 'C'}: 4.0, {'C', 'A'}: 4.0,
	{'A', 'A'}: 5.0, {'C', 'C'}: 5.0, {'G', 'G'}: 5.0, {'T', 'T'}: 5.0,
	{'A', 'G'}: 4.0, {'G', 'A'}: 4.0, {'C', 'T'}: 4.0, {'T', 'C'}: 4.0,
}

func PosMultiplier(i, n int) float64 {
	if n <= 0 {
		return 1.0
	}
	if i >= n-3 {
		return 2.0
	}
	if i <= 2 {
		return 1.5
	}
	return 1.0
}

func PairDeltaTm(primerBase, targetBase byte) float64 {
	p := byte(unicode.ToUpper(rune(primerBase)))
	t := byte(unicode.ToUpper(rune(targetBase)))
	if p == t {
		return 0
	}
	if v, ok := pairDeltaTm[[2]byte{p, t}]; ok {
		return v
	}
	return 4.0
}

func GapPenalty(i, n int) float64 {
	base := 6.0
	return base * PosMultiplier(i, n) * 1.2
}
