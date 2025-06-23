// internal/output/text.go
package output

import (
	"fmt"
	"io"

	"ipcress-go/internal/engine"
)

// WriteText prints one line per product.
func WriteText(w io.Writer, list []engine.Product) error {
	for _, p := range list {
		_, err := fmt.Fprintf(w,
			"%s\t%s\t%d\t%d\t%d\t%s\n",
			p.SequenceID, p.ExperimentID,
			p.Start, p.End, p.Length, p.Type,
		)
		if err != nil { return err }
	}
	return nil
}
