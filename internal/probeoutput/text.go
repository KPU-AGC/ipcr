package probeoutput

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func intsCSV(a []int) string {
	if len(a) == 0 { return "" }
	ss := make([]string, len(a))
	for i, v := range a { ss[i] = strconv.Itoa(v) }
	return strings.Join(ss, ",")
}

func writeOne(w io.Writer, ap AnnotatedProduct) error {
	p := ap.Product
	if _, err := fmt.Fprintf(
		w, "%s\t%s\t%s\t%d\t%d\t%d\t%s\t%d\t%d\t%s\t%s\t%s\t%s\t%t\t%s\t%s\t%s\t%s\n",
		p.SourceFile, p.SequenceID, p.ExperimentID,
		p.Start, p.End, p.Length, p.Type,
		p.FwdMM, p.RevMM,
		intsCSV(p.FwdMismatchIdx), intsCSV(p.RevMismatchIdx),
		ap.ProbeName, ap.ProbeSeq, ap.ProbeFound,
		ap.ProbeStrand,
		// empty if not found:
		func() string { if ap.ProbeFound { return strconv.Itoa(ap.ProbePos) } else { return "" } }(),
		func() string { if ap.ProbeFound { return strconv.Itoa(ap.ProbeMM) } else { return "" } }(),
		ap.ProbeSite,
	); err != nil {
		return err
	}
	return nil
}

func StreamText(w io.Writer, in <-chan AnnotatedProduct, header bool) error {
	if header {
		if _, err := fmt.Fprintln(w, TSVHeaderProbe); err != nil { return err }
	}
	for ap := range in {
		if err := writeOne(w, ap); err != nil { return err }
	}
	return nil
}

func WriteText(w io.Writer, list []AnnotatedProduct, header bool) error {
	if header {
		if _, err := fmt.Fprintln(w, TSVHeaderProbe); err != nil { return err }
	}
	for _, ap := range list {
		if err := writeOne(w, ap); err != nil { return err }
	}
	return nil
}
