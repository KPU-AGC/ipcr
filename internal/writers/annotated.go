// internal/writers/annotated.go
package writers

import (
	"io"
	"ipcr/internal/common"
	"ipcr/internal/output"
	"ipcr/internal/pretty"
	"ipcr/internal/probeoutput"
)

type annotatedArgs struct {
	Sort   bool
	Header bool
	Pretty bool
	Opt    pretty.Options
	In     <-chan probeoutput.AnnotatedProduct
}

func drainAnnotated(ch <-chan probeoutput.AnnotatedProduct) []probeoutput.AnnotatedProduct {
	list := make([]probeoutput.AnnotatedProduct, 0, 128)
	for ap := range ch {
		list = append(list, ap)
	}
	return list
}

func init() {
	// JSON array
	RegisterAnnotated(output.FormatJSON, func(w io.Writer, payload interface{}) error {
		args := payload.(annotatedArgs)
		list := drainAnnotated(args.In)
		if args.Sort {
			common.SortAnnotated(list)
		}
		return probeoutput.WriteJSON(w, list)
	})

	// JSONL streaming
	RegisterAnnotated(output.FormatJSONL, func(w io.Writer, payload interface{}) error {
		args := payload.(annotatedArgs)
		pipe, done := StartAnnotatedJSONLWriter(w, 64)
		for ap := range args.In {
			pipe <- ap
		}
		close(pipe)
		return <-done
	})

	// FASTA (stream or buffered+sort)
	RegisterAnnotated(output.FormatFASTA, func(w io.Writer, payload interface{}) error {
		args := payload.(annotatedArgs)
		if args.Sort {
			list := drainAnnotated(args.In)
			common.SortAnnotated(list)
			return probeoutput.WriteFASTA(w, list)
		}
		return probeoutput.StreamFASTA(w, args.In)
	})

	// TEXT/TSV (+ optional pretty blocks) — now via renderer-capable helpers
	RegisterAnnotated(output.FormatText, func(w io.Writer, payload interface{}) error {
		args := payload.(annotatedArgs)
		render := func(ap probeoutput.AnnotatedProduct) string {
			if !args.Pretty {
				return ""
			}
			return probeoutput.RenderPrettyWithOptions(ap, args.Opt)
		}

		if args.Sort {
			list := drainAnnotated(args.In)
			common.SortAnnotated(list)
			return probeoutput.WriteTextWithRenderer(w, list, args.Header, render)
		}
		return probeoutput.StreamTextWithRenderer(w, args.In, args.Header, render)
	})
}

// Public API (unchanged)
func StartAnnotatedWriter(out io.Writer, format string, sort, header, prettyMode bool, bufSize int) (chan<- probeoutput.AnnotatedProduct, <-chan error) {
	return StartAnnotatedWriterWithPrettyOptions(out, format, sort, header, prettyMode, pretty.DefaultOptions, bufSize)
}

func StartAnnotatedWriterWithPrettyOptions(out io.Writer, format string, sort, header, prettyMode bool, popt pretty.Options, bufSize int) (chan<- probeoutput.AnnotatedProduct, <-chan error) {
	if bufSize <= 0 {
		bufSize = 64
	}
	in := make(chan probeoutput.AnnotatedProduct, bufSize)
	errCh := make(chan error, 1)
	go func() {
		err := WriteAnnotated(format, out, annotatedArgs{
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
