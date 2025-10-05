package nestedoutput

import (
	"ipcr-core/engine"
	"ipcr/pkg/api"
)

// ToAPINested converts a NestedProduct into the public wire type.
func ToAPINested(np NestedProduct) api.NestedProductV1 {
	p := np.Product
	v := api.NestedProductV1{
		ExperimentID:   p.ExperimentID,
		SequenceID:     p.SequenceID,
		Start:          p.Start,
		End:            p.End,
		Length:         p.Length,
		Type:           p.Type,
		FwdMM:          p.FwdMM,
		RevMM:          p.RevMM,
		FwdMismatchIdx: append([]int(nil), p.FwdMismatchIdx...),
		RevMismatchIdx: append([]int(nil), p.RevMismatchIdx...),
		Seq:            p.Seq,
		SourceFile:     p.SourceFile,

		InnerFound:  np.InnerFound,
		InnerPairID: np.InnerPairID,
		InnerStart:  np.InnerStart,
		InnerEnd:    np.InnerEnd,
		InnerLength: np.InnerLength,
		InnerType:   np.InnerType,
		InnerFwdMM:  np.InnerFwdMM,
		InnerRevMM:  np.InnerRevMM,
	}
	// Conditionally attach score (thermo-only; no-op otherwise).
	applyScoreToNested(&v, p)
	return v
}

// ToAPINestedSlice converts a slice of NestedProduct to the v1 wire schema.
func ToAPINestedSlice(list []NestedProduct) []api.NestedProductV1 {
	out := make([]api.NestedProductV1, 0, len(list))
	for _, np := range list {
		out = append(out, ToAPINested(np))
	}
	return out
}

// Make sure we keep a dependency to engine in this file for score gating signatures.
var _ engine.Product
