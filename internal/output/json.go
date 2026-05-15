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
			Model:                   p.Thermo.Model,
			SaltModel:               p.Thermo.SaltModel,
			NaM:                     p.Thermo.NaM,
			MgM:                     p.Thermo.MgM,
			DntpM:                   p.Thermo.DntpM,
			EffectiveNaM:            p.Thermo.EffectiveNaM,
			FreeMgM:                 p.Thermo.FreeMgM,
			AnnealTempC:             p.Thermo.AnnealTempC,
			IUPACPolicy:             p.Thermo.IUPACPolicy,
			IUPACThermoPolicy:       p.Thermo.IUPACThermoPolicy,
			IUPACExpansionCount:     p.Thermo.IUPACExpansionCount,
			IUPACExpansionCapped:    p.Thermo.IUPACExpansionCapped,
			IUPACEffectiveVariant:   p.Thermo.IUPACEffectiveVariant,
			IUPACVariants:           toAPIIUPACVariants(p.Thermo.IUPACVariants),
			MismatchPolicy:          p.Thermo.MismatchPolicy,
			StructurePolicy:         p.Thermo.StructurePolicy,
			ScoreProfile:            p.Thermo.ScoreProfile,
			ScoreC:                  p.Thermo.ScoreC,
			BaseScoreC:              p.Thermo.BaseScoreC,
			AmpliconAdjustmentC:     p.Thermo.AmpliconAdjustmentC,
			ExtensionLogit:          p.Thermo.ExtensionLogit,
			ExtensionBonusC:         p.Thermo.ExtensionBonusC,
			LengthPenaltyC:          p.Thermo.LengthPenaltyC,
			BandMassBonusC:          p.Thermo.BandMassBonusC,
			StructurePenaltyC:       p.Thermo.StructurePenaltyC,
			LimitingSide:            p.Thermo.LimitingSide,
			Fwd:                     toAPIThermoEndpoint(p.Thermo.Fwd),
			Rev:                     toAPIThermoEndpoint(p.Thermo.Rev),
			Probe:                   toAPIProbeThermo(p.Thermo.Probe),
			WorstHairpin:            toAPIThermoStructure(p.Thermo.WorstHairpin),
			WorstSelfDimer:          toAPIThermoStructure(p.Thermo.WorstSelfDimer),
			CrossDimer:              toAPIThermoStructure(p.Thermo.CrossDimer),
			PanelCrossDimer:         toAPIThermoStructure(p.Thermo.PanelCrossDimer),
			PanelCrossDimerPenaltyC: p.Thermo.PanelCrossDimerPenaltyC,
			PanelCrossDimerBurdenC:  p.Thermo.PanelCrossDimerBurdenC,
			PanelCrossDimerCount:    p.Thermo.PanelCrossDimerCount,
		}
	}
	// Conditionally attach Score (thermo-only).
	applyScoreToAPI(&v, p)
	return v
}

func toAPIProbeThermo(src *engine.ProbeThermoDetails) *api.ProbeThermoV1 {
	if src == nil {
		return nil
	}
	return &api.ProbeThermoV1{
		Name:                  src.Name,
		Seq:                   src.Seq,
		Found:                 src.Found,
		Strand:                src.Strand,
		Pos:                   src.Pos,
		MM:                    src.MM,
		Site:                  src.Site,
		ScoreMode:             src.ScoreMode,
		MinMarginC:            src.MinMarginC,
		ScoreContributionC:    src.ScoreContributionC,
		GatePenaltyC:          src.GatePenaltyC,
		IUPACThermoPolicy:     src.IUPACThermoPolicy,
		IUPACExpansionCount:   src.IUPACExpansionCount,
		IUPACExpansionCapped:  src.IUPACExpansionCapped,
		IUPACEffectiveVariant: src.IUPACEffectiveVariant,
		TmC:                   src.TmC,
		AnnealMarginC:         src.AnnealMarginC,
		DeltaGAtAnnealKcal:    src.DeltaGAtAnnealKcal,
		MismatchPenaltyC:      src.MismatchPenaltyC,
		MismatchDeltaGKcal:    src.MismatchDeltaGKcal,
		MismatchCount:         src.MismatchCount,
		MismatchFallbackCount: src.MismatchFallbackCount,
		MismatchTripletCount:  src.MismatchTripletCount,
		MismatchPolicy:        src.MismatchPolicy,
		HasNonWatsonCrick:     src.HasNonWatsonCrick,
		UsedHeuristicAdjust:   src.UsedHeuristicAdjust,
	}
}

func toAPIIUPACVariants(src []engine.ThermoVariant) []api.ThermoIUPACVariantV1 {
	if len(src) == 0 {
		return nil
	}
	out := make([]api.ThermoIUPACVariantV1, len(src))
	for i, v := range src {
		out[i] = api.ThermoIUPACVariantV1{
			FwdVariant:        v.FwdPrimer,
			RevVariant:        v.RevPrimer,
			ScoreC:            v.ScoreC,
			BaseScoreC:        v.BaseScoreC,
			StructurePenaltyC: v.StructurePenaltyC,
			LimitingSide:      v.LimitingSide,
			FwdTmC:            v.FwdTmC,
			RevTmC:            v.RevTmC,
			FwdMarginC:        v.FwdMarginC,
			RevMarginC:        v.RevMarginC,
		}
	}
	return out
}

func toAPIThermoEndpoint(src engine.ThermoEndpoint) api.ThermoEndpointV1 {
	return api.ThermoEndpointV1{
		Side:                               src.Side,
		TmC:                                src.TmC,
		AnnealMarginC:                      src.AnnealMarginC,
		DeltaGAtAnnealKcal:                 src.DeltaGAtAnnealKcal,
		MismatchPenaltyC:                   src.MismatchPenaltyC,
		MismatchDeltaGKcal:                 src.MismatchDeltaGKcal,
		TerminalMismatchPenaltyC:           src.TerminalMismatchPenaltyC,
		TerminalMismatchDeltaGKcal:         src.TerminalMismatchDeltaGKcal,
		DanglingEndAdjustmentC:             src.DanglingEndAdjustmentC,
		DanglingEndDeltaGKcal:              src.DanglingEndDeltaGKcal,
		DanglingEndCount:                   src.DanglingEndCount,
		MismatchCount:                      src.MismatchCount,
		FivePrimeMismatchCount:             src.FivePrimeMismatchCount,
		ThreePrimeMismatchCount:            src.ThreePrimeMismatchCount,
		FivePrimeTerminalMismatchCount:     src.FivePrimeTerminalMismatchCount,
		ThreePrimeTerminalMismatchCount:    src.ThreePrimeTerminalMismatchCount,
		TerminalMismatchCount:              src.TerminalMismatchCount,
		FivePrimeTerminalMismatchPenaltyC:  src.FivePrimeTerminalMismatchPenaltyC,
		ThreePrimeTerminalMismatchPenaltyC: src.ThreePrimeTerminalMismatchPenaltyC,
		MismatchFallbackCount:              src.MismatchFallbackCount,
		MismatchTripletCount:               src.MismatchTripletCount,
		EffectiveDenomCalK:                 src.EffectiveDenomCalK,
		MismatchPolicy:                     src.MismatchPolicy,
		EndEffectPolicy:                    src.EndEffectPolicy,
		HasNonWatsonCrick:                  src.HasNonWatsonCrick,
		UsedHeuristicAdjust:                src.UsedHeuristicAdjust,
	}
}

func toAPIThermoStructure(src *engine.ThermoStructure) *api.ThermoStructureV1 {
	if src == nil {
		return nil
	}
	return &api.ThermoStructureV1{
		Kind:                        src.Kind,
		Model:                       src.Model,
		QueryA:                      src.QueryA,
		QueryB:                      src.QueryB,
		DeltaGAtAnnealKcal:          src.DeltaGAtAnnealKcal,
		TmC:                         src.TmC,
		AnnealMarginC:               src.AnnealMarginC,
		StemLen:                     src.StemLen,
		LoopLen:                     src.LoopLen,
		AStart:                      src.AStart,
		AEnd:                        src.AEnd,
		BStart:                      src.BStart,
		BEnd:                        src.BEnd,
		ThreePrimeAnchored:          src.ThreePrimeAnchored,
		BothThreePrimeAnchor:        src.BothThreePrimeAnchor,
		SegmentCount:                src.SegmentCount,
		BulgeCount:                  src.BulgeCount,
		InternalLoopCount:           src.InternalLoopCount,
		DanglingEndCount:            src.DanglingEndCount,
		LoopPenaltyKcal:             src.LoopPenaltyKcal,
		BulgePenaltyKcal:            src.BulgePenaltyKcal,
		InternalLoopPenaltyKcal:     src.InternalLoopPenaltyKcal,
		StructureDanglingDeltaGKcal: src.StructureDanglingDeltaGKcal,
		PenaltyC:                    src.PenaltyC,
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
