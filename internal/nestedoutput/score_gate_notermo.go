// internal/nestedoutput/score_gate_notermo.go
//go:build !thermo

package nestedoutput

import (
	"ipcr-core/engine"
	"ipcr/pkg/api"
)

// Non-thermo: omit score field.
func applyScoreToNested(_ *api.NestedProductV1, _ engine.Product) {}
