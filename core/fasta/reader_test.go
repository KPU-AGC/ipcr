// core/fasta/reader_test.go
package fasta

import (
	"compress/gzip"
	"context"
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

	defer func() { _ = os.Remove(gzPath) }()

	ch, err := StreamChunks(gzPath, 0, 0)
	if err != nil {
		t.Fatalf("stream gz: %v", err)
	}

	ids := make([]string, 0, 2)
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
		go func() { _, _ = io.WriteString(w, plain); _ = w.Close() }()
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

func TestStreamChunksPathCtx_RollingOverlap(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "chunk.fa")
	if err := os.WriteFile(fn, []byte(">s\nACGTACGTACGT\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var got []Record
	if err := StreamChunksPathCtx(context.Background(), fn, 5, 2, func(r Record) error {
		got = append(got, r)
		return nil
	}); err != nil {
		t.Fatalf("StreamChunksPathCtx: %v", err)
	}

	wantIDs := []string{"s:0-5", "s:3-8", "s:6-11", "s:9-12"}
	wantSeqs := []string{"ACGTA", "TACGT", "GTACG", "CGT"}
	if len(got) != len(wantIDs) {
		t.Fatalf("got %d chunks, want %d: %+v", len(got), len(wantIDs), got)
	}
	for i := range wantIDs {
		if got[i].ID != wantIDs[i] || string(got[i].Seq) != wantSeqs[i] {
			t.Fatalf("chunk %d got %q %q, want %q %q", i, got[i].ID, got[i].Seq, wantIDs[i], wantSeqs[i])
		}
	}
}

func TestStreamChunksPathCtx_ShortRecordKeepsBaseID(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "short.fa")
	if err := os.WriteFile(fn, []byte(">s\nACGTA\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var got []Record
	if err := StreamChunksPathCtx(context.Background(), fn, 5, 2, func(r Record) error {
		got = append(got, r)
		return nil
	}); err != nil {
		t.Fatalf("StreamChunksPathCtx: %v", err)
	}
	if len(got) != 1 || got[0].ID != "s" || string(got[0].Seq) != "ACGTA" {
		t.Fatalf("unexpected chunks: %+v", got)
	}
}
