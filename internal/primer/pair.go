// internal/primer/pair.go
package primer

type Pair struct {
	ID                string
	Forward           string // Primer A sequence (5'→3', binds forward strand)
	Reverse           string // Primer B sequence (5'→3', binds reverse strand)
	MinProduct        int
	MaxProduct        int
}
// ===