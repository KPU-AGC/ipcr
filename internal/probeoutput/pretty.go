// internal/probeoutput/pretty.go
package probeoutput

import "ipcr/internal/pretty"

func probeAnnotation(ap AnnotatedProduct) pretty.ProbeAnnotation {
	return pretty.ProbeAnnotation{
		Name:   ap.ProbeName,
		Seq:    ap.ProbeSeq,
		Found:  ap.ProbeFound,
		Strand: ap.ProbeStrand,
		Pos:    ap.ProbePos,
		MM:     ap.ProbeMM,
		Site:   ap.ProbeSite,
	}
}

// RenderPretty renders the ASCII alignment block with the probe overlay.
func RenderPretty(ap AnnotatedProduct) string {
	return pretty.RenderAnnotated(ap.Product, probeAnnotation(ap))
}

// RenderPrettyWithOptions allows custom pretty glyph/options when needed.
func RenderPrettyWithOptions(ap AnnotatedProduct, opt pretty.Options) string {
	return pretty.RenderAnnotatedWithOptions(ap.Product, probeAnnotation(ap), opt)
}
