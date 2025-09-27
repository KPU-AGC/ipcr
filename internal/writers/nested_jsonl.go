// internal/writers/nested_jsonl.go  (NEW)
package writers

import (
	"bufio"
	"encoding/json"
	"io"

	"ipcr/internal/nestedoutput"
)

// StartNestedJSONLWriter streams NestedProduct as api.NestedProductV1 JSONL.
func StartNestedJSONLWriter(out io.Writer, bufSize int) (chan<- nestedoutput.NestedProduct, <-chan error) {
	if bufSize <= 0 {
		bufSize = 64
	}
	in := make(chan nestedoutput.NestedProduct, bufSize)
	done := make(chan error, 1)
	go func() {
		bw := bufio.NewWriterSize(out, 64<<10)
		enc := json.NewEncoder(bw)
		for np := range in {
			if err := enc.Encode(nestedoutput.ToAPINested(np)); err != nil {
				done <- err
				return
			}
		}
		if err := bw.Flush(); err != nil && !IsBrokenPipe(err) {
			done <- err
			return
		}
		done <- nil
	}()
	return in, done
}
