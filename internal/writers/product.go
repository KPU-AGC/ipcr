package writers

import (
	"io"
	"ipcr-core/engine"
	"ipcr/internal/common"
	"ipcr/internal/output"
	"ipcr/internal/pretty"
)

type productArgs struct {
	Sort        bool
	Header      bool
	Pretty      bool
	Opt         pretty.Options
	Scores      bool // NEW: include 'score' in TSV
	RankByScore bool // NEW: prefer score sort over coord
	In          <-chan engine.Product
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
		// Sorting for JSON (if requested)
		if args.Sort {
			if args.RankByScore {
				common.SortProductsByScore(list)
			} else {
				common.SortProducts(list)
			}
		}
		return output.WriteJSON(w, list)
	})

	// JSONL streaming
	RegisterProduct(output.FormatJSONL, func(w io.Writer, payload interface{}) error {
		args := payload.(productArgs)
		pipe, done := StartProductJSONLWriter(w, 64)
		// Sorting not applicable in streaming; honor input order.
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
			if args.RankByScore {
				common.SortProductsByScore(list)
			} else {
				common.SortProducts(list)
			}
			return output.WriteFASTA(w, list)
		}
		return output.StreamFASTA(w, args.In)
	})

	// TEXT/TSV (+ optional pretty blocks + optional 'score' column)
	RegisterProduct(output.FormatText, func(w io.Writer, payload interface{}) error {
		args := payload.(productArgs)

		writeHeader := func() error {
			if !args.Header {
				return nil
			}
			h := output.TSVHeader // canonical base header
			if args.Scores {
				h = h + "\tscore"
			}
			_, err := io.WriteString(w, h+"\n")
			return err
		}

		writeRow := func(p engine.Product) error {
			if args.Scores {
				if _, err := io.WriteString(w, output.FormatRowTSVWithScore(p)+"\n"); err != nil {
					return err
				}
			} else {
				if _, err := io.WriteString(w, output.FormatBaseRowTSV(p)+"\n"); err != nil {
					return err
				}
			}
			if args.Pretty {
				if _, err := io.WriteString(w, pretty.RenderProductWithOptions(p, args.Opt)); err != nil {
					return err
				}
			}
			return nil
		}

		if args.Sort {
			// buffer, sort, then emit header + rows
			list := drainProducts(args.In)
			if args.RankByScore {
				common.SortProductsByScore(list)
			} else {
				common.SortProducts(list)
			}
			if err := writeHeader(); err != nil {
				return err
			}
			for _, p := range list {
				if err := writeRow(p); err != nil {
					return err
				}
			}
			return nil
		}

		// streaming mode (no sort): write header, then rows as they arrive
		if err := writeHeader(); err != nil {
			return err
		}
		for p := range args.In {
			if err := writeRow(p); err != nil {
				return err
			}
		}
		return nil
	})
}

// Public API (updated to carry scores & rank-by-score)
func StartProductWriter(out io.Writer, format string, sort, header, prettyMode bool, includeScore bool, rankByScore bool, bufSize int) (chan<- engine.Product, <-chan error) {
	return StartProductWriterWithPrettyOptions(out, format, sort, header, prettyMode, includeScore, rankByScore, pretty.DefaultOptions, bufSize)
}

func StartProductWriterWithPrettyOptions(out io.Writer, format string, sort, header, prettyMode bool, includeScore bool, rankByScore bool, popt pretty.Options, bufSize int) (chan<- engine.Product, <-chan error) {
	if bufSize <= 0 {
		bufSize = 64
	}
	in := make(chan engine.Product, bufSize)
	errCh := make(chan error, 1)
	go func() {
		err := WriteProduct(format, out, productArgs{
			Sort:        sort,
			Header:      header,
			Pretty:      prettyMode,
			Opt:         popt,
			Scores:      includeScore,
			RankByScore: rankByScore,
			In:          in,
		})
		errCh <- err
	}()
	return in, errCh
}
