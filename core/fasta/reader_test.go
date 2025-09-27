package fasta

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const plain = `>seq1
ACGT
>seq2
NNnn
`

// writeGz creates a gzipped FASTA file with provided data, returns the file path.
func writeGz(t *testing.T, data string) string {
	tmpdir := os.TempDir()
	path := filepath.Join(tmpdir, fmt.Sprintf("test-%d.fa.gz", time.Now().UnixNano()))
	fh, err := os.Create(path)
	if err != nil {
		t.Fatalf("tmp: %v", err)
	}
	gw := gzip.NewWriter(fh)
	if _, err := gw.Write([]byte(data)); err != nil {
		t.Fatalf("write gz: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	if err := fh.Sync(); err != nil {
		t.Fatalf("sync file: %v", err)
	}
	if err := fh.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}
	return path
}

func TestStreamGzip(t *testing.T) {
	gzPath := writeGz(t, plain)
	defer func(){ _ = os.Remove(gzPath) }()

	ch, err := StreamChunks(gzPath, 0, 0)
	if err != nil {
		t.Fatalf("stream gz: %v", err)
	}

	var ids []string
	for r := range ch {
		ids = append(ids, r.ID)
	}

	if len(ids) != 2 || !strings.HasPrefix(ids[0], "seq1") || !strings.HasPrefix(ids[1], "seq2") {
		t.Fatalf("gzip parse failed, ids=%v", ids)
	}
}

func TestStreamStdin(t *testing.T) {
	// Fake stdin by swapping os.Stdin
	orig := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = orig }()

	// Write sample then close writer to signal EOF
	go func() {
	_, _ = io.WriteString(w, plain)
	_ = w.Close()
	}()

	ch, err := StreamChunks("-", 0, 0)
	if err != nil {
		t.Fatalf("stream stdin: %v", err)
	}
	count := 0
	for range ch {
		count++
	}
	if count != 2 {
		t.Fatalf("expected 2 records from stdin, got %d", count)
	}
}
