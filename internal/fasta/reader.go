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

// Record represents a parsed FASTA sequence chunk.
type Record struct {
	ID  string
	Seq []byte
}

// StreamChunks streams sequence windows (chunks) from a FASTA file.
// If win==0, yields whole records as a single chunk.
func StreamChunks(path string, win, overlap int) (<-chan Record, error) {
	rc, err := openReader(path)
	if err != nil {
		return nil, err
	}

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
			if len(buf) == 0 {
				return
			}
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
			if len(line) > 0 && line[0] == '>' {
				// New header: flush previous record (if any)
				if id != "" {
					flushWindow(startCoord + len(buf))
				}
				id = strings.Fields(string(line[1:]))[0]
				buf = buf[:0]
				startCoord = 0
				continue
			}
			// Sequence line
			line = bytes.ToUpper(line)
			for len(line) > 0 {
				rem := win - len(buf)
				if win == 0 {
					rem = len(line)
				}
				if rem > len(line) {
					rem = len(line)
				}
				buf = append(buf, line[:rem]...)
				line = line[rem:]

				// If buffer full, flush and slide window
				if win > 0 && len(buf) == win {
					flushWindow(startCoord + win)
					slide := win - overlap
					if slide < 1 {
						slide = win // avoid infinite loop
					}
					startCoord += slide
					buf = append([]byte(nil), buf[slide:]...) // copy tail
				}
			}
			if eof {
				break
			}
		}
		// Flush tail of last record
		if id != "" {
			flushWindow(startCoord + len(buf))
		}
	}()
	return out, nil
}

// multiReadCloser closes multiple io.Closers when Close() is called.
type multiReadCloser struct {
	io.Reader
	closers []io.Closer
}

func (m *multiReadCloser) Close() error {
	var err error
	for _, c := range m.closers {
		if cerr := c.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

// openReader opens a file (possibly gzip) or stdin for streaming.
func openReader(path string) (io.ReadCloser, error) {
	if path == "-" {
		return io.NopCloser(os.Stdin), nil
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(path, ".gz") {
		gr, err := gzip.NewReader(fh)
		if err != nil {
			fh.Close()
			return nil, err
		}
		return &multiReadCloser{
			Reader:  gr,
			closers: []io.Closer{gr, fh},
		}, nil
	}
	return fh, nil
}
// ===