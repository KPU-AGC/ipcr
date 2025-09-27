// ./internal/visitors/nested.go
package visitors

import (
	"sort"

	"ipcr/internal/engine"
	"ipcr/internal/nestedoutput"
	"ipcr/internal/primer"
)

type Nested struct {
	InnerPairs  []primer.Pair
	EngineCfg   engine.Config
	RequireInner bool
}

func (v Nested) Visit(p engine.Product) (bool, nestedoutput.NestedProduct, error) {
	amp := []byte(p.Seq)
	eng := engine.New(v.EngineCfg)
	hits := eng.SimulateBatch("amplicon", amp, v.InnerPairs)

	// Pick a deterministic “best” inner: fewest total mismatches, then longest, then leftmost.
	bestIdx := -1
	bestScore := 1<<30
	for i := range hits {
		totalMM := hits[i].FwdMM + hits[i].RevMM
		if bestIdx == -1 || totalMM < bestScore ||
			(totalMM == bestScore && (hits[i].Length > hits[bestIdx].Length ||
				(hits[i].Length == hits[bestIdx].Length && hits[i].Start < hits[bestIdx].Start))) {
			bestIdx = i
			bestScore = totalMM
		}
	}

	if bestIdx == -1 {
		if v.RequireInner {
			return false, nestedoutput.NestedProduct{}, nil
		}
		return true, nestedoutput.NestedProduct{
			Product:     p,
			InnerFound:  false,
			InnerPairID: "",
		}, nil
	}

	// Stabilize by start position if multiple equal scored remain (already handled, but keep order tidy).
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Start < hits[j].Start })

	inner := hits[bestIdx]
	np := nestedoutput.NestedProduct{
		Product:      p, // outer
		InnerFound:   true,
		InnerPairID:  inner.ExperimentID,
		InnerStart:   inner.Start,  // relative to amplicon
		InnerEnd:     inner.End,    // relative to amplicon
		InnerLength:  inner.Length,
		InnerType:    inner.Type,
		InnerFwdMM:   inner.FwdMM,
		InnerRevMM:   inner.RevMM,
	}
	return true, np, nil
}
