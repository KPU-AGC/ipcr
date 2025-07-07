// internal/integration/integration_test.go
package integration

import (
	"fmt"
	"bytes"
	"os"
	"testing"

	"ipcr/internal/app"
)

func write(t *testing.T, fn, data string) string {
	t.Helper()
	if err := os.WriteFile(fn, []byte(data), 0644); err != nil {
		t.Fatalf("write %s: %v", fn, err)
	}
	return fn
}

func TestEndToEnd(t *testing.T) {
	fa := write(t, "itest.fa", ">s\nACGTACGTACGT\n")
	defer os.Remove(fa)

	var out, errBuf bytes.Buffer
	code := app.Run([]string{
		"--forward", "ACG",
		"--reverse", "ACG",
		"--sequences", fa,
	}, &out, &errBuf)

	if code != 0 {
		t.Fatalf("run exit %d, err=%s", code, errBuf.String())
	}
	if out.Len() == 0 {
		t.Fatalf("expected text output")
	}
}

func TestParallelMatchesEqualSerial(t *testing.T) {
	fa := write(t, "par.fa", ">s\nACGTACGTACGT\n")
	defer os.Remove(fa)

	run := func(threads int) string {
		var out, errB bytes.Buffer
		code := app.Run([]string{
			"--forward", "ACG",
			"--reverse", "ACG",
			"--sequences", fa,
			"--threads", fmt.Sprint(threads),
			"--output", "json",
		}, &out, &errB)
		if code != 0 {
			t.Fatalf("exit %d err %s", code, errB.String())
		}
		return out.String()
	}

	serial := run(1)
	parallel := run(4)

	if serial != parallel {
		t.Fatalf("parallel output differs from serial\nserial: %s\nparallel:%s", serial, parallel)
	}
}
// ===