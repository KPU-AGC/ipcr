package writers

import (
	"bytes"
	"encoding/json"
	"testing"

	"ipcr/internal/engine"
)

func TestStartProductWriter_JSON(t *testing.T) {
	var buf bytes.Buffer
	in, done := StartProductWriter(&buf, "json", true, false, false, 4)
	in <- engine.Product{ExperimentID: "x", SequenceID: "s", Start: 0, End: 4, Length: 4, Type: "forward"}
	in <- engine.Product{ExperimentID: "y", SequenceID: "s", Start: 2, End: 6, Length: 4, Type: "revcomp"}
	close(in)
	if err := <-done; err != nil {
		t.Fatalf("writer err: %v", err)
	}
	var got []engine.Product
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil || len(got) != 2 {
		t.Fatalf("json roundtrip: %v len=%d", err, len(got))
	}
}
