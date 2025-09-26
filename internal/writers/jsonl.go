package writers

import (
	"bufio"
	"encoding/json"
	"io"

	"ipcr/internal/engine"
	"ipcr/internal/probeoutput"
)

// StartProductJSONLWriter streams each engine.Product as one JSON line.
func StartProductJSONLWriter(out io.Writer, bufSize int) (chan<- engine.Product, <-chan error) {
	if bufSize <= 0 { bufSize = 64 }
	in := make(chan engine.Product, bufSize)
	errCh := make(chan error, 1)
	w := bufio.NewWriterSize(out, 64<<10)

	go func() {
		enc := json.NewEncoder(w)
		for p := range in {
			if err := enc.Encode(p); err != nil { errCh <- err; return }
		}
		if err := w.Flush(); err != nil && !IsBrokenPipe(err) { errCh <- err; return }
		errCh <- nil
	}()
	return in, errCh
}

// StartAnnotatedJSONLWriter streams each AnnotatedProduct as one JSON line.
func StartAnnotatedJSONLWriter(out io.Writer, bufSize int) (chan<- probeoutput.AnnotatedProduct, <-chan error) {
	if bufSize <= 0 { bufSize = 64 }
	in := make(chan probeoutput.AnnotatedProduct, bufSize)
	errCh := make(chan error, 1)
	w := bufio.NewWriterSize(out, 64<<10)

	go func() {
		enc := json.NewEncoder(w)
		for ap := range in {
			if err := enc.Encode(ap); err != nil { errCh <- err; return }
		}
		if err := w.Flush(); err != nil && !IsBrokenPipe(err) { errCh <- err; return }
		errCh <- nil
	}()
	return in, errCh
}
