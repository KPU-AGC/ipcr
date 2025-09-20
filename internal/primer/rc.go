// internal/primer/rc.go
package primer

var complement [256]byte

func init() {
    complement['A'] = 'T'; complement['C'] = 'G'; complement['G'] = 'C'; complement['T'] = 'A'
    complement['R'] = 'Y'; complement['Y'] = 'R'
    complement['S'] = 'S'; complement['W'] = 'W'
    complement['K'] = 'M'; complement['M'] = 'K'
    complement['B'] = 'V'; complement['V'] = 'B'
    complement['D'] = 'H'; complement['H'] = 'D'
    complement['N'] = 'N'
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
		if c == 0 { c = 'N' }
	out[i] = c
	}
	return out
}
// ===