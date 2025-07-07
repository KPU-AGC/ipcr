// internal/output/fasta.go
package output

import (
	"fmt"
	"io"
	"text/tabwriter"

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
			">%s_%d start=%d end=%d len=%d\n%s\n",
			p.ExperimentID, idx, p.Start, p.End, p.Length, p.Seq,
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
			">%s_%d start=%d end=%d len=%d\n%s\n",
			p.ExperimentID, i+1, p.Start, p.End, p.Length, p.Seq,
		); err != nil {
			return err
		}
	}
	return nil
}

// WriteTSV writes products as a tab-delimited table.
func WriteTSV(w io.Writer, list []engine.Product) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, p := range list {
		fmt.Fprintf(
			tw, "%s\t%s\t%d\t%d\t%d\t%s\n",
			p.SequenceID, p.ExperimentID,
			p.Start, p.End, p.Length, p.Type,
		)
	}
	return tw.Flush()
}
// ===