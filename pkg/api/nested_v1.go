package api

type NestedProductV1 struct {
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

	InnerFound  bool   `json:"inner_found"`
	InnerPairID string `json:"inner_experiment_id,omitempty"`
	InnerStart  int    `json:"inner_start,omitempty"`
	InnerEnd    int    `json:"inner_end,omitempty"`
	InnerLength int    `json:"inner_length,omitempty"`
	InnerType   string `json:"inner_type,omitempty"`
	InnerFwdMM  int    `json:"inner_fwd_mm,omitempty"`
	InnerRevMM  int    `json:"inner_rev_mm,omitempty"`
}
