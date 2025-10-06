package common

import (
	"ipcr-core/engine"
	"sort"
)

// LessProduct defines a stable order for products (for -sort).
func LessProduct(a, b engine.Product) bool {
	if a.SequenceID != b.SequenceID {
		return a.SequenceID < b.SequenceID
	}
	if a.Start != b.Start {
		return a.Start < b.Start
	}
	if a.End != b.End {
		return a.End < b.End
	}
	if a.Type != b.Type {
		return a.Type < b.Type
	}
	return a.ExperimentID < b.ExperimentID
}

func SortProducts(ps []engine.Product) {
	sort.Slice(ps, func(i, j int) bool { return LessProduct(ps[i], ps[j]) })
}

// score-priority sort (descending), then fall back to coord order.
func LessProductByScore(a, b engine.Product) bool {
	if a.Score != b.Score {
		return a.Score > b.Score // higher first
	}
	return LessProduct(a, b)
}

func SortProductsByScore(ps []engine.Product) {
	sort.Slice(ps, func(i, j int) bool { return LessProductByScore(ps[i], ps[j]) })
}
