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
	NaM                     float64          `json:"na_m,omitempty"`
	MgM                     float64          `json:"mg_m,omitempty"`
	DntpM                   float64          `json:"dntp_m,omitempty"`
	EffectiveNaM            float64          `json:"effective_na_m,omitempty"`
	FreeMgM                 float64          `json:"free_mg_m,omitempty"`
	AnnealTempC             float64          `json:"anneal_temp_c"`
	IUPACPolicy             string           `json:"iupac_policy"`
	IUPACThermoPolicy       string           `json:"iupac_thermo_policy,omitempty"`
	IUPACExpansionCount     int              `json:"iupac_expansion_count,omitempty"`
	IUPACExpansionCapped    bool             `json:"iupac_expansion_capped,omitempty"`
	IUPACEffectiveVariant   string           `json:"iupac_effective_variant,omitempty"`
	IUPACVariants           []ThermoVariant  `json:"iupac_variants,omitempty"`
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

// ThermoVariant summarizes one concrete A/C/G/T expansion of a degenerate
// primer pair under an IUPAC thermodynamics policy. It is populated only for
// enumerate mode to keep ordinary JSON output compact.
type ThermoVariant struct {
	FwdPrimer         string  `json:"fwd_primer"`
	RevPrimer         string  `json:"rev_primer"`
	ScoreC            float64 `json:"score_c"`
	BaseScoreC        float64 `json:"base_score_c,omitempty"`
	StructurePenaltyC float64 `json:"structure_penalty_c,omitempty"`
	LimitingSide      string  `json:"limiting_side,omitempty"`
	FwdTmC            float64 `json:"fwd_tm_c,omitempty"`
	RevTmC            float64 `json:"rev_tm_c,omitempty"`
	FwdMarginC        float64 `json:"fwd_margin_c,omitempty"`
	RevMarginC        float64 `json:"rev_margin_c,omitempty"`
}

// ThermoEndpoint describes one primer-template endpoint in 5'→3' primer
// coordinates. DeltaGAtAnnealKcal is an effective two-state binding term at the
// configured annealing temperature; negative values are favorable.
type ThermoEndpoint struct {
	Side                               string  `json:"side"`
	TmC                                float64 `json:"tm_c"`
	AnnealMarginC                      float64 `json:"anneal_margin_c"`
	DeltaGAtAnnealKcal                 float64 `json:"delta_g_at_anneal_kcal"`
	MismatchPenaltyC                   float64 `json:"mismatch_penalty_c"`
	MismatchDeltaGKcal                 float64 `json:"mismatch_delta_g_kcal,omitempty"`
	TerminalMismatchPenaltyC           float64 `json:"terminal_mismatch_penalty_c,omitempty"`
	TerminalMismatchDeltaGKcal         float64 `json:"terminal_mismatch_delta_g_kcal,omitempty"`
	DanglingEndAdjustmentC             float64 `json:"dangling_end_adjustment_c,omitempty"`
	DanglingEndDeltaGKcal              float64 `json:"dangling_end_delta_g_kcal,omitempty"`
	DanglingEndCount                   int     `json:"dangling_end_count,omitempty"`
	MismatchCount                      int     `json:"mismatch_count,omitempty"`
	FivePrimeMismatchCount             int     `json:"five_prime_mismatch_count,omitempty"`
	ThreePrimeMismatchCount            int     `json:"three_prime_mismatch_count,omitempty"`
	FivePrimeTerminalMismatchCount     int     `json:"five_prime_terminal_mismatch_count,omitempty"`
	ThreePrimeTerminalMismatchCount    int     `json:"three_prime_terminal_mismatch_count,omitempty"`
	TerminalMismatchCount              int     `json:"terminal_mismatch_count,omitempty"`
	FivePrimeTerminalMismatchPenaltyC  float64 `json:"five_prime_terminal_mismatch_penalty_c,omitempty"`
	ThreePrimeTerminalMismatchPenaltyC float64 `json:"three_prime_terminal_mismatch_penalty_c,omitempty"`
	MismatchFallbackCount              int     `json:"mismatch_fallback_count,omitempty"`
	MismatchTripletCount               int     `json:"mismatch_triplet_count,omitempty"`
	EffectiveDenomCalK                 float64 `json:"effective_denom_cal_per_k_mol"`
	MismatchPolicy                     string  `json:"mismatch_policy"`
	EndEffectPolicy                    string  `json:"end_effect_policy,omitempty"`
	HasNonWatsonCrick                  bool    `json:"has_non_watson_crick"`
	UsedHeuristicAdjust                bool    `json:"used_heuristic_adjust"`
}

// ThermoStructure describes a primer secondary-structure candidate used by
// nn-structure-v1. PenaltyC is the °C-equivalent competition penalty applied to
// the final score.
type ThermoStructure struct {
	Kind                        string  `json:"kind"`
	Model                       string  `json:"model,omitempty"`
	QueryA                      string  `json:"query_a,omitempty"`
	QueryB                      string  `json:"query_b,omitempty"`
	DeltaGAtAnnealKcal          float64 `json:"delta_g_at_anneal_kcal"`
	TmC                         float64 `json:"tm_c"`
	AnnealMarginC               float64 `json:"anneal_margin_c"`
	StemLen                     int     `json:"stem_len"`
	LoopLen                     int     `json:"loop_len,omitempty"`
	AStart                      int     `json:"a_start"`
	AEnd                        int     `json:"a_end"`
	BStart                      int     `json:"b_start"`
	BEnd                        int     `json:"b_end"`
	ThreePrimeAnchored          bool    `json:"three_prime_anchored"`
	BothThreePrimeAnchor        bool    `json:"both_three_prime_anchor,omitempty"`
	SegmentCount                int     `json:"segment_count,omitempty"`
	BulgeCount                  int     `json:"bulge_count,omitempty"`
	InternalLoopCount           int     `json:"internal_loop_count,omitempty"`
	DanglingEndCount            int     `json:"dangling_end_count,omitempty"`
	LoopPenaltyKcal             float64 `json:"loop_penalty_kcal,omitempty"`
	BulgePenaltyKcal            float64 `json:"bulge_penalty_kcal,omitempty"`
	InternalLoopPenaltyKcal     float64 `json:"internal_loop_penalty_kcal,omitempty"`
	StructureDanglingDeltaGKcal float64 `json:"structure_dangling_delta_g_kcal,omitempty"`
	PenaltyC                    float64 `json:"penalty_c,omitempty"`
}
