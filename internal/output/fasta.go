package output

import (
	"fmt"
	"io"

	"ipcr/internal/engine"
)

// StreamFASTA streams FASTA records from a channel to the writer.
func StreamFASTA(w io.Writer, in <-chan engine.Product) error {
	idx := 1
	for p := range in {
		if p.Seq == "" {
			continue
		}
		if _, err := fmt.Fprintf(
			w,
			">%s_%d start=%d end=%d len=%d source_file=%s\n%s\n",
			p.ExperimentID, idx, p.Start, p.End, p.Length, p.SourceFile, p.Seq,
		); err != nil {
			return err
		}
		idx++
	}
	return nil
}

// WriteFASTA writes a slice of products as FASTA records to the writer.
func WriteFASTA(w io.Writer, list []engine.Product) error {
	for i, p := range list {
		if p.Seq == "" {
			continue
		}
		if _, err := fmt.Fprintf(
			w,
			">%s_%d start=%d end=%d len=%d source_file=%s\n%s\n",
			p.ExperimentID, i+1, p.Start, p.End, p.Length, p.SourceFile, p.Seq,
		); err != nil {
			return err
		}
	}
	return nil
}

// WriteTSV writes products as a tab-delimited table (parity with text output).
func WriteTSV(w io.Writer, list []engine.Product, header bool) error {
	if header {
		if _, err := fmt.Fprintln(w, TSVHeader); err != nil {
			return err
		}
	}
	for _, p := range list {
		if _, err := fmt.Fprintf(
			w, "%s\t%s\t%s\t%d\t%d\t%d\t%s\t%d\t%d\t%s\t%s\n",
			p.SourceFile, p.SequenceID, p.ExperimentID,
			p.Start, p.End, p.Length, p.Type,
			p.FwdMM, p.RevMM,
			intsCSV(p.FwdMismatchIdx), intsCSV(p.RevMismatchIdx),
		); err != nil {
			return err
		}
	}
	return nil
}
