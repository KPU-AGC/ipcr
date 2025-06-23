// internal/primer/mismatch.go
package primer

func MismatchCount(g, p []byte) int {
	if len(g) != len(p) {
		panic("MismatchCount: length mismatch")
	}
	mm := 0
	for i := 0; i < len(p); i++ {
		if !BaseMatch(g[i], p[i]) {
			mm++
		}
	}
	return mm
}
