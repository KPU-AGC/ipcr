// internal/engine/product.go
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
}
