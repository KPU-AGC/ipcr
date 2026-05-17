// internal/probeintegration/integration_test.go
package probeintegration

import (
	"bytes"
	"encoding/json"
	"ipcr/internal/probeapp"
	"ipcr/pkg/api"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, fn, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), fn)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestJSONSmoke(t *testing.T) {
	fa := write(t, "ref.fa", ">chr1\nACGTTTACGTTTACGTTT\n")
	var out, errB bytes.Buffer
	code := probeapp.Run([]string{
		"--forward", "ACG",
		"--reverse", "ACG",
		"--sequences", fa,
		"--probe", "ACGTT",
		"--output", "json",
	}, &out, &errB)
	if code != 0 {
		t.Fatalf("exit %d err=%s", code, errB.String())
	}
	var v []api.AnnotatedProductV1
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(v) == 0 {
		t.Fatalf("expected at least one object")
	}
}

func TestRequireProbe_NoHits_ExitZeroByDefault(t *testing.T) {
	fa := write(t, "ref.fa", ">chr1\nACGTTTACGTTTACGTTT\n")

	// --require-probe defaults to true; probe is absent.
	var out, errB bytes.Buffer
	code := probeapp.Run([]string{
		"--forward", "ACG",
		"--reverse", "ACG",
		"--sequences", fa,
		"--probe", "AAAAA", // not present
		"--output", "json",
	}, &out, &errB)

	// New default: --no-match-exit-code=0 → exit 0 when nothing matches.
	if code != 0 {
		t.Fatalf("expected zero exit when no hits under --require-probe=true (got %d, err=%s)", code, errB.String())
	}
}
