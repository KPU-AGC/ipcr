package probeoutput

import (
	"encoding/json"
	"io"
)

func WriteJSON(w io.Writer, list []AnnotatedProduct) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(list)
}
