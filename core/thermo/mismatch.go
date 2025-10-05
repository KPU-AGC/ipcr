// core/thermo/mismatch.go
package thermo

// Minimal, extensible mismatch chemistry support.
//
// We expose both a ΔTm (°C) lookup and a ΔΔG (kcal/mol) lookup.
// If ΔTm for a given context is missing, callers can fall back to
// ΔΔG and convert to ΔTm via ΔTm ≈ (ΔΔG * 1000) / D, where
// D ≈ ΔS_Na + R ln(CT/x) (effective denominator from the Tm formula).

// Triplet context (uppercased A/C/G/T, 'N' allowed on target side).
type MismatchKey struct {
	P5, P, P3 byte // primer context
	T5, T, T3 byte // target context (3'→5')
}

// Optional: triplet-level ΔTm (°C) overrides (empty for now; fill as you curate).
var DeltaTmTriplet = map[MismatchKey]float64{}

// Pair-only ΔTm (°C) fallback (primer base vs target base).
// Lower = milder penalty (e.g., G·T wobble), higher = harsher.
var pairDeltaTm = map[[2]byte]float64{
	// Wobbles (milder)
	{'G', 'T'}: 2.5, {'T', 'G'}: 2.5,
	// Transitions (moderate)
	{'A', 'G'}: 3.5, {'G', 'A'}: 3.5,
	{'C', 'T'}: 3.5, {'T', 'C'}: 3.5,
	// Harsher pairs (catch-all)
	{'A', 'C'}: 4.0, {'C', 'A'}: 4.0,
	{'A', 'A'}: 5.0, {'C', 'C'}: 5.0, {'G', 'G'}: 5.0, {'T', 'T'}: 5.0,
	{'A', 'T'}: 4.5, {'T', 'A'}: 4.5,
	{'C', 'G'}: 4.5, {'G', 'C'}: 4.5,
}

// Optional: triplet-level ΔΔG (kcal/mol) table (empty for now; fill as curated).
var DeltaGTriplet = map[MismatchKey]float64{}

// Pair-only ΔΔG (kcal/mol) fallback.
// Scale roughly to preserve the ΔTm ordering above when divided by typical denominators.
var pairDeltaG = map[[2]byte]float64{
	// Wobbles (milder)
	{'G', 'T'}: 0.60, {'T', 'G'}: 0.60,
	// Transitions (moderate)
	{'A', 'G'}: 0.85, {'G', 'A'}: 0.85,
	{'C', 'T'}: 0.85, {'T', 'C'}: 0.85,
	// Harsher pairs
	{'A', 'C'}: 1.10, {'C', 'A'}: 1.10,
	{'A', 'A'}: 1.40, {'C', 'C'}: 1.40, {'G', 'G'}: 1.40, {'T', 'T'}: 1.40,
	{'A', 'T'}: 1.20, {'T', 'A'}: 1.20,
	{'C', 'G'}: 1.20, {'G', 'C'}: 1.20,
}

// LookupDeltaTm returns ΔTm (°C) for a mismatch with local context.
// Returns (penalty, true) if P and T are valid; otherwise false.
func LookupDeltaTm(p5, p, p3, t5, t, t3 byte) (float64, bool) {
	if !isACGT(p) || !isNT(t) {
		return 0, false
	}
	// 1) exact triplet override if available
	if isACGT(p5) && isACGT(p3) && isNT(t5) && isNT(t3) {
		if d, ok := DeltaTmTriplet[MismatchKey{P5: p5, P: p, P3: p3, T5: t5, T: t, T3: t3}]; ok {
			return d, true
		}
	}
	// 2) pair-only fallback
	key := [2]byte{p, t}
	d, ok := pairDeltaTm[key]
	if !ok {
		d = 4.0 // conservative default
	}
	// neighbor tweak (GC-rich slightly stabilizes, AT-rich slightly destabilizes)
	gc := countGC(p5) + countGC(p3) + countGC(t5) + countGC(t3)
	at := countAT(p5) + countAT(p3) + countAT(t5) + countAT(t3)
	adj := 0.0
	if gc >= at+2 {
		adj = -0.3
	} else if at >= gc+2 {
		adj = +0.3
	}
	pen := d + adj
	if pen < 0.5 {
		pen = 0.5
	}
	return pen, true
}

// LookupDeltaG returns ΔΔG (kcal/mol) using the same precedence:
// 1) triplet context if available, else 2) pair-only fallback.
// Returns (deltaG, true) if applicable; else false.
func LookupDeltaG(p5, p, p3, t5, t, t3 byte) (float64, bool) {
	if !isACGT(p) || !isNT(t) {
		return 0, false
	}
	if isACGT(p5) && isACGT(p3) && isNT(t5) && isNT(t3) {
		if dg, ok := DeltaGTriplet[MismatchKey{P5: p5, P: p, P3: p3, T5: t5, T: t, T3: t3}]; ok {
			return dg, true
		}
	}
	key := [2]byte{p, t}
	dg, ok := pairDeltaG[key]
	if !ok {
		dg = 1.0 // conservative default
	}
	// neighbor tweak to mirror ΔTm's shape (small ±0.05 kcal/mol)
	gc := countGC(p5) + countGC(p3) + countGC(t5) + countGC(t3)
	at := countAT(p5) + countAT(p3) + countAT(t5) + countAT(t3)
	if gc >= at+2 {
		dg -= 0.05
	} else if at >= gc+2 {
		dg += 0.05
	}
	if dg < 0.05 {
		dg = 0.05
	}
	return dg, true
}

// DeltaGToDeltaTm converts ΔΔG (kcal/mol) to ΔTm (°C) using an effective denominator D (cal/K/mol):
//   ΔTm ≈ (ΔΔG * 1000) / D
func DeltaGToDeltaTm(deltaG_kcal, denom float64) float64 {
	if denom <= 0 {
		return 4.0 // safe fallback if D is not available
	}
	return (deltaG_kcal * 1000.0) / denom
}

func isACGT(b byte) bool { return b == 'A' || b == 'C' || b == 'G' || b == 'T' }
func isNT(b byte) bool  { return b == 'A' || b == 'C' || b == 'G' || b == 'T' || b == 'N' }

func countGC(b byte) int {
	if b == 'G' || b == 'C' {
		return 1
	}
	return 0
}
func countAT(b byte) int {
	if b == 'A' || b == 'T' {
		return 1
	}
	return 0
}
