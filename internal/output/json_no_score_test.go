// internal/output/json_no_score_test.go
//go:build !thermo

package output

import (
	"bytes"
	"encoding/json"
	"ipcr-core/engine"
	"testing"
)

func TestJSON_OmitsScore_InNonThermoBuild(t *testing.T) {
	var buf bytes.Buffer
	p := engine.Product{
		ExperimentID: "x",
		SequenceID:   "s",
		Start:        0,
		End:          10,
		Length:       10,
		Type:         "forward",
		Score:        42, // in-memory score set
	}
	if err := WriteJSON(&buf, []engine.Product{p}); err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatalf("expected JSON array with at least one element")
	}
	if _, ok := out[0]["score"]; ok {
		t.Fatalf("score should be omitted in non-thermo builds")
	}
}
