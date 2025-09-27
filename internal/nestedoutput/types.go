// ./internal/nestedoutput/types.go
package nestedoutput

import "ipcr/internal/engine"

type NestedProduct struct {
	// Outer
	engine.Product

	// Inner summary (relative to Product.Seq)
	InnerFound  bool   `json:"inner_found"`
	InnerPairID string `json:"inner_experiment_id,omitempty"`
	InnerStart  int    `json:"inner_start,omitempty"`
	InnerEnd    int    `json:"inner_end,omitempty"`
	InnerLength int    `json:"inner_length,omitempty"`
	InnerType   string `json:"inner_type,omitempty"`
	InnerFwdMM  int    `json:"inner_fwd_mm,omitempty"`
	InnerRevMM  int    `json:"inner_rev_mm,omitempty"`
}

const TSVHeaderNested = "source_file\tsequence_id\touter_experiment_id\touter_start\touter_end\touter_length\touter_type\t" +
	"inner_experiment_id\tinner_found\tinner_start\tinner_end\tinner_length\tinner_type\tinner_fwd_mm\tinner_rev_mm"
