// internal/runutil/runutil.go
package runutil

import (
	"fmt"
)

// EffectiveTerminalWindow returns the enforced 3' terminal-window length.
// Rule: default is 3 (set by flag). Any value < 1 disables the clamp (returns 0).
func EffectiveTerminalWindow(v int) int {
	if v < 1 {
		return 0
	}
	return v
}

// ComputeOverlap returns the overlap used for chunking.
// If maxLen>0, use that (so a product never spans >1 chunk). Otherwise,
// fall back to maxPrimerLen-1 (minimum to keep flanking primers together).
func ComputeOverlap(maxLen, maxPrimerLen int) int {
	if maxLen > 0 {
		return maxLen
	}
	if maxPrimerLen > 0 {
		if v := maxPrimerLen - 1; v > 0 {
			return v
		}
	}
	return 0
}

// ValidateChunking decides effective chunking and emits human-readable warnings.
// Rules (matching tests):
//   - circular => disable (warn)
//   - maxLen==0 => disable (warn)
//   - chunkSize<=maxLen => disable (warn)
//   - else enable with overlap=ComputeOverlap(maxLen,maxPrimerLen)
func ValidateChunking(circular bool, chunkSize, maxLen, maxPrimerLen int) (int, int, []string) {
	var warns []string

	if circular {
		warns = append(warns, "chunking disabled for circular templates")
		return 0, 0, warns
	}
	if maxLen <= 0 {
		warns = append(warns, "chunking disabled: --max-length is required to compute safe overlap")
		return 0, 0, warns
	}
	if chunkSize <= maxLen {
		warns = append(warns, fmt.Sprintf("chunk-size (%d) <= max-length (%d): disabling chunking", chunkSize, maxLen))
		return 0, 0, warns
	}
	ov := ComputeOverlap(maxLen, maxPrimerLen)
	return chunkSize, ov, warns
}
