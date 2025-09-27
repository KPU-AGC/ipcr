package writers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"ipcr-core/engine"
	"ipcr/pkg/api"
	"testing"
)

func TestProductJSONL_StreamsValidV1(t *testing.T) {
	var buf bytes.Buffer
	in, done := StartProductJSONLWriter(&buf, 2)
	in <- engine.Product{ExperimentID: "x", SequenceID: "s:0-4", Start: 0, End: 4, Length: 4, Type: "forward"}
	in <- engine.Product{ExperimentID: "y", SequenceID: "s:2-6", Start: 2, End: 6, Length: 4, Type: "revcomp"}
	close(in)
	if err := <-done; err != nil {
		t.Fatalf("writer err: %v", err)
	}

	sc := bufio.NewScanner(bytes.NewReader(buf.Bytes()))
	var n int
	for sc.Scan() {
		n++
		var v api.ProductV1
		if err := json.Unmarshal(sc.Bytes(), &v); err != nil {
			t.Fatalf("bad json line %d: %v\n%s", n, err, sc.Text())
		}
	}
	if n != 2 {
		t.Fatalf("want 2 lines, got %d", n)
	}
}
