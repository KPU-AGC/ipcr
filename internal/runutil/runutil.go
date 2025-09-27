// internal/runutil/runutil.go
package runutil

// ComputeTerminalWindow returns the effective terminal 3' window
// given the CLI mode and an override value. If terminalWindow >= 0,
// that value is used as-is. Otherwise: realistic=3, everything else=0.
func ComputeTerminalWindow(mode string, terminalWindow int) int {
	if terminalWindow >= 0 {
		return terminalWindow
	}
	if mode == "realistic" {
		return 3
	}
	return 0
}

// ComputeOverlap chooses a safe chunk overlap. If maxLen > 0, overlap must be
// at least maxLen to ensure a product straddling a boundary is seen. We also
// ensure overlap >= (maxPrimerLen - 1) so a primer site cannot be split.
func ComputeOverlap(maxLen, maxPrimerLen int) int {
	ov := 0
	if maxLen > 0 {
		ov = maxLen
	}
	if mpl := maxPrimerLen - 1; mpl > ov {
		ov = mpl
	}
	return ov
}

// ValidateChunking decides whether chunking is allowed, returns (chunkSize, overlap, warnings).
// Rules (matching current behavior):
//  • --circular disables chunking (ignore --chunk-size)
//  • If --chunk-size <= 0 → no chunking
//  • If --max-length <= 0 → disable chunking (needs max-length)
//  • If --chunk-size <= --max-length → disable chunking (must be larger)
// When enabled, overlap is ComputeOverlap(maxLen, maxPrimerLen).
func ValidateChunking(circular bool, chunkSize, maxLen, maxPrimerLen int) (int, int, []string) {
	var warns []string
	if circular {
		if chunkSize != 0 {
			warns = append(warns, "warning: --circular disables chunking; ignoring --chunk-size")
		}
		return 0, 0, warns
	}
	if chunkSize <= 0 {
		return 0, 0, nil
	}
	if maxLen <= 0 {
		warns = append(warns, "warning: --chunk-size requires --max-length; disabling")
		return 0, 0, warns
	}
	if chunkSize <= maxLen {
		warns = append(warns, "warning: --chunk-size must be > --max-length; disabling")
		return 0, 0, warns
	}
	return chunkSize, ComputeOverlap(maxLen, maxPrimerLen), nil
}

// ComputeNeedSeq tells the pipeline whether to populate Product.Seq.
// For ipcr: we need sequences for --products, for pretty text, and for FASTA.
func ComputeNeedSeq(output string, products, pretty bool) bool {
	if products {
		return true
	}
	if output == "fasta" {
		return true
	}
	if output == "text" && pretty {
		return true
	}
	return false
}
