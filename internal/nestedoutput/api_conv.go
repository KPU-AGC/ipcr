// internal/nestedoutput/api_conv.go  (NEW)
package nestedoutput

import (
	"ipcr/pkg/api"
)

// ToAPINested converts a NestedProduct into the public wire type.
func ToAPINested(np NestedProduct) api.NestedProductV1 {
	p := np.Product
	return api.NestedProductV1{
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
}
