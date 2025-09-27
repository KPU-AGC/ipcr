package pretty

import (
	"os"
	"path/filepath"
	"testing"

	"ipcr-core/engine"
)

func writeIfMissingOrUpdate(path string, got string) (created bool, err error) {
	// Ensure the testdata directory exists before writing.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	// Allow updating goldens explicitly.
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		return true, os.WriteFile(path, []byte(got), 0644)
	}
	// First-run: create golden if missing.
	if _, e := os.Stat(path); os.IsNotExist(e) {
		return true, os.WriteFile(path, []byte(got), 0644)
	}
	return false, nil
}

func mustRead(path string, t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}
	return string(b)
}

func TestRenderProductForward_Golden(t *testing.T) {
	p := engine.Product{
		FwdPrimer: "AAA", RevPrimer: "TTT",
		FwdSite: "AAA",  RevSite: "TTT",
		Length: 22, Start: 0, End: 22, Type: "forward",
	}
	got := RenderProduct(p)
	path := filepath.Join("testdata", "forward.golden")
	if created, err := writeIfMissingOrUpdate(path, got); err != nil {
		t.Fatalf("write golden: %v", err)
	} else if created {
		t.Logf("wrote %s", path)
		return
	}
	want := mustRead(path, t)
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderProductRevcomp_Golden(t *testing.T) {
	p := engine.Product{
		FwdPrimer: "ACGT", RevPrimer: "ACGT",
		FwdSite: "ACGT",  RevSite: "ACGT",
		Length: 30, Start: 10, End: 40, Type: "revcomp",
	}
	got := RenderProduct(p)
	path := filepath.Join("testdata", "revcomp.golden")
	if created, err := writeIfMissingOrUpdate(path, got); err != nil {
		t.Fatalf("write golden: %v", err)
	} else if created {
		t.Logf("wrote %s", path)
		return
	}
	want := mustRead(path, t)
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderAnnotated_Plus_Golden(t *testing.T) {
	p := engine.Product{
		FwdPrimer: "TCAG", RevPrimer: "GATC",
		FwdSite: "TCAG",  RevSite: "GATC",
		Length: 40, Start: 0, End: 40, Type: "forward",
	}
	ann := ProbeAnnotation{
		Name: "probe", Seq: "GTACGT", Found: true, Strand: "+", Pos: 6, MM: 0, Site: "GTACGT",
	}
	got := RenderAnnotated(p, ann)
	path := filepath.Join("testdata", "probe_plus.golden")
	if created, err := writeIfMissingOrUpdate(path, got); err != nil {
		t.Fatalf("write golden: %v", err)
	} else if created {
		t.Logf("wrote %s", path)
		return
	}
	want := mustRead(path, t)
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderAnnotated_Minus_Golden(t *testing.T) {
	p := engine.Product{
		FwdPrimer: "TCAGGCCTTTGCTACAATGAAC",
		RevPrimer: "TCAGGCCTTTGCTACAATGAAC",
		FwdSite:   "TCAGGCCTTTGCTACAAYGAAC", // Y to make mismatches obvious if you tweak visuals
		RevSite:   "TCAGGCCTTTGCTACAATGAAC",
		Length:    112, Start: 353221, End: 353333, Type: "revcomp",
	}
	ann := ProbeAnnotation{
		Name: "probe", Seq: "AACTGCATCATATCACATACT", Found: true, Strand: "-", Pos: 52, MM: 0, Site: "AGTATGTGATATGATGCAGTT",
	}
	got := RenderAnnotated(p, ann)
	path := filepath.Join("testdata", "probe_minus.golden")
	if created, err := writeIfMissingOrUpdate(path, got); err != nil {
		t.Fatalf("write golden: %v", err)
	} else if created {
		t.Logf("wrote %s", path)
		return
	}
	want := mustRead(path, t)
	if got != want {
		t.Fatalf("mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
