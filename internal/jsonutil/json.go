// internal/jsonutil/json.go
package jsonutil

import (
	"encoding/json"
	"io"
)

// EncodePretty writes v as indented JSON to w.
func EncodePretty(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
