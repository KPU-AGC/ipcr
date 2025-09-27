package output

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"ipcr/internal/engine"
)

func intsCSV(a []int) string {
	if len(a) == 0 {
		return ""
	}
	ss := make([]string, len(a))
	for i, v := range a {
		ss[i] = strconv.Itoa(v)
	}
	return strings.Join(ss, ",")
}

func writeRowTSV(w io.Writer, p engine.Product) error {
	_, err := fmt.Fprintf(
		w, "%s\t%s\t%s\t%d\t%d\t%d\t%s\t%d\t%d\t%s\t%s\n",
		p.SourceFile, p.SequenceID, p.ExperimentID,
		p.Start, p.End, p.Length, p.Type,
		p.FwdMM, p.RevMM,
		intsCSV(p.FwdMismatchIdx), intsCSV(p.RevMismatchIdx),
	)
	return err
}

// New: renderer-capable streaming writer for text mode
func StreamTextWithRenderer(w io.Writer, in <-chan engine.Product, header bool, prettyMode bool, render func(engine.Product) string) error {
	if header {
		if _, err := fmt.Fprintln(w, TSVHeader); err != nil {
			return err
		}
	}
	for p := range in {
		if err := writeRowTSV(w, p); err != nil {
			return err
		}
		if prettyMode {
			if _, err := io.WriteString(w, render(p)); err != nil {
				return err
			}
		}
	}
	return nil
}

// New: renderer-capable buffered writer for text mode
func WriteTextWithRenderer(w io.Writer, list []engine.Product, header bool, prettyMode bool, render func(engine.Product) string) error {
	if header {
		if _, err := fmt.Fprintln(w, TSVHeader); err != nil {
			return err
		}
	}
	for _, p := range list {
		if err := writeRowTSV(w, p); err != nil {
			return err
		}
		if prettyMode {
			if _, err := io.WriteString(w, render(p)); err != nil {
				return err
			}
		}
	}
	return nil
}

// Backward-compat wrappers (use default renderer wired in output package)
func StreamText(w io.Writer, in <-chan engine.Product, header bool, prettyMode bool) error {
	return StreamTextWithRenderer(w, in, header, prettyMode, func(p engine.Product) string { return "" })
}

func WriteText(w io.Writer, list []engine.Product, header bool, prettyMode bool) error {
	return WriteTextWithRenderer(w, list, header, prettyMode, func(p engine.Product) string { return "" })
}
