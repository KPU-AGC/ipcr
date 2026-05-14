package fasta

import "bytes"

func appendNormalizedSeqLine(dst []byte, line []byte) []byte {
	line = bytes.TrimSpace(line)
	for _, b := range line {
		if b >= 'a' && b <= 'z' {
			b -= 'a' - 'A'
		}
		dst = append(dst, b)
	}
	return dst
}
