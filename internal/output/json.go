// internal/output/json.go
package output

import (
	"encoding/json"
	"io"

	"ipcr/internal/engine"
)

func WriteJSON(w io.Writer, list []engine.Product) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(list)
}
// ===