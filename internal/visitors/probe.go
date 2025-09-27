package visitors

import (
	"ipcr-core/engine"
	"ipcr-core/probe"
	"ipcr/internal/probeoutput"
	"strings"
)

// Probe annotates with an internal oligo and returns an AnnotatedProduct.
type Probe struct {
	Name    string
	Seq     string // 5'→3'
	MaxMM   int
	Require bool
}

func (v Probe) Visit(p engine.Product) (bool, probeoutput.AnnotatedProduct, error) {
	ann := probe.AnnotateAmplicon(p.Seq, v.Seq, v.MaxMM)
	if v.Require && !ann.Found {
		return false, probeoutput.AnnotatedProduct{}, nil
	}
	return true, probeoutput.AnnotatedProduct{
		Product:     p,
		ProbeName:   v.Name,
		ProbeSeq:    strings.ToUpper(v.Seq),
		ProbeFound:  ann.Found,
		ProbeStrand: ann.Strand,
		ProbePos:    ann.Pos,
		ProbeMM:     ann.MM,
		ProbeSite:   ann.Site,
	}, nil
}
