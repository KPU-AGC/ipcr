package fasta

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStreamChunksCtxPath_CancelImmediately_YieldsNoRecords(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "x.fa")
	if err := os.WriteFile(fn, []byte(">s\nACGT\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled

	ch, err := StreamChunksCtxPath(ctx, fn, 0, 0)
	if err != nil {
		t.Fatalf("StreamChunksCtxPath: %v", err)
	}
	n := 0
	for range ch {
		n++
	}
	if n != 0 {
		t.Fatalf("expected 0 records due to immediate cancel, got %d", n)
	}
}
