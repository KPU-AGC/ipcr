// internal/fasta/reader.go
package fasta

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
)

// Record is unchanged.
type Record struct {
	ID  string
	Seq []byte
}

// StreamChunks streams windows of at most win bp with `overlap`.
// If win==0 it behaves like the old Stream (whole record).
func StreamChunks(path string, win, overlap int) (<-chan Record, error) {
	rc, err := openReader(path)
	if err != nil { return nil, err }

	out := make(chan Record, 4)

	go func() {
		defer rc.Close()
		defer close(out)

		r := bufio.NewReader(rc)
		var (
			id         string
			buf        []byte
			startCoord int
		)

		flushWindow := func(endCoord int) {
			if len(buf) == 0 { return }
			out <- Record{
				ID:  fmt.Sprintf("%s:%d-%d", id, startCoord, endCoord),
				Seq: bytes.Clone(buf), // copy; workers may mutate
			}
		}

		for {
			line, err := r.ReadBytes('\n')
			eof := err == io.EOF
			if len(line) > 0 && line[len(line)-1] == '\n' {
				line = line[:len(line)-1]
			}
			if eof && len(line) == 0 {
				break
			}
			if len(line) > 0 && line[0] == '>' { // new header
				// flush residual buffer from previous record
				if id != "" {
					flushWindow(startCoord + len(buf))
				}
				id = strings.Fields(string(line[1:]))[0]
				buf = buf[:0]
				startCoord = 0
				continue
			}
			// sequence line
			line = bytes.ToUpper(line)
			for len(line) > 0 {
				rem := win - len(buf)
				if win == 0 { rem = len(line) }         // no chunking
				if rem > len(line) { rem = len(line) }
				buf = append(buf, line[:rem]...)
				line = line[rem:]

				// if buffer full, flush and slide
				if win > 0 && len(buf) == win {
					flushWindow(startCoord + win)
					// slide buffer by (win - overlap)
					slide := win - overlap
					if slide < 1 { slide = win } // avoid infinite loop
					startCoord += slide
					buf = append([]byte(nil), buf[slide:]...) // copy tail
				}
			}
			if eof { break }
		}
		// flush tail of last record
		if id != "" {
			flushWindow(startCoord + len(buf))
		}
	}()
	return out, nil
}

/* ---------------- small helpers ---------------- */

func openReader(path string) (io.ReadCloser, error) {
	if path == "-" {
		return io.NopCloser(os.Stdin), nil
	}
	fh, err := os.Open(path)
	if err != nil { return nil, err }
	if strings.HasSuffix(path, ".gz") {
		gr, err := gzip.NewReader(fh)
		if err != nil { fh.Close(); return nil, err }
		return struct {
			io.Reader
			io.Closer
		}{Reader: gr, Closer: fh}, nil
	}
	return fh, nil
}

