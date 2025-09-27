// internal/primer/loader_test.go
package primer

import (
	"os"
	"testing"
)

func TestLoadTSV(t *testing.T) {
	tmp := "tmp_primers.tsv"
	os.WriteFile(tmp, []byte("p1 ACG ACG 5 15\n#comment\n"), 0644)
	defer func(){ _ = os.Remove(tmp) }()

	ps, err := LoadTSV(tmp)
	if err != nil || len(ps) != 1 || ps[0].ID != "p1" || ps[0].MinProduct != 5 {
		t.Fatalf("LoadTSV failed: %+v %v", ps, err)
	}
}
// ===