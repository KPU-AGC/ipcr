// internal/writers/product.go
package writers

import (
	"io"
	"ipcr-core/engine"
	"ipcr/internal/common"
	"ipcr/internal/output"
	"ipcr/internal/pretty"
)

type productArgs struct {
	Sort   bool
	Header bool
	Pretty bool
	Opt    pretty.Options
	In     <-chan engine.Product
}

func drainProducts(ch <-chan engine.Product) []engine.Product {
	list := make([]engine.Product, 0, 128)
	for p := range ch {
		list = append(list, p)
	}
	return list
}

func init() {
	// JSON array
	RegisterProduct(output.FormatJSON, func(w io.Writer, payload interface{}) error {
		args := payload.(productArgs)
		list := drainProducts(args.In)
		if args.Sort {
			common.SortProducts(list)
		}
		return output.WriteJSON(w, list)
	})

	// JSONL streaming
	RegisterProduct(output.FormatJSONL, func(w io.Writer, payload interface{}) error {
		args := payload.(productArgs)
		pipe, done := StartProductJSONLWriter(w, 64)
		for p := range args.In {
			pipe <- p
		}
		close(pipe)
		return <-done
	})

	// FASTA (stream or buffered+sort)
	RegisterProduct(output.FormatFASTA, func(w io.Writer, payload interface{}) error {
		args := payload.(productArgs)
		if args.Sort {
			list := drainProducts(args.In)
			common.SortProducts(list)
			return output.WriteFASTA(w, list)
		}
		return output.StreamFASTA(w, args.In)
	})

	// TEXT/TSV (+ optional pretty blocks)
	RegisterProduct(output.FormatText, func(w io.Writer, payload interface{}) error {
		args := payload.(productArgs)
		render := func(p engine.Product) string { return pretty.RenderProductWithOptions(p, args.Opt) }
		if args.Sort {
			list := drainProducts(args.In)
			common.SortProducts(list)
			return output.WriteTextWithRenderer(w, list, args.Header, args.Pretty, render)
		}
		return output.StreamTextWithRenderer(w, args.In, args.Header, args.Pretty, render)
	})
}

// Public API (unchanged)
func StartProductWriter(out io.Writer, format string, sort, header, prettyMode bool, bufSize int) (chan<- engine.Product, <-chan error) {
	return StartProductWriterWithPrettyOptions(out, format, sort, header, prettyMode, pretty.DefaultOptions, bufSize)
}

func StartProductWriterWithPrettyOptions(out io.Writer, format string, sort, header, prettyMode bool, popt pretty.Options, bufSize int) (chan<- engine.Product, <-chan error) {
	if bufSize <= 0 {
		bufSize = 64
	}
	in := make(chan engine.Product, bufSize)
	errCh := make(chan error, 1)
	go func() {
		err := WriteProduct(format, out, productArgs{
			Sort:   sort,
			Header: header,
			Pretty: prettyMode,
			Opt:    popt,
			In:     in,
		})
		errCh <- err
	}()
	return in, errCh
}
