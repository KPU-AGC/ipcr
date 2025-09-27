// internal/runutil/runutil.go
package runutil

import (
	"fmt"
	"strings"

	"ipcr/internal/output"
)

// ComputeTerminalWindow picks the default 3' terminal-window length.
// override >= 0 takes precedence; otherwise "realistic" => 3, "debug" => 0.
func ComputeTerminalWindow(mode string, override int) int {
	if override >= 0 {
		return override
	}
	switch strings.ToLower(mode) {
	case "debug":
		return 0
	default: // realistic (or anything else) defaults to 3
		return 3
	}
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
//   • circular => disable (warn)
//   • maxLen==0 => disable (warn)
//   • chunkSize<=maxLen => disable (warn)
//   • else enable with overlap=ComputeOverlap(maxLen,maxPrimerLen)
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

// ComputeNeedSeq decides if the pipeline must materialize sequence strings.
// True if: --products, FASTA output, or pretty rendering requested (any format).
func ComputeNeedSeq(format string, pretty bool, products bool) bool {
	if products {
		return true
	}
	if format == output.FormatFASTA {
		return true
	}
	if pretty { // treat pretty as “needs seq” even if format isn’t text (matches tests)
		return true
	}
	return false
}
