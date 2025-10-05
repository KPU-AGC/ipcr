// core/fasta/open.go
package fasta

import (
	"compress/gzip"
	"io"
	"os"
	"strings"
)

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

// openReader keeps existing gzip + "-" (stdin) behavior.
// Used by both path_ctx.go and reader wrappers.
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
			_ = fh.Close()
			return nil, err
		}
		return &multiReadCloser{Reader: gr, closers: []io.Closer{gr, fh}}, nil
	}
	return fh, nil
}
