// internal/output/fasta.go
package output

import (
	"fmt"
	"io"
	"ipcr-core/engine"
)

// FASTA record-ID styles selectable with --fasta-ids.
const (
	// FastaIDsExperiment is the historical style: ">{experiment}_{N}", where N
	// is a run-wide counter (e.g. ">manual_1").
	FastaIDsExperiment = "experiment"
	// FastaIDsSource names each amplicon after the input FASTA record it came
	// from: ">{sequence_id}_amp{N}", where N restarts at 1 for every source
	// sequence (e.g. ">NC_000913.3_amp2").
	FastaIDsSource = "source"
)

// ValidFastaIDs reports whether mode is a recognised --fasta-ids value.
func ValidFastaIDs(mode string) bool {
	return mode == FastaIDsExperiment || mode == FastaIDsSource
}

// FASTANamer assigns record IDs and the shared leading header fields for
// FASTA output. It is not safe for concurrent use; writers consume products
// from a single goroutine.
type FASTANamer struct {
	mode   string
	perSeq map[string]int
}

// NewFASTANamer returns a namer for the given --fasta-ids mode. An empty or
// unknown mode falls back to FastaIDsExperiment.
func NewFASTANamer(mode string) *FASTANamer {
	if !ValidFastaIDs(mode) {
		mode = FastaIDsExperiment
	}
	return &FASTANamer{mode: mode, perSeq: make(map[string]int)}
}

// Header returns the FASTA header line (without the leading '>' or trailing
// newline) for p. runIdx is the run-wide 1-based record number used by the
// experiment style. Call it once per emitted record, in output order.
func (n *FASTANamer) Header(p engine.Product, runIdx int) string {
	if n.mode == FastaIDsSource {
		// Count per sequence ID (not per file) so record IDs stay unique
		// within one output even if two inputs reuse a sequence ID.
		n.perSeq[p.SequenceID]++
		return fmt.Sprintf(
			"%s_amp%d experiment=%s sequence_id=%s start=%d end=%d len=%d type=%s source_file=%s",
			p.SequenceID, n.perSeq[p.SequenceID], p.ExperimentID, p.SequenceID,
			p.Start, p.End, p.Length, p.Type, p.SourceFile,
		)
	}
	return fmt.Sprintf(
		"%s_%d start=%d end=%d len=%d source_file=%s",
		p.ExperimentID, runIdx, p.Start, p.End, p.Length, p.SourceFile,
	)
}

// StreamFASTA streams FASTA records from a channel to the writer.
func StreamFASTA(w io.Writer, in <-chan engine.Product) error {
	return StreamFASTAWithIDs(w, in, FastaIDsExperiment)
}

// StreamFASTAWithIDs streams FASTA records using the given --fasta-ids mode.
func StreamFASTAWithIDs(w io.Writer, in <-chan engine.Product, mode string) error {
	namer := NewFASTANamer(mode)
	idx := 1
	for p := range in {
		if p.Seq == "" {
			continue
		}
		if _, err := fmt.Fprintf(w, ">%s\n%s\n", namer.Header(p, idx), p.Seq); err != nil {
			return err
		}
		idx++
	}
	return nil
}

// WriteFASTA writes a slice of products as FASTA records to the writer.
func WriteFASTA(w io.Writer, list []engine.Product) error {
	return WriteFASTAWithIDs(w, list, FastaIDsExperiment)
}

// WriteFASTAWithIDs writes a slice of products as FASTA records using the
// given --fasta-ids mode.
func WriteFASTAWithIDs(w io.Writer, list []engine.Product, mode string) error {
	namer := NewFASTANamer(mode)
	for i, p := range list {
		if p.Seq == "" {
			continue
		}
		if _, err := fmt.Fprintf(w, ">%s\n%s\n", namer.Header(p, i+1), p.Seq); err != nil {
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
		if _, err := fmt.Fprintln(w, FormatBaseRowTSV(p)); err != nil {
			return err
		}
	}
	return nil
}
