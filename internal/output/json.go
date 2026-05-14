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
	if p.Thermo != nil {
		v.Thermo = &api.ThermoDetailsV1{
			Model:          p.Thermo.Model,
			SaltModel:      p.Thermo.SaltModel,
			AnnealTempC:    p.Thermo.AnnealTempC,
			IUPACPolicy:    p.Thermo.IUPACPolicy,
			MismatchPolicy: p.Thermo.MismatchPolicy,
			ScoreC:         p.Thermo.ScoreC,
			LimitingSide:   p.Thermo.LimitingSide,
			Fwd:            toAPIThermoEndpoint(p.Thermo.Fwd),
			Rev:            toAPIThermoEndpoint(p.Thermo.Rev),
		}
	}
	// Conditionally attach Score (thermo-only).
	applyScoreToAPI(&v, p)
	return v
}

func toAPIThermoEndpoint(src engine.ThermoEndpoint) api.ThermoEndpointV1 {
	return api.ThermoEndpointV1{
		Side:                src.Side,
		TmC:                 src.TmC,
		AnnealMarginC:       src.AnnealMarginC,
		DeltaGAtAnnealKcal:  src.DeltaGAtAnnealKcal,
		MismatchPenaltyC:    src.MismatchPenaltyC,
		EffectiveDenomCalK:  src.EffectiveDenomCalK,
		MismatchPolicy:      src.MismatchPolicy,
		HasNonWatsonCrick:   src.HasNonWatsonCrick,
		UsedHeuristicAdjust: src.UsedHeuristicAdjust,
	}
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
