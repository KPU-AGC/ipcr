// core/primer/rc.go
package primer

import "fmt"

var complement [256]byte

func init() {
	complement['A'] = 'T'
	complement['C'] = 'G'
	complement['G'] = 'C'
	complement['T'] = 'A'
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
}

// RevComp returns the reverse-complement of a normalized uppercase IUPAC DNA
// sequence. Unknown characters fail loudly instead of being converted to N.
func RevComp(seq []byte) []byte {
	out, err := RevCompStrict(seq)
	if err != nil {
		panic(err)
	}
	return out
}

// RevCompStrict returns the reverse-complement of a normalized uppercase IUPAC
// DNA sequence, or an error if any character is outside that alphabet.
func RevCompStrict(seq []byte) ([]byte, error) {
	n := len(seq)
	if n == 0 {
		return nil, nil
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		srcPos := n - 1 - i
		b := seq[srcPos]
		c := complement[b]
		if c == 0 {
			return nil, fmt.Errorf("invalid reverse-complement base %q at position %d; expected normalized uppercase IUPAC DNA", b, srcPos+1)
		}
		out[i] = c
	}
	return out, nil
}

// ===
