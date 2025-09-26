package probeoutput

import (
	"fmt"
	"io"
)

// StreamFASTA streams FASTA for annotated products (uses Product.Seq).
func StreamFASTA(w io.Writer, in <-chan AnnotatedProduct) error {
	idx := 1
	for ap := range in {
		p := ap.Product
		if p.Seq == "" { continue }
		if _, err := fmt.Fprintf(
			w, ">%s_%d start=%d end=%d len=%d source_file=%s probe=%s found=%t\n%s\n",
			p.ExperimentID, idx, p.Start, p.End, p.Length, p.SourceFile, ap.ProbeName, ap.ProbeFound, p.Seq,
		); err != nil { return err }
		idx++
	}
	return nil
}

func WriteFASTA(w io.Writer, list []AnnotatedProduct) error {
	for i, ap := range list {
		p := ap.Product
		if p.Seq == "" { continue }
		if _, err := fmt.Fprintf(
			w, ">%s_%d start=%d end=%d len=%d source_file=%s probe=%s found=%t\n%s\n",
			p.ExperimentID, i+1, p.Start, p.End, p.Length, p.SourceFile, ap.ProbeName, ap.ProbeFound, p.Seq,
		); err != nil { return err }
	}
	return nil
}
