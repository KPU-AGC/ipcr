// ./internal/nestedoutput/json.go
package nestedoutput

import (
	"io"
	"ipcr/internal/jsonutil"
)

func WriteJSON(w io.Writer, list []NestedProduct) error {
	return jsonutil.EncodePretty(w, ToAPINestedSlice(list))
}
