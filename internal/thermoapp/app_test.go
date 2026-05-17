package thermoapp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseOligoInlineValidatesAndNormalizes(t *testing.T) {
	o, err := parseOligoInline("assay1: acgtry ", 0)
	if err != nil {
		t.Fatalf("parseOligoInline returned error: %v", err)
	}
	if o.ID != "assay1" || o.Seq != "ACGTRY" {
		t.Fatalf("expected normalized oligo, got %+v", o)
	}
}

func TestParseOligoInlineRejectsInvalidBase(t *testing.T) {
	if _, err := parseOligoInline("assay1:ACGX", 0); err == nil {
		t.Fatal("expected invalid oligo error")
	}
}

func TestLoadOligosTSVValidatesAndNormalizes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oligos.tsv")
	if err := os.WriteFile(path, []byte("o1 acgtry\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oligos, err := loadOligosTSV(path)
	if err != nil {
		t.Fatalf("loadOligosTSV returned error: %v", err)
	}
	if len(oligos) != 1 || oligos[0].ID != "o1" || oligos[0].Seq != "ACGTRY" {
		t.Fatalf("expected normalized oligo TSV row, got %+v", oligos)
	}
}

func TestLoadOligosTSVRejectsInvalidBase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oligos.tsv")
	if err := os.WriteFile(path, []byte("o1 ACGX\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOligosTSV(path); err == nil {
		t.Fatal("expected invalid oligo TSV error")
	}
}
