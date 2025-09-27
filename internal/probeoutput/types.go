// internal/probeoutput/types.go  (REPLACE)
package probeoutput

import (
	"ipcr-core/engine"
	"ipcr/internal/output"
)

// AnnotatedProduct = base Product + probe overlay
type AnnotatedProduct struct {
	// Base product (keep named for callers that use ap.Product)
	Product engine.Product

	// Probe overlay
	ProbeName   string
	ProbeSeq    string
	ProbeFound  bool
	ProbeStrand string // "+"/"-"
	ProbePos    int
	ProbeMM     int
	ProbeSite   string
}

// Build the probe header on top of the canonical base header.
const TSVHeaderProbe = output.TSVHeader + "\t" +
	"probe_name\tprobe_seq\tprobe_found\tprobe_strand\tprobe_pos\tprobe_mm\tprobe_site"
