// internal/probeoutput/json_text_test.go
package probeoutput

import (
	"bytes"
	"encoding/json"
	"ipcr-core/engine"
	"ipcr/pkg/api"
	"strings"
	"testing"
)

func TestJSONRoundTrip(t *testing.T) {
	ap := AnnotatedProduct{}
	ap.Product.ExperimentID = "x"
	ap.Product.SequenceID = "s"
	ap.ProbeName = "p"
	ap.ProbeSeq = "ACG"
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

func TestRenderPrettyIncludesProbeOverlay(t *testing.T) {
	ap := AnnotatedProduct{
		Product: engine.Product{
			FwdPrimer: "AAA",
			RevPrimer: "TTT",
			FwdSite:   "AAA",
			RevSite:   "TTT",
			Length:    20,
		},
		ProbeName:   "probe",
		ProbeSeq:    "CCCC",
		ProbeFound:  true,
		ProbeStrand: "+",
		ProbePos:    5,
		ProbeMM:     0,
		ProbeSite:   "CCCC",
	}

	out := RenderPretty(ap)
	if !strings.Contains(out, `probe "probe"`) {
		t.Fatalf("expected probe overlay summary, got:\n%s", out)
	}
	if !strings.Contains(out, "CCCC") {
		t.Fatalf("expected probe sequence/site in pretty output, got:\n%s", out)
	}
}
