package writers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"

	"ipcr-core/engine"
	"ipcr/internal/probeoutput"
	"ipcr/pkg/api"
)

func TestAnnotatedJSONL_StreamsValidV1(t *testing.T) {
	var buf bytes.Buffer
	in, done := StartAnnotatedJSONLWriter(&buf, 2)
	in <- probeoutput.AnnotatedProduct{
		Product: engine.Product{ExperimentID: "x", SequenceID: "s:0-4", Start: 0, End: 4, Length: 4, Type: "forward"},
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
