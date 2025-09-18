// internal/output/text.go
package output

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"ipcr/internal/engine"
)

const textHeader = "sequence_id\texperiment_id\tstart\tend\tlength\ttype\tfwd_mm\trev_mm\tfwd_mismatch_idx\trev_mismatch_idx"

// Arbitrary interior gap ('.' count between left/right primer pictures in pretty diagram).
// Tweak this to change spacing; it does not depend on amplicon size.
const diagramInner = 24

// intsCSV converts []int to "1,2,3" (empty string if none)
func intsCSV(a []int) string {
	if len(a) == 0 {
		return ""
	}
	var b strings.Builder
	for i, v := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(v))
	}
	return b.String()
}

// matchLine draws '|' for matches and ' ' (space) for mismatches at given indices.
func matchLine(n int, mismIdx []int) string {
	if n <= 0 {
		return ""
	}
	m := make(map[int]struct{}, len(mismIdx))
	for _, i := range mismIdx {
		m[i] = struct{}{}
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		if _, bad := m[i]; bad {
			b.WriteByte(' ')
		} else {
			b.WriteByte('|')
		}
	}
	return b.String()
}

// diagramPretty prints a two‑strand block with left/right primers and arrows.
func diagramPretty(w io.Writer, p engine.Product) {
	// Need primer sequences and their target sites to render nicely.
	// (These are populated by the engine during simulation.)
	if p.FwdPrimer == "" || p.RevPrimer == "" || p.FwdSite == "" || p.RevSite == "" {
		return
	}

	const arrowRight = "-->"
	const arrowLeft = "<--"

	aLen := len(p.FwdPrimer)
	bLen := len(p.RevPrimer)
	inner := diagramInner

	fmt.Fprintf(w, "   %s\n", p.FwdPrimer)
	fmt.Fprintf(w, "   %s%s\n", matchLine(aLen, p.FwdMismatchIdx), arrowRight)

	fmt.Fprintf(w, "5'-%s%s-3' # (+)\n", p.FwdSite, strings.Repeat(".", inner))
	fmt.Fprintf(w, "3'-%s%s-5' # (-)\n", strings.Repeat(".", inner), p.RevSite)

	siteStart := 3 + inner

	padBars := siteStart - len(arrowLeft)
	if padBars < 0 {
		padBars = 0
	}
	fmt.Fprintf(w, "%s%s%s\n", strings.Repeat(" ", padBars), arrowLeft, matchLine(bLen, p.RevMismatchIdx))

	fmt.Fprintf(w, "%s%s\n\n", strings.Repeat(" ", siteStart), p.RevPrimer)
}

func writeOne(w io.Writer, p engine.Product, pretty bool) error {
	// Primary tabular line
	if _, err := fmt.Fprintf(
		w, "%s\t%s\t%d\t%d\t%d\t%s\t%d\t%d\t%s\t%s\n",
		p.SequenceID, p.ExperimentID, p.Start, p.End, p.Length, p.Type,
		p.FwdMM, p.RevMM, intsCSV(p.FwdMismatchIdx), intsCSV(p.RevMismatchIdx),
	); err != nil {
		return err
	}
	if !pretty {
		return nil
	}
	diagramPretty(w, p)
	return nil
}

func StreamText(w io.Writer, in <-chan engine.Product, header bool, pretty bool) error {
	wroteHeader := false
	for p := range in {
		if header && !wroteHeader {
			if _, err := fmt.Fprintln(w, textHeader); err != nil {
				return err
			}
			wroteHeader = true
		}
		if err := writeOne(w, p, pretty); err != nil {
			return err
		}
	}
	return nil
}

// WriteText writes a slice of products (used for sorted output).
func WriteText(w io.Writer, list []engine.Product, header bool, pretty bool) error {
	if header {
		if _, err := fmt.Fprintln(w, textHeader); err != nil {
			return err
		}
	}
	for _, p := range list {
		if err := writeOne(w, p, pretty); err != nil {
			return err
		}
	}
	return nil
}
