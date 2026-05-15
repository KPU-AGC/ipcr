package output

import (
	"fmt"
	"ipcr-core/engine"
	"strconv"
	"strings"
)

func IntsCSV(a []int) string {
	if len(a) == 0 {
		return ""
	}
	ss := make([]string, len(a))
	for i, v := range a {
		ss[i] = strconv.Itoa(v)
	}
	return strings.Join(ss, ",")
}

// FormatBaseRowTSV returns the 11 base columns (no trailing newline).
func FormatBaseRowTSV(p engine.Product) string {
	return fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%s\t%d\t%d\t%s\t%s",
		p.SourceFile, p.SequenceID, p.ExperimentID,
		p.Start, p.End, p.Length, p.Type,
		p.FwdMM, p.RevMM,
		IntsCSV(p.FwdMismatchIdx), IntsCSV(p.RevMismatchIdx),
	)
}

// NEW: append score as a trailing column (no trailing newline).
func FormatRowTSVWithScore(p engine.Product) string {
	base := FormatBaseRowTSV(p)
	return fmt.Sprintf("%s\t%g", base, p.Score)
}

const ThermoDetailsTSVHeader = "thermo_model\tsalt_model\tna_m\tmg_m\tdntp_m\teffective_na_m\tfree_mg_m\tanneal_temp_c\tscore_profile\tbase_score_c\tfinal_score_c\tamplicon_adjustment_c\textension_logit\textension_bonus_c\tlength_penalty_c\tband_mass_bonus_c\tstructure_penalty_c\tlimiting_side\tfwd_tm_c\trev_tm_c\tfwd_margin_c\trev_margin_c\tfwd_dg_kcal\trev_dg_kcal\tfwd_mismatch_penalty_c\trev_mismatch_penalty_c\tfwd_mismatch_count\trev_mismatch_count\tfwd_3p_mismatch_count\trev_3p_mismatch_count\tfwd_mismatch_fallback_count\trev_mismatch_fallback_count\tfwd_mismatch_dg_kcal\trev_mismatch_dg_kcal\tfwd_terminal_mismatch_penalty_c\trev_terminal_mismatch_penalty_c\tfwd_5p_terminal_mismatch_penalty_c\trev_5p_terminal_mismatch_penalty_c\tfwd_3p_terminal_mismatch_penalty_c\trev_3p_terminal_mismatch_penalty_c\tfwd_terminal_mismatch_dg_kcal\trev_terminal_mismatch_dg_kcal\tfwd_dangling_end_adjustment_c\trev_dangling_end_adjustment_c\tfwd_dangling_end_dg_kcal\trev_dangling_end_dg_kcal\tfwd_end_effect_policy\trev_end_effect_policy\thairpin_penalty_c\tself_dimer_penalty_c\tcross_dimer_penalty_c\tpanel_cross_dimer_penalty_c\tpanel_cross_dimer_burden_c\tpanel_cross_dimer_count\tpanel_cross_dimer_partner"

func thermoFloat(x float64) string {
	return strconv.FormatFloat(x, 'g', -1, 64)
}

// FormatThermoDetailsTSV returns optional NN thermodynamic component columns.
// Legacy/heuristic rows emit empty fields so TSV width remains stable when
// --thermo-details is requested.
func FormatThermoDetailsTSV(p engine.Product) string {
	fields := make([]string, len(strings.Split(ThermoDetailsTSVHeader, "\t")))
	if p.Thermo == nil {
		return strings.Join(fields, "\t")
	}
	t := p.Thermo
	fields[0] = t.Model
	fields[1] = t.SaltModel
	fields[2] = thermoFloat(t.NaM)
	fields[3] = thermoFloat(t.MgM)
	fields[4] = thermoFloat(t.DntpM)
	fields[5] = thermoFloat(t.EffectiveNaM)
	fields[6] = thermoFloat(t.FreeMgM)
	fields[7] = thermoFloat(t.AnnealTempC)
	fields[8] = t.ScoreProfile
	fields[9] = thermoFloat(t.BaseScoreC)
	fields[10] = thermoFloat(t.ScoreC)
	fields[11] = thermoFloat(t.AmpliconAdjustmentC)
	fields[12] = thermoFloat(t.ExtensionLogit)
	fields[13] = thermoFloat(t.ExtensionBonusC)
	fields[14] = thermoFloat(t.LengthPenaltyC)
	fields[15] = thermoFloat(t.BandMassBonusC)
	fields[16] = thermoFloat(t.StructurePenaltyC)
	fields[17] = t.LimitingSide
	fields[18] = thermoFloat(t.Fwd.TmC)
	fields[19] = thermoFloat(t.Rev.TmC)
	fields[20] = thermoFloat(t.Fwd.AnnealMarginC)
	fields[21] = thermoFloat(t.Rev.AnnealMarginC)
	fields[22] = thermoFloat(t.Fwd.DeltaGAtAnnealKcal)
	fields[23] = thermoFloat(t.Rev.DeltaGAtAnnealKcal)
	fields[24] = thermoFloat(t.Fwd.MismatchPenaltyC)
	fields[25] = thermoFloat(t.Rev.MismatchPenaltyC)
	if t.Fwd.MismatchCount > 0 {
		fields[26] = strconv.Itoa(t.Fwd.MismatchCount)
	}
	if t.Rev.MismatchCount > 0 {
		fields[27] = strconv.Itoa(t.Rev.MismatchCount)
	}
	if t.Fwd.ThreePrimeMismatchCount > 0 {
		fields[28] = strconv.Itoa(t.Fwd.ThreePrimeMismatchCount)
	}
	if t.Rev.ThreePrimeMismatchCount > 0 {
		fields[29] = strconv.Itoa(t.Rev.ThreePrimeMismatchCount)
	}
	if t.Fwd.MismatchFallbackCount > 0 {
		fields[30] = strconv.Itoa(t.Fwd.MismatchFallbackCount)
	}
	if t.Rev.MismatchFallbackCount > 0 {
		fields[31] = strconv.Itoa(t.Rev.MismatchFallbackCount)
	}
	fields[32] = thermoFloat(t.Fwd.MismatchDeltaGKcal)
	fields[33] = thermoFloat(t.Rev.MismatchDeltaGKcal)
	fields[34] = thermoFloat(t.Fwd.TerminalMismatchPenaltyC)
	fields[35] = thermoFloat(t.Rev.TerminalMismatchPenaltyC)
	fields[36] = thermoFloat(t.Fwd.FivePrimeTerminalMismatchPenaltyC)
	fields[37] = thermoFloat(t.Rev.FivePrimeTerminalMismatchPenaltyC)
	fields[38] = thermoFloat(t.Fwd.ThreePrimeTerminalMismatchPenaltyC)
	fields[39] = thermoFloat(t.Rev.ThreePrimeTerminalMismatchPenaltyC)
	fields[40] = thermoFloat(t.Fwd.TerminalMismatchDeltaGKcal)
	fields[41] = thermoFloat(t.Rev.TerminalMismatchDeltaGKcal)
	fields[42] = thermoFloat(t.Fwd.DanglingEndAdjustmentC)
	fields[43] = thermoFloat(t.Rev.DanglingEndAdjustmentC)
	fields[44] = thermoFloat(t.Fwd.DanglingEndDeltaGKcal)
	fields[45] = thermoFloat(t.Rev.DanglingEndDeltaGKcal)
	fields[46] = t.Fwd.EndEffectPolicy
	fields[47] = t.Rev.EndEffectPolicy
	if t.WorstHairpin != nil {
		fields[48] = thermoFloat(t.WorstHairpin.PenaltyC)
	}
	if t.WorstSelfDimer != nil {
		fields[49] = thermoFloat(t.WorstSelfDimer.PenaltyC)
	}
	if t.CrossDimer != nil {
		fields[50] = thermoFloat(t.CrossDimer.PenaltyC)
	}
	fields[51] = thermoFloat(t.PanelCrossDimerPenaltyC)
	fields[52] = thermoFloat(t.PanelCrossDimerBurdenC)
	if t.PanelCrossDimerCount > 0 {
		fields[53] = strconv.Itoa(t.PanelCrossDimerCount)
	}
	if t.PanelCrossDimer != nil {
		fields[54] = t.PanelCrossDimer.QueryA + "~" + t.PanelCrossDimer.QueryB
	}
	return strings.Join(fields, "\t")
}

func FormatRowTSVWithThermoDetails(p engine.Product) string {
	return FormatBaseRowTSV(p) + "\t" + FormatThermoDetailsTSV(p)
}

func FormatRowTSVWithScoreAndThermoDetails(p engine.Product) string {
	return FormatRowTSVWithScore(p) + "\t" + FormatThermoDetailsTSV(p)
}
