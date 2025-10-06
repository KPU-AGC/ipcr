// internal/writers/nested.go
package writers

import (
	"io"
	"ipcr/internal/common"
	"ipcr/internal/nestedoutput"
	"ipcr/internal/output"
	"ipcr/internal/pretty"
	"sort"
)

type nestedArgs struct {
	Sort   bool
	Header bool
	Pretty bool
	Opt    pretty.Options
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

	// JSONL streaming
	RegisterNested(output.FormatJSONL, func(w io.Writer, payload interface{}) error {
		args := payload.(nestedArgs)
		pipe, done := StartNestedJSONLWriter(w, 64)
		for np := range args.In {
			pipe <- np
		}
		close(pipe)
		return <-done
	})

	// TEXT/TSV (+ optional pretty blocks) — now unified via renderer
	RegisterNested(output.FormatText, func(w io.Writer, payload interface{}) error {
		args := payload.(nestedArgs)
		render := func(np nestedoutput.NestedProduct) string {
			if !args.Pretty {
				return ""
			}
			return nestedoutput.RenderPrettyWithOptions(np, args.Opt)
		}

		if args.Sort {
			list := drainNested(args.In)
			sort.Slice(list, func(i, j int) bool { return common.LessProduct(list[i].Product, list[j].Product) })
			return nestedoutput.WriteTextWithRenderer(w, list, args.Header, render)
		}
		return nestedoutput.StreamTextWithRenderer(w, args.In, args.Header, render)
	})
}

// Back-compat: original signature (pretty=false)
func StartNestedWriter(out io.Writer, format string, sortOut, header bool, bufSize int) (chan<- nestedoutput.NestedProduct, <-chan error) {
	return StartNestedWriterWithPretty(out, format, sortOut, header, false, bufSize)
}

// Convenience: pretty on/off with default glyphs
func StartNestedWriterWithPretty(out io.Writer, format string, sortOut, header, prettyMode bool, bufSize int) (chan<- nestedoutput.NestedProduct, <-chan error) {
	return StartNestedWriterWithPrettyOptions(out, format, sortOut, header, prettyMode, pretty.DefaultOptions, bufSize)
}

// Full control: pretty + options
func StartNestedWriterWithPrettyOptions(out io.Writer, format string, sortOut, header, prettyMode bool, popt pretty.Options, bufSize int) (chan<- nestedoutput.NestedProduct, <-chan error) {
	if bufSize <= 0 {
		bufSize = 64
	}
	in := make(chan nestedoutput.NestedProduct, bufSize)
	errCh := make(chan error, 1)
	go func() {
		err := WriteNested(format, out, nestedArgs{
			Sort:   sortOut,
			Header: header,
			Pretty: prettyMode,
			Opt:    popt,
			In:     in,
		})
		errCh <- err
	}()
	return in, errCh
}
