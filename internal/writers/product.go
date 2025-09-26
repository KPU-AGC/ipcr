package writers

import (
	"fmt"
	"io"

	"ipcr/internal/common"
	"ipcr/internal/engine"
	"ipcr/internal/output"
	"ipcr/internal/pretty"
)

// StartProductWriter spins up a writer goroutine for plain engine.Product items.
// (Backward-compatible wrapper using pretty.DefaultOptions)
func StartProductWriter(out io.Writer, format string, sort bool, header bool, prettyMode bool, bufSize int) (chan<- engine.Product, <-chan error) {
	return StartProductWriterWithPrettyOptions(out, format, sort, header, prettyMode, pretty.DefaultOptions, bufSize)
}

// StartProductWriterWithPrettyOptions allows customizing the pretty renderer.
func StartProductWriterWithPrettyOptions(out io.Writer, format string, sort bool, header bool, prettyMode bool, popt pretty.Options, bufSize int) (chan<- engine.Product, <-chan error) {
	if bufSize <= 0 {
		bufSize = 64
	}
	in := make(chan engine.Product, bufSize)
	errCh := make(chan error, 1)

	go func() {
		var err error
		switch format {
		case "json":
			var buf []engine.Product
			for p := range in {
				buf = append(buf, p)
			}
			if sort {
				common.SortProducts(buf)
			}
			err = output.WriteJSON(out, buf)

		case "fasta":
			if sort {
				var buf []engine.Product
				for p := range in {
					buf = append(buf, p)
				}
				common.SortProducts(buf)
				err = output.WriteFASTA(out, buf)
			} else {
				err = output.StreamFASTA(out, in)
			}

		case "text":
			if sort {
				var buf []engine.Product
				for p := range in {
					buf = append(buf, p)
				}
				common.SortProducts(buf)
				// write TSV (and pretty) in a buffered manner
				err = output.WriteTextWithRenderer(out, buf, header, prettyMode,
					func(p engine.Product) string { return pretty.RenderProductWithOptions(p, popt) },
				)
			} else {
				// streaming
				err = output.StreamTextWithRenderer(out, in, header, prettyMode,
					func(p engine.Product) string { return pretty.RenderProductWithOptions(p, popt) },
				)
			}

		default:
			err = fmt.Errorf("unsupported output %q", format)
		}
		errCh <- err
	}()

	return in, errCh
}
