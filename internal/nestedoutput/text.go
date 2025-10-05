// ./internal/nestedoutput/text.go
package nestedoutput

import (
	"fmt"
	"io"
	"strconv"
)

func writeRowTSV(w io.Writer, np NestedProduct) error {
	p := np.Product

	// Only hide inner fields when InnerFound=false; otherwise print numeric 0s.
	innerStart := emptyIf(!np.InnerFound, strconv.Itoa(np.InnerStart))
	innerEnd := emptyIf(!np.InnerFound, strconv.Itoa(np.InnerEnd))
	innerLen := emptyIf(!np.InnerFound, strconv.Itoa(np.InnerLength))
	innerType := emptyIf(!np.InnerFound, np.InnerType)
	innerFwdMM := emptyIf(!np.InnerFound, strconv.Itoa(np.InnerFwdMM))
	innerRevMM := emptyIf(!np.InnerFound, strconv.Itoa(np.InnerRevMM))

	_, err := fmt.Fprintf(
		w, "%s\t%s\t%s\t%d\t%d\t%d\t%s\t%s\t%t\t%s\t%s\t%s\t%s\t%s\t%s\n",
		p.SourceFile, p.SequenceID, p.ExperimentID,
		p.Start, p.End, p.Length, p.Type,
		np.InnerPairID, np.InnerFound,
		innerStart, innerEnd, innerLen,
		innerType, innerFwdMM, innerRevMM,
	)
	return err
}

// New: renderer-capable streaming writer (keeps parity with products/probe)
func StreamTextWithRenderer(w io.Writer, in <-chan NestedProduct, header bool, render func(NestedProduct) string) error {
	if header {
		if _, err := io.WriteString(w, TSVHeaderNested+"\n"); err != nil {
			return err
		}
	}
	for np := range in {
		if err := writeRowTSV(w, np); err != nil {
			return err
		}
		if render != nil {
			if _, err := io.WriteString(w, render(np)); err != nil {
				return err
			}
		}
	}
	return nil
}

// New: renderer-capable buffered writer
func WriteTextWithRenderer(w io.Writer, list []NestedProduct, header bool, render func(NestedProduct) string) error {
	if header {
		if _, err := io.WriteString(w, TSVHeaderNested+"\n"); err != nil {
			return err
		}
	}
	for _, np := range list {
		if err := writeRowTSV(w, np); err != nil {
			return err
		}
		if render != nil {
			if _, err := io.WriteString(w, render(np)); err != nil {
				return err
			}
		}
	}
	return nil
}

// Back-compat wrappers
func StreamText(w io.Writer, in <-chan NestedProduct, header bool) error {
	return StreamTextWithRenderer(w, in, header, nil)
}
func WriteText(w io.Writer, list []NestedProduct, header bool) error {
	return WriteTextWithRenderer(w, list, header, nil)
}

func emptyIf(cond bool, s string) string {
	if cond {
		return ""
	}
	return s
}
