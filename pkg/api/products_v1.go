// pkg/api/products_v1.go
package api

// ProductV1 is the stable JSON/JSONL schema for plain amplicons.
// Keep fields, names, and types stable. Add new fields only with ",omitempty".
type ProductV1 struct {
	ExperimentID   string `json:"experiment_id"`
	SequenceID     string `json:"sequence_id"`
	Start          int    `json:"start"`
	End            int    `json:"end"`
	Length         int    `json:"length"`
	Type           string `json:"type"` // "forward" | "revcomp"
	FwdMM          int    `json:"fwd_mm,omitempty"`
	RevMM          int    `json:"rev_mm,omitempty"`
	FwdMismatchIdx []int  `json:"fwd_mm_i,omitempty"`
	RevMismatchIdx []int  `json:"rev_mm_i,omitempty"`
	Seq            string `json:"seq,omitempty"`
	SourceFile     string `json:"source_file,omitempty"`

	// NEW: optional score, used by ipcr-thermo; omitted otherwise
	Score float64 `json:"score,omitempty"`

	// Optional thermodynamic score components, emitted by NN thermo models.
	Thermo *ThermoDetailsV1 `json:"thermo,omitempty"`
}

// ThermoDetailsV1 is an optional extension object for ipcr-thermo NN modes.
type ThermoDetailsV1 struct {
	Model                   string             `json:"model"`
	SaltModel               string             `json:"salt_model"`
	AnnealTempC             float64            `json:"anneal_temp_c"`
	IUPACPolicy             string             `json:"iupac_policy"`
	MismatchPolicy          string             `json:"mismatch_policy"`
	StructurePolicy         string             `json:"structure_policy,omitempty"`
	ScoreProfile            string             `json:"score_profile,omitempty"`
	ScoreC                  float64            `json:"score_c"`
	BaseScoreC              float64            `json:"base_score_c,omitempty"`
	AmpliconAdjustmentC     float64            `json:"amplicon_adjustment_c,omitempty"`
	ExtensionLogit          float64            `json:"extension_logit,omitempty"`
	ExtensionBonusC         float64            `json:"extension_bonus_c,omitempty"`
	LengthPenaltyC          float64            `json:"length_penalty_c,omitempty"`
	BandMassBonusC          float64            `json:"band_mass_bonus_c,omitempty"`
	StructurePenaltyC       float64            `json:"structure_penalty_c,omitempty"`
	LimitingSide            string             `json:"limiting_side"`
	Fwd                     ThermoEndpointV1   `json:"fwd"`
	Rev                     ThermoEndpointV1   `json:"rev"`
	WorstHairpin            *ThermoStructureV1 `json:"worst_hairpin,omitempty"`
	WorstSelfDimer          *ThermoStructureV1 `json:"worst_self_dimer,omitempty"`
	CrossDimer              *ThermoStructureV1 `json:"cross_dimer,omitempty"`
	PanelCrossDimer         *ThermoStructureV1 `json:"panel_cross_dimer,omitempty"`
	PanelCrossDimerPenaltyC float64            `json:"panel_cross_dimer_penalty_c,omitempty"`
	PanelCrossDimerBurdenC  float64            `json:"panel_cross_dimer_burden_c,omitempty"`
	PanelCrossDimerCount    int                `json:"panel_cross_dimer_count,omitempty"`
}

// ThermoStructureV1 describes a secondary-structure competitor.
type ThermoStructureV1 struct {
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

// ThermoEndpointV1 describes a single primer-template endpoint.
type ThermoEndpointV1 struct {
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

// AnnotatedProductV1 is the stable schema for probe-annotated outputs.
type AnnotatedProductV1 struct {
	// Base
	ExperimentID   string `json:"experiment_id"`
	SequenceID     string `json:"sequence_id"`
	Start          int    `json:"start"`
	End            int    `json:"end"`
	Length         int    `json:"length"`
	Type           string `json:"type"`
	FwdMM          int    `json:"fwd_mm,omitempty"`
	RevMM          int    `json:"rev_mm,omitempty"`
	FwdMismatchIdx []int  `json:"fwd_mm_i,omitempty"`
	RevMismatchIdx []int  `json:"rev_mm_i,omitempty"`
	Seq            string `json:"seq,omitempty"`
	SourceFile     string `json:"source_file,omitempty"`

	// NEW: surface base score when present
	Score float64 `json:"score,omitempty"`

	// Probe overlay
	ProbeName   string `json:"probe_name"`
	ProbeSeq    string `json:"probe_seq"`
	ProbeFound  bool   `json:"probe_found"`
	ProbeStrand string `json:"probe_strand,omitempty"` // "+"/"-"
	ProbePos    int    `json:"probe_pos,omitempty"`
	ProbeMM     int    `json:"probe_mm,omitempty"`
	ProbeSite   string `json:"probe_site,omitempty"`
}
