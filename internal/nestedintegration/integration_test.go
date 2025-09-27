package nestedintegration

import (
	"bytes"
	"ipcr/internal/nestedapp"
	"os"
	"testing"
)

func write(t *testing.T, name, data string) string {
	t.Helper()
	if err := os.WriteFile(name, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return name
}

func TestNestedText_EndToEnd(t *testing.T) {
	fa := write(t, "n_itest.fa", ">s\nACGTACGTACGT\n")
	defer func() { _ = os.Remove(fa) }()

	var out, errB bytes.Buffer
	code := nestedapp.Run([]string{
		"--forward", "ACG",
		"--reverse", "ACG",
		"--inner-forward", "ACG",
		"--inner-reverse", "ACG",
		"--sequences", fa,
		"--output", "text",
		"--no-header",
		"--sort",
	}, &out, &errB)
	if code != 0 {
		t.Fatalf("exit %d err=%s", code, errB.String())
	}
	if out.Len() == 0 {
		t.Fatalf("expected at least one TSV row")
	}
}
