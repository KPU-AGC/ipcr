package version

import (
	"fmt"
	"io"
)

// Version is the package/distribution version. It is intentionally monotonic for
// package managers such as Bioconda and may be overridden at build time by:
//
//	-X ipcr/internal/version.Version=<version>
var Version = "5.0.0"

const (
	EngineVersion       = "ac-approx-seed-v0.1"
	ThermoVersion       = "nn-imperfect-v1"
	OutputSchemaVersion = "ipcr-jsonl-v1"
)

// Write prints the package version plus scientific/model subcomponent versions.
func Write(out io.Writer, command string) {
	if command == "" {
		command = "ipcr"
	}
	_, _ = fmt.Fprintf(out, "%s %s\n", command, Version)
	_, _ = fmt.Fprintf(out, "engine: %s\n", EngineVersion)
	_, _ = fmt.Fprintf(out, "thermo: %s\n", ThermoVersion)
	_, _ = fmt.Fprintf(out, "output-schema: %s\n", OutputSchemaVersion)
}
