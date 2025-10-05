// core/primer/loader_test.go
package primer

import (
	"os"
	"testing"
)

func TestLoadTSV(t *testing.T) {
	tmp := "tmp_primers.tsv"
	if err := os.WriteFile(tmp, []byte("p1 ACG ACG 5 15\n#comment\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmp) }()

	ps, err := LoadTSV(tmp)
	if err != nil || len(ps) != 1 || ps[0].ID != "p1" || ps[0].MinProduct != 5 || ps[0].MaxProduct != 15 {
		t.Fatalf("LoadTSV failed: %+v %v", ps, err)
	}
}

func TestLoadTSV_MinOnly_4Fields_OK(t *testing.T) {
	tmp := "tmp_primers_min.tsv"
	if err := os.WriteFile(tmp, []byte("p2 TTA GGA 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmp) }()

	ps, err := LoadTSV(tmp)
	if err != nil {
		t.Fatalf("LoadTSV 4-field should succeed, got err: %v", err)
	}
	if len(ps) != 1 {
		t.Fatalf("expected 1 row, got %d", len(ps))
	}
	if ps[0].ID != "p2" || ps[0].Forward != "TTA" || ps[0].Reverse != "GGA" {
		t.Fatalf("unexpected row: %+v", ps[0])
	}
	if ps[0].MinProduct != 7 || ps[0].MaxProduct != 0 {
		t.Fatalf("expected min=7 max=0, got min=%d max=%d", ps[0].MinProduct, ps[0].MaxProduct)
	}
}
