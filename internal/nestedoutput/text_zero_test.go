package nestedoutput

import (
	"bytes"
	"strings"
	"testing"

	"ipcr-core/engine"
)

func TestWriteText_InnerAtZero_PrintsZero(t *testing.T) {
	var buf bytes.Buffer
	np := NestedProduct{
		Product: engine.Product{
			SourceFile: "ref.fa", SequenceID: "s:0-8", ExperimentID: "outer",
			Start: 0, End: 8, Length: 8, Type: "forward",
		},
		InnerFound:  true,
		InnerPairID: "inner",
		InnerStart:  0,
		InnerEnd:    4,
		InnerLength: 4,
		InnerType:   "forward",
		InnerFwdMM:  0,
		InnerRevMM:  0,
	}
	if err := WriteText(&buf, []NestedProduct{np}, true); err != nil {
		t.Fatalf("write text: %v", err)
	}
	line := ""
	for _, l := range strings.Split(buf.String(), "\n") {
		if strings.HasPrefix(l, "ref.fa") {
			line = l
			break
		}
	}
	if line == "" || !strings.Contains(line, "\ttrue\t0\t") {
		t.Fatalf("expected inner_start=0 printed; got:\n%s", buf.String())
	}
}
