package probeoutput

import (
	"fmt"
	"io"
	"ipcr/internal/output"
)

// StreamFASTA streams FASTA for annotated products (uses Product.Seq).
func StreamFASTA(w io.Writer, in <-chan AnnotatedProduct) error {
	return StreamFASTAWithIDs(w, in, output.FastaIDsExperiment)
}

// StreamFASTAWithIDs streams FASTA for annotated products using the given
// --fasta-ids mode.
func StreamFASTAWithIDs(w io.Writer, in <-chan AnnotatedProduct, mode string) error {
	namer := output.NewFASTANamer(mode)
	idx := 1
	for ap := range in {
		p := ap.Product
		if p.Seq == "" {
			continue
		}
		if _, err := fmt.Fprintf(
			w, ">%s probe=%s found=%t\n%s\n",
			namer.Header(p, idx), ap.ProbeName, ap.ProbeFound, p.Seq,
		); err != nil {
			return err
		}
		idx++
	}
	return nil
}

func WriteFASTA(w io.Writer, list []AnnotatedProduct) error {
	return WriteFASTAWithIDs(w, list, output.FastaIDsExperiment)
}

// WriteFASTAWithIDs writes annotated products as FASTA using the given
// --fasta-ids mode.
func WriteFASTAWithIDs(w io.Writer, list []AnnotatedProduct, mode string) error {
	namer := output.NewFASTANamer(mode)
	for i, ap := range list {
		p := ap.Product
		if p.Seq == "" {
			continue
		}
		if _, err := fmt.Fprintf(
			w, ">%s probe=%s found=%t\n%s\n",
			namer.Header(p, i+1), ap.ProbeName, ap.ProbeFound, p.Seq,
		); err != nil {
			return err
		}
	}
	return nil
}
