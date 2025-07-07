// internal/primer/rc.go
package primer

var complement = map[byte]byte{
	'A': 'T', 'C': 'G', 'G': 'C', 'T': 'A',
	'R': 'Y', 'Y': 'R', // A/G  <->  C/T
	'S': 'S', 'W': 'W', // GC   <->  GC   ; AT <-> AT
	'K': 'M', 'M': 'K',
	'B': 'V', 'V': 'B',
	'D': 'H', 'H': 'D',
	'N': 'N',
}

func RevComp(seq []byte) []byte {
	n := len(seq)
	if n == 0 {
		return nil
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		b := seq[n-1-i]
		if c, ok := complement[b]; ok {
			out[i] = c
		} else {
			out[i] = 'N'
		}
	}
	return out
}
// ===