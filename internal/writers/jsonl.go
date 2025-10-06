// internal/writers/jsonl.go
package writers

import (
	"encoding/json"
	"io"
	"ipcr-core/engine"
	"ipcr/internal/jsonlutil"
	"ipcr/internal/output"
	"ipcr/internal/probeoutput"
)

// StartProductJSONLWriter streams each engine.Product as one JSON line (v1).
func StartProductJSONLWriter(out io.Writer, bufSize int) (chan<- engine.Product, <-chan error) {
	return jsonlutil.Start[engine.Product](out, bufSize,
		func(enc *json.Encoder, p engine.Product) error {
			return enc.Encode(output.ToAPIProduct(p))
		},
		IsBrokenPipe,
	)
}

// StartAnnotatedJSONLWriter streams each AnnotatedProduct as one JSON line (v1).
func StartAnnotatedJSONLWriter(out io.Writer, bufSize int) (chan<- probeoutput.AnnotatedProduct, <-chan error) {
	return jsonlutil.Start[probeoutput.AnnotatedProduct](out, bufSize,
		func(enc *json.Encoder, ap probeoutput.AnnotatedProduct) error {
			return enc.Encode(probeoutput.ToAPIAnnotated(ap))
		},
		IsBrokenPipe,
	)
}
