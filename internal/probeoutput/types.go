// internal/probeoutput/types.go
package probeoutput

import "ipcr/internal/engine"

type AnnotatedProduct struct {
	engine.Product

	ProbeName   string `json:"probe_name"`
	ProbeSeq    string `json:"probe_seq"`
	ProbeFound  bool   `json:"probe_found"`
	ProbeStrand string `json:"probe_strand,omitempty"` // "+", "-" if found
	ProbePos    int    `json:"probe_pos,omitempty"`
	ProbeMM     int    `json:"probe_mm,omitempty"`
	ProbeSite   string `json:"probe_site,omitempty"`
}

// Append probe columns to the base TSV header used by ipcr text output.
const TSVHeaderProbe = "source_file\tsequence_id\texperiment_id\tstart\tend\tlength\ttype\t" +
	"fwd_mm\trev_mm\tfwd_mm_i\trev_mm_i\t" +
	"probe_name\tprobe_seq\tprobe_found\tprobe_strand\tprobe_pos\tprobe_mm\tprobe_site"
