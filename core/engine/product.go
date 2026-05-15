// core/engine/product.go
package engine

type Product struct {
	ExperimentID string `json:"experiment_id"`
	SequenceID   string `json:"sequence_id"`
	Start        int    `json:"start"`
	End          int    `json:"end"`
	Length       int    `json:"length"`
	Type         string `json:"type"`

	// mismatch summary & positions (primer 5'→3', 0-based)
	FwdMM          int   `json:"fwd_mm,omitempty"`
	RevMM          int   `json:"rev_mm,omitempty"`
	FwdMismatchIdx []int `json:"fwd_mismatch_idx,omitempty"`
	RevMismatchIdx []int `json:"rev_mismatch_idx,omitempty"`

	// pretty support: primer seqs and matching target sites (in the same 5'→3' orientation)
	FwdPrimer string `json:"-"`
	RevPrimer string `json:"-"`
	FwdSite   string `json:"-"`
	RevSite   string `json:"-"`

	// Optional amplicon sequence
	Seq string `json:"seq,omitempty"`

	// Optional score (thermo / realistic mode). Higher is better. The numeric
	// meaning depends on the selected thermo model; see Thermo for components.
	Score float64 `json:"score,omitempty"`

	// Optional thermodynamic score components. Populated by ipcr-thermo NN modes.
	Thermo *ThermoDetails `json:"thermo,omitempty"`

	SourceFile string `json:"source_file"`
}

// ThermoDetails contains interpretable thermodynamic score components for a
// product. It is intentionally model-labelled because legacy heuristic scores
// and NN-derived scores are not numerically comparable.
type ThermoDetails struct {
	Model                   string           `json:"model"`
	SaltModel               string           `json:"salt_model"`
	AnnealTempC             float64          `json:"anneal_temp_c"`
	IUPACPolicy             string           `json:"iupac_policy"`
	MismatchPolicy          string           `json:"mismatch_policy"`
	StructurePolicy         string           `json:"structure_policy,omitempty"`
	ScoreProfile            string           `json:"score_profile,omitempty"`
	ScoreC                  float64          `json:"score_c"`
	BaseScoreC              float64          `json:"base_score_c,omitempty"`
	AmpliconAdjustmentC     float64          `json:"amplicon_adjustment_c,omitempty"`
	ExtensionLogit          float64          `json:"extension_logit,omitempty"`
	ExtensionBonusC         float64          `json:"extension_bonus_c,omitempty"`
	LengthPenaltyC          float64          `json:"length_penalty_c,omitempty"`
	BandMassBonusC          float64          `json:"band_mass_bonus_c,omitempty"`
	StructurePenaltyC       float64          `json:"structure_penalty_c,omitempty"`
	LimitingSide            string           `json:"limiting_side"`
	Fwd                     ThermoEndpoint   `json:"fwd"`
	Rev                     ThermoEndpoint   `json:"rev"`
	WorstHairpin            *ThermoStructure `json:"worst_hairpin,omitempty"`
	WorstSelfDimer          *ThermoStructure `json:"worst_self_dimer,omitempty"`
	CrossDimer              *ThermoStructure `json:"cross_dimer,omitempty"`
	PanelCrossDimer         *ThermoStructure `json:"panel_cross_dimer,omitempty"`
	PanelCrossDimerPenaltyC float64          `json:"panel_cross_dimer_penalty_c,omitempty"`
	PanelCrossDimerBurdenC  float64          `json:"panel_cross_dimer_burden_c,omitempty"`
	PanelCrossDimerCount    int              `json:"panel_cross_dimer_count,omitempty"`
}

// ThermoEndpoint describes one primer-template endpoint in 5'→3' primer
// coordinates. DeltaGAtAnnealKcal is an effective two-state binding term at the
// configured annealing temperature; negative values are favorable.
type ThermoEndpoint struct {
	Side                string  `json:"side"`
	TmC                 float64 `json:"tm_c"`
	AnnealMarginC       float64 `json:"anneal_margin_c"`
	DeltaGAtAnnealKcal  float64 `json:"delta_g_at_anneal_kcal"`
	MismatchPenaltyC    float64 `json:"mismatch_penalty_c"`
	EffectiveDenomCalK  float64 `json:"effective_denom_cal_per_k_mol"`
	MismatchPolicy      string  `json:"mismatch_policy"`
	HasNonWatsonCrick   bool    `json:"has_non_watson_crick"`
	UsedHeuristicAdjust bool    `json:"used_heuristic_adjust"`
}

// ThermoStructure describes a primer secondary-structure candidate used by
// nn-structure-v1. PenaltyC is the °C-equivalent competition penalty applied to
// the final score.
type ThermoStructure struct {
	Kind                 string  `json:"kind"`
	QueryA               string  `json:"query_a,omitempty"`
	QueryB               string  `json:"query_b,omitempty"`
	DeltaGAtAnnealKcal   float64 `json:"delta_g_at_anneal_kcal"`
	TmC                  float64 `json:"tm_c"`
	AnnealMarginC        float64 `json:"anneal_margin_c"`
	StemLen              int     `json:"stem_len"`
	LoopLen              int     `json:"loop_len,omitempty"`
	AStart               int     `json:"a_start"`
	AEnd                 int     `json:"a_end"`
	BStart               int     `json:"b_start"`
	BEnd                 int     `json:"b_end"`
	ThreePrimeAnchored   bool    `json:"three_prime_anchored"`
	BothThreePrimeAnchor bool    `json:"both_three_prime_anchor,omitempty"`
	PenaltyC             float64 `json:"penalty_c,omitempty"`
}
