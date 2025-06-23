// internal/fasta/reader.go
package fasta

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"strings"
)

// Record is unchanged.
type Record struct {
	ID  string
	Seq []byte
}

// openReader returns an io.ReadCloser for a path.
//   • "-"  → os.Stdin (never closed here; caller responsibility)
//   • "*.gz" → gzip reader
//   • else plain file
func openReader(path string) (io.ReadCloser, error) {
	if path == "-" {
		// wrap Stdin in a no‑op closer so caller has uniform interface
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
		// close both when gr is closed
		return struct {
			io.Reader
			io.Closer
		}{Reader: gr, Closer: fh}, nil
	}
	return fh, nil
}

// Stream now uses openReader.
func Stream(path string) (<-chan Record, error) {
	rc, err := openReader(path)
	if err != nil {
		return nil, err
	}

	out := make(chan Record, 4)

	go func() {
		defer rc.Close()
		defer close(out)

		reader := bufio.NewReader(rc)
		var (
			id  string
			buf []byte
		)

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil && err != io.EOF {
				panic(err)
			}
			if len(line) > 0 && line[len(line)-1] == '\n' {
				line = line[:len(line)-1]
			}
			if len(line) == 0 && err == io.EOF {
				break
			}
			if len(line) > 0 && line[0] == '>' {
				if id != "" {
					out <- Record{ID: id, Seq: bytes.ToUpper(buf)}
				}
				id = strings.Fields(string(line[1:]))[0]
				buf = buf[:0]
			} else {
				buf = append(buf, line...)
			}
			if err == io.EOF {
				break
			}
		}
		if id != "" {
			out <- Record{ID: id, Seq: bytes.ToUpper(buf)}
		}
	}()
	return out, nil
}
