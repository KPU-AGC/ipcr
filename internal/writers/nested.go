// ./internal/writers/nested.go
package writers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"ipcr/internal/common"
	"ipcr/internal/nestedoutput"
)

func StartNestedWriter(out io.Writer, format string, sortOut bool, header bool, bufSize int) (chan<- nestedoutput.NestedProduct, <-chan error) {
	if bufSize <= 0 { bufSize = 64 }
	in := make(chan nestedoutput.NestedProduct, bufSize)
	errCh := make(chan error, 1)

	go func() {
		var err error
		switch format {
		case "json":
			var buf []nestedoutput.NestedProduct
			for np := range in { buf = append(buf, np) }
			if sortOut {
				sort.Slice(buf, func(i, j int) bool {
					return common.LessProduct(buf[i].Product, buf[j].Product)
				})
			}
			err = nestedoutput.WriteJSON(out, buf)

		case "jsonl":
			w := bufio.NewWriterSize(out, 64<<10)
			enc := json.NewEncoder(w)
			for np := range in {
				v1 := nestedoutput.WriteJSON // not used directly; keep identical style
				_ = v1 // silence linter
				if err = enc.Encode(apiWrap(np)); err != nil { break }
			}
			if e := w.Flush(); err == nil && e != nil && !IsBrokenPipe(e) { err = e }

		case "text":
			if sortOut {
				var buf []nestedoutput.NestedProduct
				for np := range in { buf = append(buf, np) }
				sort.Slice(buf, func(i, j int) bool {
					return common.LessProduct(buf[i].Product, buf[j].Product)
				})
				err = nestedoutput.WriteText(out, buf, header)
			} else {
				err = nestedoutput.StreamText(out, in, header)
			}

		case "fasta":
			// For nested: FASTA would just be outer sequences; reuse Product FASTA by mapping?
			// Keep it simple: advise using ipcr --products + writer if FASTA is needed.
			err = fmt.Errorf("unsupported output %q for ipcr-nested; use text|json|jsonl", format)

		default:
			err = fmt.Errorf("unsupported output %q", format)
		}
		errCh <- err
	}()
	return in, errCh
}

// Minimal inline wrapper to jsonl-encode v1 without creating a new file.
// We avoid pulling in output/probeoutput here to honor boundaries.
type apiNestedLine struct {
	ExperimentID   string `json:"experiment_id"`
	SequenceID     string `json:"sequence_id"`
	Start          int    `json:"start"`
	End            int    `json:"end"`
	Length         int    `json:"length"`
	Type           string `json:"type"`
	FwdMM          int    `json:"fwd_mm,omitempty"`
	RevMM          int    `json:"rev_mm,omitempty"`
	FwdMismatchIdx []int  `json:"fwd_mm_i,omitempty"`
	RevMismatchIdx []int  `json:"rev_mm_i,omitempty"`
	Seq            string `json:"seq,omitempty"`
	SourceFile     string `json:"source_file,omitempty"`
	InnerFound     bool   `json:"inner_found"`
	InnerPairID    string `json:"inner_experiment_id,omitempty"`
	InnerStart     int    `json:"inner_start,omitempty"`
	InnerEnd       int    `json:"inner_end,omitempty"`
	InnerLength    int    `json:"inner_length,omitempty"`
	InnerType      string `json:"inner_type,omitempty"`
	InnerFwdMM     int    `json:"inner_fwd_mm,omitempty"`
	InnerRevMM     int    `json:"inner_rev_mm,omitempty"`
}

func apiWrap(np nestedoutput.NestedProduct) apiNestedLine {
	p := np.Product
	return apiNestedLine{
		ExperimentID:   p.ExperimentID,
		SequenceID:     p.SequenceID,
		Start:          p.Start,
		End:            p.End,
		Length:         p.Length,
		Type:           p.Type,
		FwdMM:          p.FwdMM,
		RevMM:          p.RevMM,
		FwdMismatchIdx: append([]int(nil), p.FwdMismatchIdx...),
		RevMismatchIdx: append([]int(nil), p.RevMismatchIdx...),
		Seq:            p.Seq,
		SourceFile:     p.SourceFile,
		InnerFound:     np.InnerFound,
		InnerPairID:    np.InnerPairID,
		InnerStart:     np.InnerStart,
		InnerEnd:       np.InnerEnd,
		InnerLength:    np.InnerLength,
		InnerType:      np.InnerType,
		InnerFwdMM:     np.InnerFwdMM,
		InnerRevMM:     np.InnerRevMM,
	}
}
