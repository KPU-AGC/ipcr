// ./internal/nestedoutput/text.go
package nestedoutput

import (
	"fmt"
	"io"
)

func writeRowTSV(w io.Writer, np NestedProduct) error {
	p := np.Product
	_, err := fmt.Fprintf(
		w, "%s\t%s\t%s\t%d\t%d\t%d\t%s\t%s\t%t\t%s\t%s\t%s\t%s\t%s\t%s\n",
		p.SourceFile, p.SequenceID, p.ExperimentID,
		p.Start, p.End, p.Length, p.Type,
		np.InnerPairID, np.InnerFound,
		ise(np.InnerStart), ise(np.InnerEnd), ise(np.InnerLength),
		np.InnerType, isi(np.InnerFwdMM), isi(np.InnerRevMM),
	)
	return err
}

func StreamText(w io.Writer, in <-chan NestedProduct, header bool) error {
	if header {
		if _, err := io.WriteString(w, TSVHeaderNested+"\n"); err != nil { return err }
	}
	for np := range in {
		if err := writeRowTSV(w, np); err != nil { return err }
	}
	return nil
}

func WriteText(w io.Writer, list []NestedProduct, header bool) error {
	if header {
		if _, err := io.WriteString(w, TSVHeaderNested+"\n"); err != nil { return err }
	}
	for _, np := range list {
		if err := writeRowTSV(w, np); err != nil { return err }
	}
	return nil
}

func ise(v int) string {
	if v == 0 {
		return "" // empty when not applicable
	}
	return fmt.Sprintf("%d", v)
}
func isi(v int) string {
	if v == 0 {
		return "" // empty when not applicable
	}
	return fmt.Sprintf("%d", v)
}
