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

// Record represents a parsed FASTA sequence (or a chunk of one).
type Record struct {
	ID  string
	Seq []byte
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

func openReader(path string) (io.ReadCloser, error) {
	if path == "-" {
		return io.NopCloser(os.Stdin), nil
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	// Detect gzip by magic number (1F 8B) or by .gz suffix.
	var sig [2]byte
	n, _ := fh.Read(sig[:])
	_, _ = fh.Seek(0, io.SeekStart)
	if (n == 2 && sig[0] == 0x1f && sig[1] == 0x8b) || strings.HasSuffix(path, ".gz") {
		gr, err := gzip.NewReader(fh)
		if err != nil {
			fh.Close()
			return nil, err
		}
		return &multiReadCloser{Reader: gr, closers: []io.Closer{gr, fh}}, nil
	}
	return fh, nil
}

// StreamChunks reads FASTA from a file path (or "-" for stdin) and emits
// records (optionally chunked) on a channel. Overlap is the number of bases
// each chunk overlaps the next (step = chunkSize - overlap). If chunkSize<=0,
// the full record is emitted once (no suffix in ID).
func StreamChunks(path string, chunkSize, overlap int) (<-chan Record, error) {
	rc, err := openReader(path)
	if err != nil {
		return nil, err
	}
	out := make(chan Record, 8)

	go func() {
		defer close(out)
		defer rc.Close()

		sc := bufio.NewScanner(rc)
		const maxLine = 64 * 1024 * 1024 // allow very long single-line sequences (64 MiB)
		buf := make([]byte, 64*1024)
		sc.Buffer(buf, maxLine)

		var id string
		seq := make([]byte, 0, 1<<20)

		flush := func() {
			if id == "" {
				return
			}
			if chunkSize <= 0 || chunkSize >= len(seq) {
				out <- Record{ID: id, Seq: append([]byte(nil), seq...)}
				return
			}
			step := chunkSize - overlap
			if step <= 0 {
				out <- Record{ID: id, Seq: append([]byte(nil), seq...)}
				return
			}
			for off := 0; off < len(seq); off += step {
				end := off + chunkSize
				if end > len(seq) {
					end = len(seq)
				}
				out <- Record{
					ID:  fmt.Sprintf("%s:%d-%d", id, off, end),
					Seq: append([]byte(nil), seq[off:end]...),
				}
				if end == len(seq) {
					break
				}
			}
		}

		for sc.Scan() {
			line := sc.Bytes()
			if len(line) == 0 {
				continue
			}
			if line[0] == '>' {
				if id != "" {
					flush()
					seq = seq[:0]
				}
				id = parseHeaderID(line[1:])
				continue
			}
			seq = append(seq, bytes.TrimSpace(line)...)
		}
		if id != "" || len(seq) > 0 {
			flush()
		}
	}()

	return out, nil
}
