// internal/writers/nested.go
package writers

import (
	"io"
	"ipcr/internal/common"
	"ipcr/internal/nestedoutput"
	"ipcr/internal/output"
	"sort"
)

type nestedArgs struct {
	Sort   bool
	Header bool
	In     <-chan nestedoutput.NestedProduct
}

func drainNested(ch <-chan nestedoutput.NestedProduct) []nestedoutput.NestedProduct {
	list := make([]nestedoutput.NestedProduct, 0, 128)
	for np := range ch {
		list = append(list, np)
	}
	return list
}

func init() {
	// JSON array
	RegisterNested(output.FormatJSON, func(w io.Writer, payload interface{}) error {
		args := payload.(nestedArgs)
		list := drainNested(args.In)
		if args.Sort {
			sort.Slice(list, func(i, j int) bool { return common.LessProduct(list[i].Product, list[j].Product) })
		}
		return nestedoutput.WriteJSON(w, list)
	})

	// JSONL streaming (unified via StartNestedJSONLWriter)
	RegisterNested(output.FormatJSONL, func(w io.Writer, payload interface{}) error {
		args := payload.(nestedArgs)
		pipe, done := StartNestedJSONLWriter(w, 64)
		for np := range args.In {
			pipe <- np
		}
		close(pipe)
		return <-done
	})

	// TEXT/TSV (parity stub: renderer kept nil for now)
	RegisterNested(output.FormatText, func(w io.Writer, payload interface{}) error {
		args := payload.(nestedArgs)
		if args.Sort {
			list := drainNested(args.In)
			sort.Slice(list, func(i, j int) bool { return common.LessProduct(list[i].Product, list[j].Product) })
			return nestedoutput.WriteTextWithRenderer(w, list, args.Header, nil)
		}
		return nestedoutput.StreamTextWithRenderer(w, args.In, args.Header, nil)
	})
}

// Public API (unchanged)
func StartNestedWriter(out io.Writer, format string, sortOut, header bool, bufSize int) (chan<- nestedoutput.NestedProduct, <-chan error) {
	if bufSize <= 0 {
		bufSize = 64
	}
	in := make(chan nestedoutput.NestedProduct, bufSize)
	errCh := make(chan error, 1)
	go func() {
		err := WriteNested(format, out, nestedArgs{
			Sort:   sortOut,
			Header: header,
			In:     in,
		})
		errCh <- err
	}()
	return in, errCh
}
