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

const ThermoDetailsTSVHeader = "thermo_model\tsalt_model\tanneal_temp_c\tscore_profile\tbase_score_c\tfinal_score_c\tamplicon_adjustment_c\textension_logit\textension_bonus_c\tlength_penalty_c\tband_mass_bonus_c\tstructure_penalty_c\tlimiting_side\tfwd_tm_c\trev_tm_c\tfwd_margin_c\trev_margin_c\tfwd_dg_kcal\trev_dg_kcal\tfwd_mismatch_penalty_c\trev_mismatch_penalty_c\thairpin_penalty_c\tself_dimer_penalty_c\tcross_dimer_penalty_c\tpanel_cross_dimer_penalty_c\tpanel_cross_dimer_burden_c\tpanel_cross_dimer_count\tpanel_cross_dimer_partner"

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
	fields[2] = thermoFloat(t.AnnealTempC)
	fields[3] = t.ScoreProfile
	fields[4] = thermoFloat(t.BaseScoreC)
	fields[5] = thermoFloat(t.ScoreC)
	fields[6] = thermoFloat(t.AmpliconAdjustmentC)
	fields[7] = thermoFloat(t.ExtensionLogit)
	fields[8] = thermoFloat(t.ExtensionBonusC)
	fields[9] = thermoFloat(t.LengthPenaltyC)
	fields[10] = thermoFloat(t.BandMassBonusC)
	fields[11] = thermoFloat(t.StructurePenaltyC)
	fields[12] = t.LimitingSide
	fields[13] = thermoFloat(t.Fwd.TmC)
	fields[14] = thermoFloat(t.Rev.TmC)
	fields[15] = thermoFloat(t.Fwd.AnnealMarginC)
	fields[16] = thermoFloat(t.Rev.AnnealMarginC)
	fields[17] = thermoFloat(t.Fwd.DeltaGAtAnnealKcal)
	fields[18] = thermoFloat(t.Rev.DeltaGAtAnnealKcal)
	fields[19] = thermoFloat(t.Fwd.MismatchPenaltyC)
	fields[20] = thermoFloat(t.Rev.MismatchPenaltyC)
	if t.WorstHairpin != nil {
		fields[21] = thermoFloat(t.WorstHairpin.PenaltyC)
	}
	if t.WorstSelfDimer != nil {
		fields[22] = thermoFloat(t.WorstSelfDimer.PenaltyC)
	}
	if t.CrossDimer != nil {
		fields[23] = thermoFloat(t.CrossDimer.PenaltyC)
	}
	fields[24] = thermoFloat(t.PanelCrossDimerPenaltyC)
	fields[25] = thermoFloat(t.PanelCrossDimerBurdenC)
	if t.PanelCrossDimerCount > 0 {
		fields[26] = strconv.Itoa(t.PanelCrossDimerCount)
	}
	if t.PanelCrossDimer != nil {
		fields[27] = t.PanelCrossDimer.QueryA + "~" + t.PanelCrossDimer.QueryB
	}
	return strings.Join(fields, "\t")
}

func FormatRowTSVWithThermoDetails(p engine.Product) string {
	return FormatBaseRowTSV(p) + "\t" + FormatThermoDetailsTSV(p)
}

func FormatRowTSVWithScoreAndThermoDetails(p engine.Product) string {
	return FormatRowTSVWithScore(p) + "\t" + FormatThermoDetailsTSV(p)
}
