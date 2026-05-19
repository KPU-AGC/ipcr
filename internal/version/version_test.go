package version

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteIncludesPackageAndComponentVersions(t *testing.T) {
	var buf bytes.Buffer
	Write(&buf, "ipcr")
	out := buf.String()
	for _, want := range []string{
		"ipcr " + Version,
		"engine: " + EngineVersion,
		"thermo: " + ThermoVersion,
		"output-schema: " + OutputSchemaVersion,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("version output missing %q in:\n%s", want, out)
		}
	}
}

func TestWriteDefaultsCommandName(t *testing.T) {
	var buf bytes.Buffer
	Write(&buf, "")
	if !strings.HasPrefix(buf.String(), "ipcr ") {
		t.Fatalf("default version command prefix = %q", buf.String())
	}
}
