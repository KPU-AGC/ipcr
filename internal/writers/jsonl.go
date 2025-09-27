// internal/writers/jsonl.go
package writers

import (
	"bufio"
	"encoding/json"
	"io"
	"ipcr-core/engine"
	"ipcr/internal/output"
	"ipcr/internal/probeoutput"
	"ipcr/pkg/api"
)

// StartProductJSONLWriter streams each engine.Product as one JSON line (v1).
func StartProductJSONLWriter(out io.Writer, bufSize int) (chan<- engine.Product, <-chan error) {
	if bufSize <= 0 {
		bufSize = 64
	}
	in := make(chan engine.Product, bufSize)
	errCh := make(chan error, 1)
	w := bufio.NewWriterSize(out, 64<<10)

	go func() {
		enc := json.NewEncoder(w)
		for p := range in {
			var v1 api.ProductV1 = output.ToAPIProduct(p)
			if err := enc.Encode(v1); err != nil {
				errCh <- err
				return
			}
		}
		if err := w.Flush(); err != nil && !IsBrokenPipe(err) {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	return in, errCh
}

// StartAnnotatedJSONLWriter streams each AnnotatedProduct as one JSON line (v1).
func StartAnnotatedJSONLWriter(out io.Writer, bufSize int) (chan<- probeoutput.AnnotatedProduct, <-chan error) {
	if bufSize <= 0 {
		bufSize = 64
	}
	in := make(chan probeoutput.AnnotatedProduct, bufSize)
	errCh := make(chan error, 1)
	w := bufio.NewWriterSize(out, 64<<10)

	go func() {
		enc := json.NewEncoder(w)
		for ap := range in {
			v1 := probeoutput.ToAPIAnnotated(ap)
			if err := enc.Encode(v1); err != nil {
				errCh <- err
				return
			}
		}
		if err := w.Flush(); err != nil && !IsBrokenPipe(err) {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	return in, errCh
}
