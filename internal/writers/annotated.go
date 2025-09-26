package writers

import (
	"fmt"
	"io"

	"ipcr/internal/common"
	"ipcr/internal/pretty"
	"ipcr/internal/probeoutput"
)

// Backward-compatible wrapper
func StartAnnotatedWriter(out io.Writer, format string, sort bool, header bool, prettyMode bool, bufSize int) (chan<- probeoutput.AnnotatedProduct, <-chan error) {
	return StartAnnotatedWriterWithPrettyOptions(out, format, sort, header, prettyMode, pretty.DefaultOptions, bufSize)
}

// Options-aware variant
func StartAnnotatedWriterWithPrettyOptions(out io.Writer, format string, sort bool, header bool, prettyMode bool, popt pretty.Options, bufSize int) (chan<- probeoutput.AnnotatedProduct, <-chan error) {
	if bufSize <= 0 { bufSize = 64 }
	in := make(chan probeoutput.AnnotatedProduct, bufSize)
	errCh := make(chan error, 1)

	go func() {
		var err error
		switch format {
		case "json":
			var buf []probeoutput.AnnotatedProduct
			for ap := range in { buf = append(buf, ap) }
			if sort { common.SortAnnotated(buf) }
			err = probeoutput.WriteJSON(out, buf)

		case "fasta":
			if sort {
				var buf []probeoutput.AnnotatedProduct
				for ap := range in { buf = append(buf, ap) }
				common.SortAnnotated(buf)
				err = probeoutput.WriteFASTA(out, buf)
			} else {
				err = probeoutput.StreamFASTA(out, in)
			}

		case "text":
			if prettyMode {
				// Header once
				if header {
					if _, e := io.WriteString(out, probeoutput.TSVHeaderProbe+"\n"); e != nil && err == nil { err = e }
				}
				if sort {
					var buf []probeoutput.AnnotatedProduct
					for ap := range in { buf = append(buf, ap) }
					common.SortAnnotated(buf)
					for _, ap := range buf {
						if e := probeoutput.WriteRowTSV(out, ap); e != nil && err == nil { err = e }
						if _, e := io.WriteString(out, probeoutput.RenderPrettyWithOptions(ap, popt)); e != nil && err == nil { err = e }
					}
				} else {
					for ap := range in {
						if e := probeoutput.WriteRowTSV(out, ap); e != nil && err == nil { err = e }
						if _, e := io.WriteString(out, probeoutput.RenderPrettyWithOptions(ap, popt)); e != nil && err == nil { err = e }
					}
				}
			} else {
				if sort {
					var buf []probeoutput.AnnotatedProduct
					for ap := range in { buf = append(buf, ap) }
					common.SortAnnotated(buf)
					err = probeoutput.WriteText(out, buf, header)
				} else {
					err = probeoutput.StreamText(out, in, header)
				}
			}

		default:
			err = fmt.Errorf("unsupported output %q", format)
		}
		errCh <- err
	}()

	return in, errCh
}
