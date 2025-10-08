// core/thermo/mismatch.go
package thermo

// Context-aware mismatch chemistry support.
//
// We expose both a ΔTm (°C) lookup and a ΔΔG (kcal/mol) lookup.
// Callers should first try ΔTm (triplet overrides); if not available,
// fall back to ΔΔG and convert to ΔTm via:
//      ΔTm ≈ (ΔΔG * 1000) / D
// where D ≈ ΔS_Na + R·ln(CT/x) (effective denominator from the Tm formula).
//
// Design goals for realism (based on the literature):
//  • Some mismatches (e.g., G·T wobble) are often milder than others
//    and can even be slightly stabilizing in AT‑rich contexts.
//  • Chemistry ordering (typical): G·T < transitions (A·G/C·T) < transversions;
//    C·C is generally the harshest like‑with‑like mispair.
//  • Local sequence context matters; we include simple neighbor‑aware rules.
//  • 3' vs 5' position severity is handled by the caller (posMultiplier).
//
// Implementation notes:
//  • Triplet tables (DeltaTmTriplet/DeltaGTriplet) allow precise overrides.
//    They can also use 'N' on the TARGET side (T5/T3) to encode "neighbor
//    is a mismatch/unknown".
//  • When triplet overrides are missing, we apply a pair‑chemistry fallback
//    with lightweight context adjustments (see LookupDeltaG).
//
// Triplet context (primer and target bases are given as runes A/C/G/T; target is 3'→5').
type MismatchKey struct {
        P5, P, P3 byte // primer context
        T5, T, T3 byte // target context (3'→5')
}

// Optional: triplet-level ΔTm (°C) overrides (empty by default; fill as curated).
var DeltaTmTriplet = map[MismatchKey]float64{}

// Pair-only ΔTm (°C) fallback (primer base vs target base).
// NOTE: We no longer use this in LookupDeltaTm (so callers can fall back to ΔΔG).
// It is kept here for completeness and possible future use.
var pairDeltaTm = map[[2]byte]float64{
        // Wobbles (milder)
        {'G', 'T'}: 2.5, {'T', 'G'}: 2.5,
        // Transitions (moderate)
        {'A', 'G'}: 3.5, {'G', 'A'}: 3.5,
        {'C', 'T'}: 3.5, {'T', 'C'}: 3.5,
        // Harsher pairs (catch‑alls)
        {'A', 'C'}: 4.0, {'C', 'A'}: 4.0,
        {'A', 'A'}: 5.0, {'C', 'C'}: 5.0, {'G', 'G'}: 5.0, {'T', 'T'}: 5.0,
        {'A', 'T'}: 4.5, {'T', 'A'}: 4.5,
        {'C', 'G'}: 4.5, {'G', 'C'}: 4.5,
}

// Optional: triplet-level ΔΔG (kcal/mol) table (empty by default; fill as curated).
var DeltaGTriplet = map[MismatchKey]float64{}

// Pair-only ΔΔG (kcal/mol) baseline penalties.
// Chosen to preserve the intended ordering; further tuned by context in LookupDeltaG.
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
// It ONLY returns true when a triplet override is available. If not, return false
// so callers can fall back to ΔΔG via LookupDeltaG + DeltaGToDeltaTm.
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
        // No pair fallback — let caller use ΔΔG.
        return 0, false
}

// LookupDeltaG returns ΔΔG (kcal/mol) using precedence:
// 1) triplet context if available, else 2) pair-only with context tweaks.
// A negative value is allowed (rare stabilizing contexts).
func LookupDeltaG(p5, p, p3, t5, t, t3 byte) (float64, bool) {
        if !isACGT(p) || !isNT(t) {
                return 0, false
        }
        // 1) exact triplet override
        if isACGT(p5) && isACGT(p3) && isNT(t5) && isNT(t3) {
                if dg, ok := DeltaGTriplet[MismatchKey{P5: p5, P: p, P3: p3, T5: t5, T: t, T3: t3}]; ok {
                        return dg, true
                }
        }

        // 2) pair-only baseline
        key := [2]byte{p, t}
        base, ok := pairDeltaG[key]
        if !ok {
                base = 1.0 // conservative default
        }

        // ---- lightweight context model (heuristic) ----------------------------
        // Encode neighbor composition
        gc := countGC(p5) + countGC(p3) + countGC(t5) + countGC(t3)
        at := countAT(p5) + countAT(p3) + countAT(t5) + countAT(t3)

        // Helper: purine neighbors on primer/target sides
        purP := bIsPurine(p5) + bIsPurine(p3) // 0..2
        purT := bIsPurine(t5) + bIsPurine(t3) // 0..2

        switch key {
        case [2]byte{'G', 'T'}, [2]byte{'T', 'G'}:
                // G·T wobble: often mild; can be slightly stabilizing in AT‑rich contexts.
                if at >= 3 { // strongly AT‑rich flank
                        base -= 0.45 // allow small negative in extreme AT flanks (see cap below)
                } else if at == 2 {
                        base -= 0.20
                }
                // Do NOT penalize GT in GC‑rich flanks beyond the generic tweak below;
                // this preserves GT < GA in GC contexts (needed for stable ordering).

        case [2]byte{'G', 'A'}, [2]byte{'A', 'G'}:
                // G·A sheared pair: modestly mild if both sides present purines.
                if purP > 0 && purT > 0 {
                        base -= 0.20
                }
        case [2]byte{'G', 'G'}:
                // G·G is the most tolerable purine‑purine like‑with‑like
                base -= 0.45
        case [2]byte{'A', 'A'}, [2]byte{'T', 'T'}:
                base -= 0.25
        case [2]byte{'C', 'C'}:
                base += 0.25
        }

        // Generic neighbor tweak to mirror literature tendencies (small effect).
        if gc >= at+2 {
                base -= 0.05
        } else if at >= gc+2 {
                base += 0.00 // already accounted above for GT; keep others neutral in AT‑rich
        }

        // Rare stabilizing floor: cap at −0.10 kcal/mol (≈ −0.5 °C at D≈200).
        if base < -0.10 {
                base = -0.10
        }
        return base, true
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
func bIsPurine(b byte) int {
        if b == 'A' || b == 'G' {
                return 1
        }
        return 0
}