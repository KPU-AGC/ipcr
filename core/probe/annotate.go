// internal/probe/annotate.go
package probe

import "ipcr-core/oligo"

type Annotation struct {
	Found  bool
	Strand string // "+" or "-"
	Pos    int
	MM     int
	Site   string
}

func AnnotateAmplicon(amplicon string, probe string, maxMM int) Annotation {
	h := oligo.BestHit(amplicon, probe, maxMM)
	return Annotation{Found: h.Found, Strand: h.Strand, Pos: h.Pos, MM: h.MM, Site: h.Site}
}
