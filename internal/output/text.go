// internal/output/text.go
package output

import (
	"fmt"
	"io"

	"ipcr/internal/engine"
)

// StreamText writes products as they arrive.
func StreamText(w io.Writer, in <-chan engine.Product) error {
	for p := range in {
		if _, err := fmt.Fprintf(w,
			"%s\t%s\t%d\t%d\t%d\t%s\n",
			p.SequenceID, p.ExperimentID, p.Start, p.End, p.Length, p.Type); err != nil {
			return err
		}
	}
	return nil
}
// ===