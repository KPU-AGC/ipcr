// ./internal/nestedoutput/json.go
package nestedoutput

import (
	"encoding/json"
	"io"

	"ipcr/pkg/api"
)

func toAPINested(np NestedProduct) api.NestedProductV1 {
	p := np.Product
	return api.NestedProductV1{
		// Outer
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
		// Inner
		InnerFound:   np.InnerFound,
		InnerPairID:  np.InnerPairID,
		InnerStart:   np.InnerStart,
		InnerEnd:     np.InnerEnd,
		InnerLength:  np.InnerLength,
		InnerType:    np.InnerType,
		InnerFwdMM:   np.InnerFwdMM,
		InnerRevMM:   np.InnerRevMM,
	}
}

func WriteJSON(w io.Writer, list []NestedProduct) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	out := make([]api.NestedProductV1, 0, len(list))
	for _, np := range list {
		out = append(out, toAPINested(np))
	}
	return enc.Encode(out)
}
