// internal/output/fasta_test.go
package output

import (
	"bytes"
	"ipcr/internal/engine"
	"strings"
	"testing"
)

func TestWriteFASTA(t *testing.T) {
	buf := &bytes.Buffer{}
	list := []engine.Product{{
		ExperimentID: "p1", Seq: "ACGT",
		Start: 0, End: 4, Length: 4,
	}}
	if err := WriteFASTA(buf, list); err != nil {
		t.Fatalf("fasta: %v", err)
	}
	if !strings.Contains(buf.String(), ">p1_1") || !strings.Contains(buf.String(), "ACGT") {
		t.Fatalf("unexpected FASTA output: %s", buf.String())
	}
}
// ===