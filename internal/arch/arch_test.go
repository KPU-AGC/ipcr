// ./internal/arch/arch_test.go
package arch

import (
	"bytes"
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"testing"
)

type pkg struct {
	ImportPath string
	Imports    []string
	Standard   bool
}

func TestImportBoundaries(t *testing.T) {
	cmd := exec.Command("go", "list", "-json", "./...")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("go list: %v", err)
	}
	dec := json.NewDecoder(&out)

	bans := map[string][]string{
		"ipcr/internal/engine": {
			"ipcr/internal/pipeline", "ipcr/internal/writers",
			"ipcr/internal/output", "ipcr/internal/probeoutput", "ipcr/internal/nestedoutput",
			"ipcr/internal/cli", "ipcr/internal/probecli", "ipcr/internal/nestedcli",
			"ipcr/internal/appcore", "ipcr/internal/app", "ipcr/cmd/",
		},
		"ipcr/internal/pipeline": {
			"ipcr/internal/appcore", "ipcr/internal/app",
			"ipcr/internal/cli", "ipcr/internal/probecli", "ipcr/internal/nestedcli",
			"ipcr/cmd/",
		},
		"ipcr/internal/writers": {
			"ipcr/internal/appcore", "ipcr/internal/app",
			"ipcr/internal/cli", "ipcr/internal/probecli", "ipcr/internal/nestedcli",
			"ipcr/internal/pipeline", "ipcr/cmd/",
		},
		"ipcr/internal/output": {
			"ipcr/internal/appcore", "ipcr/internal/app",
			"ipcr/internal/cli", "ipcr/internal/probecli", "ipcr/internal/nestedcli",
			"ipcr/internal/pipeline", "ipcr/cmd/",
		},
		"ipcr/internal/probeoutput": {
			"ipcr/internal/appcore", "ipcr/internal/app",
			"ipcr/internal/cli", "ipcr/internal/probecli", "ipcr/internal/nestedcli",
			"ipcr/internal/pipeline", "ipcr/cmd/",
		},
		"ipcr/internal/nestedoutput": {
			"ipcr/internal/appcore", "ipcr/internal/app",
			"ipcr/internal/cli", "ipcr/internal/probecli", "ipcr/internal/nestedcli",
			"ipcr/internal/pipeline", "ipcr/cmd/",
		},
		"ipcr/internal/pretty": {
			"ipcr/internal/appcore", "ipcr/internal/app",
			"ipcr/internal/cli", "ipcr/internal/probecli", "ipcr/internal/nestedcli",
			"ipcr/internal/pipeline", "ipcr/cmd/",
		},
	}

	var violations []string
	for {
		var p pkg
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !strings.HasPrefix(p.ImportPath, "ipcr/") {
			continue
		}
		imp := p.ImportPath
		for prefix, forbidden := range bans {
			if !strings.HasPrefix(imp, prefix) {
				continue
			}
			for _, dep := range p.Imports {
				if !strings.HasPrefix(dep, "ipcr/") {
					continue
				}
				for _, ban := range forbidden {
					if strings.HasPrefix(dep, ban) {
						violations = append(violations, imp+" → "+dep)
					}
				}
			}
		}
	}

	if len(violations) > 0 {
		t.Fatalf("import boundary violations:\n  %s", strings.Join(violations, "\n  "))
	}
}
