// internal/output/json_test.go
package output

import (
	"bytes"
	"encoding/json"
	"ipcr/internal/engine"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	list := []engine.Product{{
		ExperimentID: "p1", SequenceID: "s", Start: 0, End: 3, Length: 3, Type: "forward",
	}}
	if err := WriteJSON(buf, list); err != nil {
		t.Fatalf("json write: %v", err)
	}
	var got []engine.Product
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil || len(got) != 1 || got[0].ExperimentID != "p1" {
		t.Fatalf("json round-trip failed: %v %v", err, got)
	}
}
// ===