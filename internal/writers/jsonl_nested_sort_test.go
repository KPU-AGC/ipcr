package writers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"ipcr-core/engine"
	"ipcr/internal/nestedoutput"
	"ipcr/pkg/api"
	"testing"
)

func TestNestedWriter_JSONLSort(t *testing.T) {
	var buf bytes.Buffer
	in, done := StartNestedWriterWithPretty(&buf, "jsonl", true, true, false, 4)
	in <- nestedoutput.NestedProduct{Product: engine.Product{SourceFile: "ref.fa", ExperimentID: "late", SequenceID: "s", Start: 5, End: 10, Length: 5, Type: "forward"}, InnerFound: true}
	in <- nestedoutput.NestedProduct{Product: engine.Product{SourceFile: "ref.fa", ExperimentID: "early", SequenceID: "s", Start: 1, End: 6, Length: 5, Type: "forward"}, InnerFound: true}
	close(in)
	if err := <-done; err != nil {
		t.Fatalf("writer err: %v", err)
	}
	sc := bufio.NewScanner(bytes.NewReader(buf.Bytes()))
	if !sc.Scan() {
		t.Fatalf("missing first JSONL row: %q", buf.String())
	}
	var v api.NestedProductV1
	if err := json.Unmarshal(sc.Bytes(), &v); err != nil {
		t.Fatalf("bad first JSONL row: %v", err)
	}
	if v.ExperimentID != "early" {
		t.Fatalf("expected coord-sorted nested JSONL first row to be early, got %q in output:\n%s", v.ExperimentID, buf.String())
	}
}
