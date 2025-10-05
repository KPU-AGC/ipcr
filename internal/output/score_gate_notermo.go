//go:build !thermo
// +build !thermo

package output

import (
	"ipcr-core/engine"
	"ipcr/pkg/api"
)

// Non-thermo: do nothing; JSON/JSONL will omit "score".
func applyScoreToAPI(_ *api.ProductV1, _ engine.Product) {}
