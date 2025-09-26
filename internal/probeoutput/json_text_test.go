package probeoutput

import (
	"bytes"
	"encoding/json"
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
	var got []AnnotatedProduct
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil || len(got) != 1 || got[0].ProbeName != "p" {
		t.Fatalf("json roundtrip failed: %v %#v", err, got)
	}
}

func TestTextHeaderAndRow(t *testing.T) {
	ap := AnnotatedProduct{}
	ap.SourceFile = "ref.fa"
	ap.SequenceID = "s:0-10"
	ap.ExperimentID = "x"
	ap.Start, ap.End, ap.Length, ap.Type = 0, 10, 10, "forward"
	ap.ProbeName, ap.ProbeSeq, ap.ProbeFound, ap.ProbeStrand, ap.ProbePos, ap.ProbeMM, ap.ProbeSite = "p", "AAA", true, "+", 3, 0, "AAA"

	var buf bytes.Buffer
	if err := WriteText(&buf, []AnnotatedProduct{ap}, true); err != nil {
		t.Fatalf("text write: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("probe_name")) || !bytes.Contains(buf.Bytes(), []byte("\tp\t")) {
		t.Fatalf("unexpected TSV: %s", buf.String())
	}
}
