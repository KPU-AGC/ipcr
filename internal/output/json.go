// internal/output/json.go
package output

import (
	"encoding/json"
	"io"

	"ipcr-core/engine"
	"ipcr/pkg/api"
)

// ToAPIProduct converts a domain Product to the stable wire schema (v1).
func ToAPIProduct(p engine.Product) api.ProductV1 {
	return api.ProductV1{
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
}

func ToAPIProducts(list []engine.Product) []api.ProductV1 {
	out := make([]api.ProductV1, 0, len(list))
	for _, p := range list {
		out = append(out, ToAPIProduct(p))
	}
	return out
}

// WriteJSON encodes Products using the stable wire schema (v1).
func WriteJSON(w io.Writer, list []engine.Product) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(ToAPIProducts(list))
}
