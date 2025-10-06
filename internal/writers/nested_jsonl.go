// internal/writers/nested_jsonl.go
package writers

import (
	"encoding/json"
	"io"
	"ipcr/internal/jsonlutil"
	"ipcr/internal/nestedoutput"
)

// StartNestedJSONLWriter streams NestedProduct as api.NestedProductV1 JSONL.
func StartNestedJSONLWriter(out io.Writer, bufSize int) (chan<- nestedoutput.NestedProduct, <-chan error) {
	return jsonlutil.Start[nestedoutput.NestedProduct](out, bufSize,
		func(enc *json.Encoder, np nestedoutput.NestedProduct) error {
			return enc.Encode(nestedoutput.ToAPINested(np))
		},
		IsBrokenPipe,
	)
}
