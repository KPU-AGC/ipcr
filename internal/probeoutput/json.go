package probeoutput

import (
	"io"
	"ipcr/internal/jsonutil"
	"ipcr/pkg/api"
)

func ToAPIAnnotated(ap AnnotatedProduct) api.AnnotatedProductV1 {
	p := ap.Product
	return api.AnnotatedProductV1{
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

		ProbeName:   ap.ProbeName,
		ProbeSeq:    ap.ProbeSeq,
		ProbeFound:  ap.ProbeFound,
		ProbeStrand: ap.ProbeStrand,
		ProbePos:    ap.ProbePos,
		ProbeMM:     ap.ProbeMM,
		ProbeSite:   ap.ProbeSite,
	}
}

func ToAPIAnnotatedSlice(list []AnnotatedProduct) []api.AnnotatedProductV1 {
	out := make([]api.AnnotatedProductV1, 0, len(list))
	for _, ap := range list {
		out = append(out, ToAPIAnnotated(ap))
	}
	return out
}

// WriteJSON encodes AnnotatedProducts using the stable wire schema (v1).
func WriteJSON(w io.Writer, list []AnnotatedProduct) error {
	return jsonutil.EncodePretty(w, ToAPIAnnotatedSlice(list))
}
