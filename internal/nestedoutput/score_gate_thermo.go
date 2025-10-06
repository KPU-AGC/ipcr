// internal/nestedoutput/score_gate_thermo.go
//go:build thermo

package nestedoutput

import (
	"ipcr-core/engine"
	"ipcr/pkg/api"
)

// Thermo-only: attach score.
func applyScoreToNested(dst *api.NestedProductV1, p engine.Product) { dst.Score = p.Score }
