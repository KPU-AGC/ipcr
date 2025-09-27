// internal/primer/iupac.go
package primer

/* -------------------------- IUPAC lookup table -------------------------- */

var iupacMask [256]byte // bit0=A bit1=C bit2=G bit3=T

func init() {
	set := func(c byte, bits byte) { iupacMask[c] = bits }
	set('A', 1)           // 0001
	set('C', 2)           // 0010
	set('G', 4)           // 0100
	set('T', 8)           // 1000
	set('R', 1|4)         // A/G
	set('Y', 2|8)         // C/T
	set('S', 2|4)         // C/G
	set('W', 1|8)         // A/T
	set('K', 4|8)         // G/T
	set('M', 1|2)         // A/C
	set('B', 2|4|8)       // C/G/T
	set('D', 1|4|8)       // A/G/T
	set('H', 1|2|8)       // A/C/T
	set('V', 1|2|4)       // A/C/G
	set('N', 1|2|4|8)     // any   (primer side only)
}

/* --------------------------- BaseMatch (FAST) --------------------------- */

// baseMatch returns true if primer base `p` can pair with genome base `g`
// according to the IUPAC ambiguity codes *and* g ∈ {A,C,G,T}.
//
// A genome base of 'N' (or any non‑ACGT ASCII) is treated as a HARD mismatch.
// This prevents large N‑blocks from producing thousands of spurious hits.
func BaseMatch(g, p byte) bool {
	if g != 'A' && g != 'C' && g != 'G' && g != 'T' {
		return false // genome N or unknown char ⇒ mismatch
	}
	return iupacMask[p]&iupacMask[g] != 0
}
// ===