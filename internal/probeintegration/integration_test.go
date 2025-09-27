package probeintegration

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"ipcr/internal/probeapp"
	"ipcr/pkg/api"
)

func write(t *testing.T, fn, data string) string {
	t.Helper()
	if err := os.WriteFile(fn, []byte(data), 0644); err != nil {
		t.Fatalf("write %s: %v", fn, err)
	}
	return fn
}

func TestProbeEndToEndJSON(t *testing.T) {
	fa := write(t, "p_itest.fa", ">s\nACGTACGTACGT\n")
	defer os.Remove(fa)

	var out, errB bytes.Buffer
	code := probeapp.Run([]string{
		"--forward", "ACG",
		"--reverse", "ACG",
		"--sequences", fa,
		"--probe", "GTAC",
		"--probe-name", "myprobe",
		"--output", "json",
		"--sort",
	}, &out, &errB)

	if code != 0 {
		t.Fatalf("exit %d err=%s", code, errB.String())
	}
	var got []api.AnnotatedProductV1
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json parse: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected ≥1 annotated amplicon")
	}
	found := false
	for _, ap := range got {
		if ap.ProbeFound && ap.ProbeMM == 0 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected at least one perfect probe hit; got %v", got[0])
	}
}

func TestRequireProbeFilter(t *testing.T) {
	fa := write(t, "p_itest2.fa", ">s\nACGTACGTACGT\n")
	defer os.Remove(fa)

	var out, errB bytes.Buffer
	code := probeapp.Run([]string{
		"--forward", "ACG",
		"--reverse", "ACG",
		"--sequences", fa,
		"--probe", "AAAAA", // not present
		"--output", "json",
	}, &out, &errB)

	if code == 0 {
		t.Fatalf("expected non-zero exit when no hits under --require-probe=true")
	}
}
