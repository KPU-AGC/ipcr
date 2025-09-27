// internal/probeoutput/text.go  (REPLACE)
package probeoutput

import (
	"fmt"
	"io"
	"ipcr/internal/output"
	"strconv"
)

func WriteRowTSV(w io.Writer, ap AnnotatedProduct) error {
	base := output.FormatBaseRowTSV(ap.Product)

	pos := ""
	mm := ""
	if ap.ProbeFound {
		pos = strconv.Itoa(ap.ProbePos)
		mm = strconv.Itoa(ap.ProbeMM)
	}

	_, err := fmt.Fprintf(
		w, "%s\t%s\t%s\t%t\t%s\t%s\t%s\t%s\n",
		base,
		ap.ProbeName, ap.ProbeSeq, ap.ProbeFound, ap.ProbeStrand,
		pos, mm, ap.ProbeSite,
	)
	return err
}

func StreamText(w io.Writer, in <-chan AnnotatedProduct, header bool) error {
	if header {
		if _, err := fmt.Fprintln(w, TSVHeaderProbe); err != nil {
			return err
		}
	}
	for ap := range in {
		if err := WriteRowTSV(w, ap); err != nil {
			return err
		}
	}
	return nil
}

func WriteText(w io.Writer, list []AnnotatedProduct, header bool) error {
	if header {
		if _, err := fmt.Fprintln(w, TSVHeaderProbe); err != nil {
			return err
		}
	}
	for _, ap := range list {
		if err := WriteRowTSV(w, ap); err != nil {
			return err
		}
	}
	return nil
}
