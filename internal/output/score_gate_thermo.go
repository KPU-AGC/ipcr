// internal/output/score_gate_thermo.go
//go:build thermo

package output

import (
	"ipcr-core/engine"
	"ipcr/pkg/api"
)

// Thermo-only: copy the score into the wire object.
func applyScoreToAPI(dst *api.ProductV1, p engine.Product) { dst.Score = p.Score }
