package output

import (
	"encoding/json"
	"io"

	"ipcress-go/internal/engine"
)

func WriteJSON(w io.Writer, list []engine.Product) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(list)
}
