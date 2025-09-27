package probeoutput

import (
	"ipcr/internal/pretty"
)

func RenderPretty(ap AnnotatedProduct) string {
	ann := pretty.ProbeAnnotation{
		Name:   ap.ProbeName,
		Seq:    ap.ProbeSeq,
		Found:  ap.ProbeFound,
		Strand: ap.ProbeStrand,
		Pos:    ap.ProbePos,
		MM:     ap.ProbeMM,
		Site:   ap.ProbeSite,
	}
	return pretty.RenderAnnotated(ap.Product, ann)
}

func RenderPrettyWithOptions(ap AnnotatedProduct, opt pretty.Options) string {
	ann := pretty.ProbeAnnotation{
		Name:   ap.ProbeName,
		Seq:    ap.ProbeSeq,
		Found:  ap.ProbeFound,
		Strand: ap.ProbeStrand,
		Pos:    ap.ProbePos,
		MM:     ap.ProbeMM,
		Site:   ap.ProbeSite,
	}
	return pretty.RenderAnnotatedWithOptions(ap.Product, ann, opt)
}
