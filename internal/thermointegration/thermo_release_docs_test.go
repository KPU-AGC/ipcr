package thermointegration

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestThermoReleaseDocsDeclareModelBoundaries(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	data, err := os.ReadFile(filepath.Join(repoRoot, "docs", "THERMO_MODELS.md"))
	if err != nil {
		t.Fatalf("read thermo model docs: %v", err)
	}
	doc := string(data)
	for _, want := range []string{
		"PCR kinetics simulator",
		"nn-duplex-v1",
		"nn-structure-v1",
		"binding",
		"pcr",
		"gel",
		"owczarzy08",
		"IUPAC thermodynamics policy",
		"Modified probes are not fully modeled",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("thermo docs missing %q", want)
		}
	}
}
