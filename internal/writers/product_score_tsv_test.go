package writers

import (
	"bytes"
	"strings"
	"testing"

	"ipcr-core/engine"
)

func TestProductWriter_TSVScoreHeaderAndSort(t *testing.T) {
	var buf bytes.Buffer
	in, done := StartProductWriter(&buf, "text", true, true, false, true, true, 4)

	// Two rows out of order by score; sorting by score desc should flip them.
	in <- engine.Product{SourceFile: "ref.fa", SequenceID: "s", ExperimentID: "x", Start: 0, End: 10, Length: 10, Type: "forward", Score: 1.5}
	in <- engine.Product{SourceFile: "ref.fa", SequenceID: "s", ExperimentID: "y", Start: 1, End: 11, Length: 10, Type: "forward", Score: 3.2}
	close(in)
	if err := <-done; err != nil {
		t.Fatalf("writer err: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("unexpected TSV: %q", out)
	}
	if !strings.Contains(lines[0], "score") {
		t.Fatalf("expected header to include 'score', got: %q", lines[0])
	}
	// First data line should be the higher score (3.2)
	if !strings.Contains(lines[1], "\t3.2") {
		t.Fatalf("expected first row to be score=3.2, got: %q", lines[1])
	}
}
