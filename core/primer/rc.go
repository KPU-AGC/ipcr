// core/primer/rc.go
package primer

var complement [256]byte

func init() {
	// Canonical bases (DNA)
	complement['A'] = 'T'
	complement['C'] = 'G'
	complement['G'] = 'C'
	complement['T'] = 'A'
	complement['U'] = 'A' // RNA input support (U ↔ A)

	// IUPAC ambiguity codes (uppercase)
	complement['R'] = 'Y'
	complement['Y'] = 'R'
	complement['S'] = 'S'
	complement['W'] = 'W'
	complement['K'] = 'M'
	complement['M'] = 'K'
	complement['B'] = 'V'
	complement['V'] = 'B'
	complement['D'] = 'H'
	complement['H'] = 'D'
	complement['N'] = 'N'

	// Lowercase complements (preserve case)
	complement['a'] = 't'
	complement['c'] = 'g'
	complement['g'] = 'c'
	complement['t'] = 'a'
	complement['u'] = 'a'

	complement['r'] = 'y'
	complement['y'] = 'r'
	complement['s'] = 's'
	complement['w'] = 'w'
	complement['k'] = 'm'
	complement['m'] = 'k'
	complement['b'] = 'v'
	complement['v'] = 'b'
	complement['d'] = 'h'
	complement['h'] = 'd'
	complement['n'] = 'n'
}

func RevComp(seq []byte) []byte {
	n := len(seq)
	if n == 0 {
		return nil
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		b := seq[n-1-i]
		c := complement[b]
		if c == 0 {
			// Preserve case for unknown letters.
			if b >= 'a' && b <= 'z' {
				c = 'n'
			} else {
				c = 'N'
			}
		}
		out[i] = c
	}
	return out
}

// ===
