package writers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"ipcr-core/engine"
	"ipcr/internal/probeoutput"
	"ipcr/pkg/api"
	"testing"
)

func TestAnnotatedJSONL_StreamsValidV1(t *testing.T) {
	var buf bytes.Buffer
	in, done := StartAnnotatedJSONLWriter(&buf, 2)
	in <- probeoutput.AnnotatedProduct{
		Product:   engine.Product{ExperimentID: "x", SequenceID: "s:0-4", Start: 0, End: 4, Length: 4, Type: "forward"},
		ProbeName: "p", ProbeSeq: "ACG", ProbeFound: true, ProbeStrand: "+", ProbePos: 1, ProbeMM: 0, ProbeSite: "CGT",
	}
	close(in)
	if err := <-done; err != nil {
		t.Fatalf("writer err: %v", err)
	}

	sc := bufio.NewScanner(bytes.NewReader(buf.Bytes()))
	var v api.AnnotatedProductV1
	if !sc.Scan() || json.Unmarshal(sc.Bytes(), &v) != nil || v.ProbeName != "p" {
		t.Fatalf("bad jsonl annotated: %q", sc.Text())
	}
}

func TestAnnotatedWriter_JSONLSort(t *testing.T) {
	var buf bytes.Buffer
	in, done := StartAnnotatedWriter(&buf, "jsonl", true, true, false, 4)
	in <- probeoutput.AnnotatedProduct{Product: engine.Product{SourceFile: "ref.fa", ExperimentID: "late", SequenceID: "s", Start: 5, End: 10, Length: 5, Type: "forward"}, ProbeName: "p"}
	in <- probeoutput.AnnotatedProduct{Product: engine.Product{SourceFile: "ref.fa", ExperimentID: "early", SequenceID: "s", Start: 1, End: 6, Length: 5, Type: "forward"}, ProbeName: "p"}
	close(in)
	if err := <-done; err != nil {
		t.Fatalf("writer err: %v", err)
	}
	sc := bufio.NewScanner(bytes.NewReader(buf.Bytes()))
	if !sc.Scan() {
		t.Fatalf("missing first JSONL row: %q", buf.String())
	}
	var v api.AnnotatedProductV1
	if err := json.Unmarshal(sc.Bytes(), &v); err != nil {
		t.Fatalf("bad first JSONL row: %v", err)
	}
	if v.ExperimentID != "early" {
		t.Fatalf("expected coord-sorted annotated JSONL first row to be early, got %q in output:\n%s", v.ExperimentID, buf.String())
	}
}
