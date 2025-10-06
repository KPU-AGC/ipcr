// internal/output/json.go
package output

import (
	"io"

	"ipcr-core/engine"
	"ipcr/internal/jsonutil"
	"ipcr/pkg/api"
)

// ToAPIProduct converts a domain Product to the stable wire schema (v1).
// Score is attached conditionally via applyScoreToAPI (build-tagged).
func ToAPIProduct(p engine.Product) api.ProductV1 {
	v := api.ProductV1{
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
	}
	// Conditionally attach Score (thermo-only).
	applyScoreToAPI(&v, p)
	return v
}

func toAPIProducts(list []engine.Product) []api.ProductV1 {
	out := make([]api.ProductV1, 0, len(list))
	for _, p := range list {
		out = append(out, ToAPIProduct(p))
	}
	return out
}

// WriteJSON writes a single JSON array of v1 products (pretty-indented).
func WriteJSON(w io.Writer, list []engine.Product) error {
	return jsonutil.EncodePretty(w, toAPIProducts(list))
}
