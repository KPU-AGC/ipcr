// core/primer/self.go
package primer

// Oligo represents a single primer (5'→3') with an identifier.
type Oligo struct {
	ID  string
	Seq string
}

// SelfPairs converts single oligos to "self" pairs where Forward == Reverse.
// Using the standard engine joiner, this yields A×rc(A) (and the symmetric
// orientation) per oligo. Per-oligo min/max are unset so global bounds apply.
func SelfPairs(oligos []Oligo) []Pair {
	out := make([]Pair, 0, len(oligos))
	for _, o := range oligos {
		out = append(out, Pair{
			ID:         o.ID + "+self",
			Forward:    o.Seq,
			Reverse:    o.Seq,
			MinProduct: 0,
			MaxProduct: 0,
		})
	}
	return out
}
