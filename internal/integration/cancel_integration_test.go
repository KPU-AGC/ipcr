package integration

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"ipcr/internal/app"
)

func TestCtrlC_MidScan_Exit130(t *testing.T) {
	// Biggish FASTA to ensure scanning is underway.
	fn := "cancel_big.fa"
	defer os.Remove(fn)
	const Mb = 1 << 20
	seq := strings.Repeat("ACGT", (8*Mb)/4) // ~8MB
	if err := os.WriteFile(fn, []byte(">chr1\n"+seq+"\n"), 0644); err != nil {
		t.Fatalf("write fasta: %v", err)
	}

	argv := []string{
		"--forward", "ACGTACGT",
		"--reverse", "ACGTACGT",
		fn, // positional sequences arg is supported
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel shortly after start.
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	code := app.RunContext(ctx, argv, io.Discard, io.Discard)
	if code != 130 {
		t.Fatalf("expected exit 130 on cancel, got %d", code)
	}
}
