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
	Model          string         `json:"model"`
	SaltModel      string         `json:"salt_model"`
	AnnealTempC    float64        `json:"anneal_temp_c"`
	IUPACPolicy    string         `json:"iupac_policy"`
	MismatchPolicy string         `json:"mismatch_policy"`
	ScoreC         float64        `json:"score_c"`
	LimitingSide   string         `json:"limiting_side"`
	Fwd            ThermoEndpoint `json:"fwd"`
	Rev            ThermoEndpoint `json:"rev"`
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
