// internal/primer/iupac.go
package primer

var iupac = map[byte]string{
	'A': "A", 'C': "C", 'G': "G", 'T': "T",
	'R': "AG", 'Y': "CT", 'S': "GC", 'W': "AT",
	'K': "GT", 'M': "AC", 'B': "CGT", 'D': "AGT",
	'H': "ACT", 'V': "ACG", 'N': "ACGT",
}

// Example: BaseMatch('G','R') == true because R = {A,G}
func BaseMatch(g, p byte) bool {
	allowed, ok := iupac[p]
	if !ok {
		return false
	}
	for i := 0; i < len(allowed); i++ {
		if g == allowed[i] {
			return true
		}
	}
	return false
}
