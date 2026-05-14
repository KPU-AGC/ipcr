// ./internal/visitors/nested.go
package visitors

import (
	"ipcr-core/engine"
	"ipcr-core/primer"
	"ipcr/internal/nestedoutput"
	"sort"
)

type Nested struct {
	InnerPairs   []primer.Pair
	EngineCfg    engine.Config
	RequireInner bool
}

func (v Nested) Visit(p engine.Product) (bool, nestedoutput.NestedProduct, error) {
	amp := []byte(p.Seq)
	eng := engine.New(v.EngineCfg)
	hits := eng.SimulateBatch("amplicon", amp, v.InnerPairs)

	if len(hits) == 0 {
		if v.RequireInner {
			return false, nestedoutput.NestedProduct{}, nil
		}
		return true, nestedoutput.NestedProduct{
			Product:     p,
			InnerFound:  false,
			InnerPairID: "",
		}, nil
	}

	// Pick a deterministic best inner: fewest total mismatches, then longest,
	// then leftmost. Sorting first avoids stale-index bugs after reordering.
	sort.SliceStable(hits, func(i, j int) bool {
		mi := hits[i].FwdMM + hits[i].RevMM
		mj := hits[j].FwdMM + hits[j].RevMM
		if mi != mj {
			return mi < mj
		}
		if hits[i].Length != hits[j].Length {
			return hits[i].Length > hits[j].Length
		}
		if hits[i].Start != hits[j].Start {
			return hits[i].Start < hits[j].Start
		}
		if hits[i].End != hits[j].End {
			return hits[i].End < hits[j].End
		}
		return hits[i].ExperimentID < hits[j].ExperimentID
	})
	inner := hits[0]

	np := nestedoutput.NestedProduct{
		Product:     p, // outer
		InnerFound:  true,
		InnerPairID: inner.ExperimentID,
		InnerStart:  inner.Start, // relative to amplicon
		InnerEnd:    inner.End,   // relative to amplicon
		InnerLength: inner.Length,
		InnerType:   inner.Type,
		InnerFwdMM:  inner.FwdMM,
		InnerRevMM:  inner.RevMM,
	}
	return true, np, nil
}
