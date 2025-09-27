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

	// Probe overlay
	ProbeName   string `json:"probe_name"`
	ProbeSeq    string `json:"probe_seq"`
	ProbeFound  bool   `json:"probe_found"`
	ProbeStrand string `json:"probe_strand,omitempty"` // "+"/"-"
	ProbePos    int    `json:"probe_pos,omitempty"`
	ProbeMM     int    `json:"probe_mm,omitempty"`
	ProbeSite   string `json:"probe_site,omitempty"`
}
