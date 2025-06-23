// internal/fasta/reader_test.go
package fasta

import (
	"compress/gzip"
	"io"
	"os"
	"testing"
)

const plain = `>seq1
ACGT
>seq2
NNnn
`

func writeGz(t *testing.T, name string, data string) string {
	fh, err := os.CreateTemp("", name)
	if err != nil { t.Fatalf("tmp: %v", err) }
	gw := gzip.NewWriter(fh)
	if _, err := gw.Write([]byte(data)); err != nil { t.Fatalf("write gz: %v", err) }
	gw.Close(); fh.Close()
	return fh.Name()
}

func TestStreamGzip(t *testing.T) {
	gzPath := writeGz(t, "test.fa.gz", plain)
	defer os.Remove(gzPath)

	ch, err := Stream(gzPath)
	if err != nil { t.Fatalf("stream gz: %v", err) }

	var ids []string
	for r := range ch {
		ids = append(ids, r.ID)
	}
	if len(ids) != 2 || ids[0] != "seq1" || ids[1] != "seq2" {
		t.Fatalf("gzip parse failed, ids=%v", ids)
	}
}

func TestStreamStdin(t *testing.T) {
	// fake stdin by swapping os.Stdin
	orig := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = orig }()
	// write sample then close writer to signal EOF
	go func() { io.WriteString(w, plain); w.Close() }()

	ch, err := Stream("-")
	if err != nil { t.Fatalf("stream stdin: %v", err) }
	count := 0
	for range ch { count++ }
	if count != 2 {
		t.Fatalf("expected 2 records from stdin, got %d", count)
	}
}
