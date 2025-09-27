package writers

import (
	"bytes"
	"ipcr-core/engine"
	"ipcr/internal/probeoutput"
	"strings"
	"testing"
)

func TestStartAnnotatedWriter_TextHeader(t *testing.T) {
	var buf bytes.Buffer
	in, done := StartAnnotatedWriter(&buf, "text", false, true, false, 2)
	in <- probeoutput.AnnotatedProduct{
		Product: engine.Product{
			SourceFile: "ref.fa", SequenceID: "s:0-10", ExperimentID: "x",
			Start: 0, End: 10, Length: 10, Type: "forward",
		},
		ProbeName: "p", ProbeSeq: "AAA", ProbeFound: true, ProbeStrand: "+", ProbePos: 3,
	}
	close(in)
	if err := <-done; err != nil {
		t.Fatalf("writer err: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "probe_name") || !strings.Contains(out, "\tp\t") {
		t.Fatalf("unexpected TSV text:\n%s", out)
	}
}
