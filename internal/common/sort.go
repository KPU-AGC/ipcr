// internal/common/sort.go
package common

import (
	"ipcr/internal/engine"
	"ipcr/internal/probeoutput"
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

func SortAnnotated(list []probeoutput.AnnotatedProduct) {
	sort.Slice(list, func(i, j int) bool {
		return LessProduct(list[i].Product, list[j].Product)
	})
}
