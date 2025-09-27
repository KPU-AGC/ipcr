// core/primer/iupac.go
package primer

// iupacMask maps IUPAC nucleotide codes to a 4-bit mask (A=1, C=2, G=4, T=8).
// Ambiguity codes are bitwise ORs of those bases (e.g., R=A|G).
var iupacMask [256]byte

const (
	maskA = 1
	maskC = 2
	maskG = 4
	maskT = 8
)

func add(c byte, bits byte) { iupacMask[c] = bits }

func init() {
	// Canonical bases
	add('A', maskA)
	add('C', maskC)
	add('G', maskG)
	add('T', maskT)
	add('U', maskT) // treat U as T

	// Ambiguity codes (uppercase)
	add('R', maskA|maskG)       // A/G
	add('Y', maskC|maskT)       // C/T
	add('S', maskC|maskG)       // C/G
	add('W', maskA|maskT)       // A/T
	add('K', maskG|maskT)       // G/T
	add('M', maskA|maskC)       // A/C
	add('B', maskC|maskG|maskT) // C/G/T
	add('D', maskA|maskG|maskT) // A/G/T
	add('H', maskA|maskC|maskT) // A/C/T
	add('V', maskA|maskC|maskG) // A/C/G
	add('N', maskA|maskC|maskG|maskT)

	// Lowercase equivalents
	add('a', maskA)
	add('c', maskC)
	add('g', maskG)
	add('t', maskT)
	add('u', maskT)

	add('r', maskA|maskG)
	add('y', maskC|maskT)
	add('s', maskC|maskG)
	add('w', maskA|maskT)
	add('k', maskG|maskT)
	add('m', maskA|maskC)
	add('b', maskC|maskG|maskT)
	add('d', maskA|maskG|maskT)
	add('h', maskA|maskC|maskT)
	add('v', maskA|maskC|maskG)
	add('n', maskA|maskC|maskG|maskT)
}

// BaseMatch reports whether primer base p can pair with genome base g under
// IUPAC ambiguity rules. Non-ACGT genome bases (incl. 'N') are hard mismatches.
func BaseMatch(g, p byte) bool {
	if g != 'A' && g != 'C' && g != 'G' && g != 'T' {
		return false
	}
	return iupacMask[p]&iupacMask[g] != 0
}
