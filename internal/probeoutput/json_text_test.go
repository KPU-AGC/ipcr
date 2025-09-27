package probeoutput

import (
	"bytes"
	"encoding/json"
	"ipcr/pkg/api"
	"testing"
)

func TestJSONRoundTrip(t *testing.T) {
	ap := AnnotatedProduct{}
	ap.ExperimentID = "x"
	ap.SequenceID = "s"
	ap.ProbeName = "p"
	ap.ProbeSeq  = "ACG"
	ap.ProbeFound = true

	var buf bytes.Buffer
	if err := WriteJSON(&buf, []AnnotatedProduct{ap}); err != nil {
		t.Fatalf("json write: %v", err)
	}
	var got []api.AnnotatedProductV1
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil || len(got) != 1 || got[0].ProbeName != "p" {
		t.Fatalf("json roundtrip failed: %v %#v", err, got)
	}
}
