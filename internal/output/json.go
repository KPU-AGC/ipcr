// internal/output/json.go
package output

import (
	"encoding/json"
	"io"

	"ipcr-core/engine"
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

// WriteJSON writes a JSON array of products. (Uses ToAPIProduct.)
func WriteJSON(w io.Writer, products []engine.Product) error {
	enc := json.NewEncoder(w)
	// encode as a JSON array
	if _, err := w.Write([]byte("[")); err != nil {
		return err
	}
	for i, p := range products {
		if i > 0 {
			if _, err := w.Write([]byte(",")); err != nil {
				return err
			}
		}
		if err := enc.Encode(ToAPIProduct(p)); err != nil {
			return err
		}
	}
	_, err := w.Write([]byte("]"))
	return err
}
