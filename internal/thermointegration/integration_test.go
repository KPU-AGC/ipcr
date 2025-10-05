package thermointegration

import (
	"bytes"
	"os"
	"testing"

	"ipcr/internal/thermoapp"
)

func writeFA(t *testing.T, name, data string) string {
	t.Helper()
	if err := os.WriteFile(name, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return name
}

func TestThermo_EndToEnd_TSVWithScore(t *testing.T) {
	// Tiny FASTA with a simple amplicon.
	fa := writeFA(t, "thermo_it.fa", ">s\nACGTACAAAAAAGGTACC\n")
	defer func() { _ = os.Remove(fa) }()

	var out, errB bytes.Buffer
	code := thermoapp.Run([]string{
		"--forward", "ACGTAC",
		"--reverse", "GGTACC",
		"--sequences", fa,
		"--output", "text",
		"--sort",
		"--rank", "score",
	}, &out, &errB)
	if code != 0 {
		t.Fatalf("exit %d err=%s", code, errB.String())
	}
	s := out.String()
	if s == "" || s[0] == 0 {
		t.Fatalf("expected non-empty text output")
	}
	if !bytes.Contains(out.Bytes(), []byte("score")) {
		t.Fatalf("expected 'score' in header:\n%s", s)
	}
}
