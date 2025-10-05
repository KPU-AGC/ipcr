// internal/jsonlutil/jsonlutil.go
package jsonlutil

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
)

// Reuse a 64 KiB buffered writer across JSONL writers to avoid per-writer mallocs.
// Encoder itself is tiny and tied to an io.Writer, so we (re)create it per goroutine.
var bwPool = sync.Pool{
	New: func() any {
		return bufio.NewWriterSize(io.Discard, 64<<10)
	},
}

// Start spins up a JSONL encoder goroutine for values of type T.
//   - encode: fn to encode one value (convert to wire type & enc.Encode)
//   - isBroken: recognizer for broken/closed pipe errors to suppress them
func Start[T any](out io.Writer, bufSize int, encode func(*json.Encoder, T) error, isBroken func(error) bool) (chan<- T, <-chan error) {
	if bufSize <= 0 {
		bufSize = 64
	}
	in := make(chan T, bufSize)
	done := make(chan error, 1)

	go func() {
		bw := bwPool.Get().(*bufio.Writer)
		// Rebind to the actual output while keeping the pooled buffer.
		bw.Reset(out)
		// Always put back to pool and drop references to 'out'.
		defer func() {
			bw.Reset(io.Discard)
			bwPool.Put(bw)
		}()

		enc := json.NewEncoder(bw)

		for v := range in {
			if err := encode(enc, v); err != nil {
				done <- err
				return
			}
		}
		if err := bw.Flush(); err != nil && !isBroken(err) {
			done <- err
			return
		}
		done <- nil
	}()

	return in, done
}
