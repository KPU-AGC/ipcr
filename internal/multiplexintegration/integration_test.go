package multiplexintegration

import (
	"bytes"
	"os"
	"testing"

	"ipcr/internal/multiplexapp"
)

func write(t *testing.T, name, data string) string {
	t.Helper()
	if err := os.WriteFile(name, []byte(data), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return name
}

func TestMultiplexJSON_EndToEnd(t *testing.T) {
	fa := write(t, "mplex.fa", ">s\nACGTACGTACGT\n")
	defer os.Remove(fa)

	// Two trivial pairs in TSV
	tsv := write(t, "pairs.tsv", "p1 ACG ACG\np2 CGT CGT\n")
	defer os.Remove(tsv)

	var out, errB bytes.Buffer
	code := multiplexapp.Run([]string{
		"--primers", tsv,
		"--sequences", fa,
		"--output", "json",
		"--sort",
	}, &out, &errB)

	if code != 0 {
		t.Fatalf("exit %d err=%s", code, errB.String())
	}
	if out.Len() == 0 {
		t.Fatalf("expected JSON output")
	}
}
