package thermo

import "math"

const MismatchParameterSetSantaLuciaHicks2004CompiledDimerGaugeV1 = "santalucia-hicks-2004-internal-mismatch-compiled-dimer-gauge-v1"

type MismatchTripletParameter struct {
	Key               MismatchKey
	DeltaHkcal        float64
	DeltaScalK        float64
	DeltaG37kcal      float64
	PerfectRefG37kcal float64
	DeltaDeltaG37kcal float64
	Source            MismatchLookupSource
	ParameterSet      string
	Citation          string
	Note              string
}

const santaLuciaHicksInternalMismatchCompiledGaugeCitation = "SantaLucia & Hicks 2004 Table 2 + unified Watson-Crick Table 1; primary mismatch papers: Allawi & SantaLucia 1997/1998 and Peyret et al. 1999"

func santaLuciaHicksCompiledGaugeTriplet(p5, p, p3, t5, t, t3 byte, g37, refG37, ddg37 float64, note string) MismatchTripletParameter {
	return MismatchTripletParameter{
		Key:               MismatchKey{P5: p5, P: p, P3: p3, T5: t5, T: t, T3: t3},
		DeltaHkcal:        math.NaN(),
		DeltaScalK:        math.NaN(),
		DeltaG37kcal:      g37,
		PerfectRefG37kcal: refG37,
		DeltaDeltaG37kcal: ddg37,
		Source:            MismatchSourceTripletDeltaG,
		ParameterSet:      MismatchParameterSetSantaLuciaHicks2004CompiledDimerGaugeV1,
		Citation:          santaLuciaHicksInternalMismatchCompiledGaugeCitation,
		Note:              "1 M NaCl, 37 °C; isolated internal " + note + ".",
	}
}

// CuratedMismatchTriplets contains isolated internal single-base DNA/DNA
// mismatch triplet penalties. Primer is 5'→3'; target is primer-aligned 3'→5'.
//
// This table expands SantaLucia-Hicks 2004 Table 2 to all 192 oriented
// non-Watson-Crick internal single-mismatch triplets by summing the two
// adjacent compiled-dimer mismatch increments and subtracting the matched
// Watson-Crick local reference from Table 1. These are local ΔΔG°37 scoring
// penalties, not unique measured physical trimers for every mismatch family.
var CuratedMismatchTriplets = []MismatchTripletParameter{
	// A/A mismatches.
	santaLuciaHicksCompiledGaugeTriplet('A', 'A', 'A', 'T', 'A', 'T', 1.30, -2.00, 3.30, "A/A in A/T and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'A', 'C', 'T', 'A', 'G', 0.78, -2.44, 3.22, "A/A in A/T and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'A', 'G', 'T', 'A', 'C', 1.04, -2.28, 3.32, "A/A in A/T and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'A', 'T', 'T', 'A', 'A', 1.22, -1.88, 3.10, "A/A in A/T and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'A', 'A', 'G', 'A', 'T', 1.12, -2.45, 3.57, "A/A in C/G and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'A', 'C', 'G', 'A', 'G', 0.60, -2.89, 3.49, "A/A in C/G and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'A', 'G', 'G', 'A', 'C', 0.86, -2.73, 3.59, "A/A in C/G and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'A', 'T', 'G', 'A', 'A', 1.04, -2.33, 3.37, "A/A in C/G and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'A', 'A', 'C', 'A', 'T', 0.86, -2.30, 3.16, "A/A in G/C and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'A', 'C', 'C', 'A', 'G', 0.34, -2.74, 3.08, "A/A in G/C and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'A', 'G', 'C', 'A', 'C', 0.60, -2.58, 3.18, "A/A in G/C and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'A', 'T', 'C', 'A', 'A', 0.78, -2.18, 2.96, "A/A in G/C and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'A', 'A', 'A', 'A', 'T', 1.38, -1.58, 2.96, "A/A in T/A and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'A', 'C', 'A', 'A', 'G', 0.86, -2.02, 2.88, "A/A in T/A and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'A', 'G', 'A', 'A', 'C', 1.12, -1.86, 2.98, "A/A in T/A and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'A', 'T', 'A', 'A', 'A', 1.30, -1.46, 2.76, "A/A in T/A and T/A flanks"),

	// A/C mismatches.
	santaLuciaHicksCompiledGaugeTriplet('A', 'A', 'A', 'T', 'C', 'T', 2.21, -2.00, 4.21, "A/C in A/T and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'A', 'C', 'T', 'C', 'G', 1.35, -2.44, 3.79, "A/C in A/T and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'A', 'G', 'T', 'C', 'C', 1.67, -2.28, 3.95, "A/C in A/T and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'A', 'T', 'T', 'C', 'A', 1.65, -1.88, 3.53, "A/C in A/T and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'A', 'A', 'G', 'C', 'T', 2.08, -2.45, 4.53, "A/C in C/G and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'A', 'C', 'G', 'C', 'G', 1.22, -2.89, 4.11, "A/C in C/G and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'A', 'G', 'G', 'C', 'C', 1.54, -2.73, 4.27, "A/C in C/G and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'A', 'T', 'G', 'C', 'A', 1.52, -2.33, 3.85, "A/C in C/G and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'A', 'A', 'C', 'C', 'T', 2.14, -2.30, 4.44, "A/C in G/C and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'A', 'C', 'C', 'C', 'G', 1.28, -2.74, 4.02, "A/C in G/C and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'A', 'G', 'C', 'C', 'C', 1.60, -2.58, 4.18, "A/C in G/C and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'A', 'T', 'C', 'C', 'A', 1.58, -2.18, 3.76, "A/C in G/C and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'A', 'A', 'A', 'C', 'T', 2.25, -1.58, 3.83, "A/C in T/A and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'A', 'C', 'A', 'C', 'G', 1.39, -2.02, 3.41, "A/C in T/A and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'A', 'G', 'A', 'C', 'C', 1.71, -1.86, 3.57, "A/C in T/A and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'A', 'T', 'A', 'C', 'A', 1.69, -1.46, 3.15, "A/C in T/A and T/A flanks"),

	// A/G mismatches.
	santaLuciaHicksCompiledGaugeTriplet('A', 'A', 'A', 'T', 'G', 'T', 0.88, -2.00, 2.88, "A/G in A/T and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'A', 'C', 'T', 'G', 'G', -0.38, -2.44, 2.06, "A/G in A/T and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'A', 'G', 'T', 'G', 'C', 0.25, -2.28, 2.53, "A/G in A/T and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'A', 'T', 'T', 'G', 'A', 0.16, -1.88, 2.04, "A/G in A/T and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'A', 'A', 'G', 'G', 'T', 0.77, -2.45, 3.22, "A/G in C/G and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'A', 'C', 'G', 'G', 'G', -0.49, -2.89, 2.40, "A/G in C/G and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'A', 'G', 'G', 'G', 'C', 0.14, -2.73, 2.87, "A/G in C/G and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'A', 'T', 'G', 'G', 'A', 0.05, -2.33, 2.38, "A/G in C/G and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'A', 'A', 'C', 'G', 'T', 0.49, -2.30, 2.79, "A/G in G/C and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'A', 'C', 'C', 'G', 'G', -0.77, -2.74, 1.97, "A/G in G/C and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'A', 'G', 'C', 'G', 'C', -0.14, -2.58, 2.44, "A/G in G/C and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'A', 'T', 'C', 'G', 'A', -0.23, -2.18, 1.95, "A/G in G/C and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'A', 'A', 'A', 'G', 'T', 1.16, -1.58, 2.74, "A/G in T/A and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'A', 'C', 'A', 'G', 'G', -0.10, -2.02, 1.92, "A/G in T/A and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'A', 'G', 'A', 'G', 'C', 0.53, -1.86, 2.39, "A/G in T/A and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'A', 'T', 'A', 'G', 'A', 0.44, -1.46, 1.90, "A/G in T/A and T/A flanks"),

	// C/A mismatches.
	santaLuciaHicksCompiledGaugeTriplet('A', 'C', 'A', 'T', 'A', 'T', 1.69, -2.89, 4.58, "C/A in A/T and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'C', 'C', 'T', 'A', 'G', 1.58, -3.28, 4.86, "C/A in A/T and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'C', 'G', 'T', 'A', 'C', 1.52, -3.61, 5.13, "C/A in A/T and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'C', 'T', 'T', 'A', 'A', 1.65, -2.72, 4.37, "C/A in A/T and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'C', 'A', 'G', 'A', 'T', 1.71, -3.29, 5.00, "C/A in C/G and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'C', 'C', 'G', 'A', 'G', 1.60, -3.68, 5.28, "C/A in C/G and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'C', 'G', 'G', 'A', 'C', 1.54, -4.01, 5.55, "C/A in C/G and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'C', 'T', 'G', 'A', 'A', 1.67, -3.12, 4.79, "C/A in C/G and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'C', 'A', 'C', 'A', 'T', 1.39, -3.69, 5.08, "C/A in G/C and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'C', 'C', 'C', 'A', 'G', 1.28, -4.08, 5.36, "C/A in G/C and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'C', 'G', 'C', 'A', 'C', 1.22, -4.41, 5.63, "C/A in G/C and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'C', 'T', 'C', 'A', 'A', 1.35, -3.52, 4.87, "C/A in G/C and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'C', 'A', 'A', 'A', 'T', 2.25, -2.75, 5.00, "C/A in T/A and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'C', 'C', 'A', 'A', 'G', 2.14, -3.14, 5.28, "C/A in T/A and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'C', 'G', 'A', 'A', 'C', 2.08, -3.47, 5.55, "C/A in T/A and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'C', 'T', 'A', 'A', 'A', 2.21, -2.58, 4.79, "C/A in T/A and T/A flanks"),

	// C/C mismatches.
	santaLuciaHicksCompiledGaugeTriplet('A', 'C', 'A', 'T', 'C', 'T', 2.38, -2.89, 5.27, "C/C in A/T and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'C', 'C', 'T', 'C', 'G', 2.12, -3.28, 5.40, "C/C in A/T and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'C', 'G', 'T', 'C', 'C', 2.03, -3.61, 5.64, "C/C in A/T and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'C', 'T', 'T', 'C', 'A', 2.66, -2.72, 5.38, "C/C in A/T and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'C', 'A', 'G', 'C', 'T', 1.75, -3.29, 5.04, "C/C in C/G and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'C', 'C', 'G', 'C', 'G', 1.49, -3.68, 5.17, "C/C in C/G and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'C', 'G', 'G', 'C', 'C', 1.40, -4.01, 5.41, "C/C in C/G and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'C', 'T', 'G', 'C', 'A', 2.03, -3.12, 5.15, "C/C in C/G and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'C', 'A', 'C', 'C', 'T', 1.84, -3.69, 5.53, "C/C in G/C and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'C', 'C', 'C', 'C', 'G', 1.58, -4.08, 5.66, "C/C in G/C and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'C', 'G', 'C', 'C', 'C', 1.49, -4.41, 5.90, "C/C in G/C and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'C', 'T', 'C', 'C', 'A', 2.12, -3.52, 5.64, "C/C in G/C and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'C', 'A', 'A', 'C', 'T', 2.10, -2.75, 4.85, "C/C in T/A and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'C', 'C', 'A', 'C', 'G', 1.84, -3.14, 4.98, "C/C in T/A and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'C', 'G', 'A', 'C', 'C', 1.75, -3.47, 5.22, "C/C in T/A and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'C', 'T', 'A', 'C', 'A', 2.38, -2.58, 4.96, "C/C in T/A and T/A flanks"),

	// C/T mismatches.
	santaLuciaHicksCompiledGaugeTriplet('A', 'C', 'A', 'T', 'T', 'T', 1.39, -2.89, 4.28, "C/T in A/T and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'C', 'C', 'T', 'T', 'G', 1.62, -3.28, 4.90, "C/T in A/T and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'C', 'G', 'T', 'T', 'C', 1.04, -3.61, 4.65, "C/T in A/T and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'C', 'T', 'T', 'T', 'A', 1.37, -2.72, 4.09, "C/T in A/T and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'C', 'A', 'G', 'T', 'T', 1.37, -3.29, 4.66, "C/T in C/G and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'C', 'C', 'G', 'T', 'G', 1.60, -3.68, 5.28, "C/T in C/G and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'C', 'G', 'G', 'T', 'C', 1.02, -4.01, 5.03, "C/T in C/G and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'C', 'T', 'G', 'T', 'A', 1.35, -3.12, 4.47, "C/T in C/G and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'C', 'A', 'C', 'T', 'T', 1.37, -3.69, 5.06, "C/T in G/C and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'C', 'C', 'C', 'T', 'G', 1.60, -4.08, 5.68, "C/T in G/C and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'C', 'G', 'C', 'T', 'C', 1.02, -4.41, 5.43, "C/T in G/C and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'C', 'T', 'C', 'T', 'A', 1.35, -3.52, 4.87, "C/T in G/C and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'C', 'A', 'A', 'T', 'T', 1.72, -2.75, 4.47, "C/T in T/A and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'C', 'C', 'A', 'T', 'G', 1.95, -3.14, 5.09, "C/T in T/A and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'C', 'G', 'A', 'T', 'C', 1.37, -3.47, 4.84, "C/T in T/A and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'C', 'T', 'A', 'T', 'A', 1.70, -2.58, 4.28, "C/T in T/A and T/A flanks"),

	// G/A mismatches.
	santaLuciaHicksCompiledGaugeTriplet('A', 'G', 'A', 'T', 'A', 'T', 0.44, -2.58, 3.02, "G/A in A/T and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'G', 'C', 'T', 'A', 'G', -0.23, -3.52, 3.29, "G/A in A/T and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'G', 'G', 'T', 'A', 'C', 0.05, -3.12, 3.17, "G/A in A/T and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'G', 'T', 'T', 'A', 'A', 0.16, -2.72, 2.88, "G/A in A/T and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'G', 'A', 'G', 'A', 'T', 0.53, -3.47, 4.00, "G/A in C/G and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'G', 'C', 'G', 'A', 'G', -0.14, -4.41, 4.27, "G/A in C/G and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'G', 'G', 'G', 'A', 'C', 0.14, -4.01, 4.15, "G/A in C/G and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'G', 'T', 'G', 'A', 'A', 0.25, -3.61, 3.86, "G/A in C/G and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'G', 'A', 'C', 'A', 'T', -0.10, -3.14, 3.04, "G/A in G/C and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'G', 'C', 'C', 'A', 'G', -0.77, -4.08, 3.31, "G/A in G/C and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'G', 'G', 'C', 'A', 'C', -0.49, -3.68, 3.19, "G/A in G/C and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'G', 'T', 'C', 'A', 'A', -0.38, -3.28, 2.90, "G/A in G/C and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'G', 'A', 'A', 'A', 'T', 1.16, -2.75, 3.91, "G/A in T/A and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'G', 'C', 'A', 'A', 'G', 0.49, -3.69, 4.18, "G/A in T/A and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'G', 'G', 'A', 'A', 'C', 0.77, -3.29, 4.06, "G/A in T/A and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'G', 'T', 'A', 'A', 'A', 0.88, -2.89, 3.77, "G/A in T/A and T/A flanks"),

	// G/G mismatches.
	santaLuciaHicksCompiledGaugeTriplet('A', 'G', 'A', 'T', 'G', 'T', 0.31, -2.58, 2.89, "G/G in A/T and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'G', 'C', 'T', 'G', 'G', -1.24, -3.52, 2.28, "G/G in A/T and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'G', 'G', 'T', 'G', 'C', -0.24, -3.12, 2.88, "G/G in A/T and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'G', 'T', 'T', 'G', 'A', -0.26, -2.72, 2.46, "G/G in A/T and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'G', 'A', 'G', 'G', 'T', 0.33, -3.47, 3.80, "G/G in C/G and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'G', 'C', 'G', 'G', 'G', -1.22, -4.41, 3.19, "G/G in C/G and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'G', 'G', 'G', 'G', 'C', -0.22, -4.01, 3.79, "G/G in C/G and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'G', 'T', 'G', 'G', 'A', -0.24, -3.61, 3.37, "G/G in C/G and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'G', 'A', 'C', 'G', 'T', -0.67, -3.14, 2.47, "G/G in G/C and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'G', 'C', 'C', 'G', 'G', -2.22, -4.08, 1.86, "G/G in G/C and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'G', 'G', 'C', 'G', 'C', -1.22, -3.68, 2.46, "G/G in G/C and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'G', 'T', 'C', 'G', 'A', -1.24, -3.28, 2.04, "G/G in G/C and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'G', 'A', 'A', 'G', 'T', 0.88, -2.75, 3.63, "G/G in T/A and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'G', 'C', 'A', 'G', 'G', -0.67, -3.69, 3.02, "G/G in T/A and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'G', 'G', 'A', 'G', 'C', 0.33, -3.29, 3.62, "G/G in T/A and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'G', 'T', 'A', 'G', 'A', 0.31, -2.89, 3.20, "G/G in T/A and T/A flanks"),

	// G/T mismatches.
	santaLuciaHicksCompiledGaugeTriplet('A', 'G', 'A', 'T', 'T', 'T', 1.05, -2.58, 3.63, "G/T in A/T and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'G', 'C', 'T', 'T', 'G', 0.12, -3.52, 3.64, "G/T in A/T and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'G', 'G', 'T', 'T', 'C', 0.39, -3.12, 3.51, "G/T in A/T and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'G', 'T', 'T', 'T', 'A', 0.78, -2.72, 3.50, "G/T in A/T and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'G', 'A', 'G', 'T', 'T', -0.13, -3.47, 3.34, "G/T in C/G and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'G', 'C', 'G', 'T', 'G', -1.06, -4.41, 3.35, "G/T in C/G and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'G', 'G', 'G', 'T', 'C', -0.79, -4.01, 3.22, "G/T in C/G and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'G', 'T', 'G', 'T', 'A', -0.40, -3.61, 3.21, "G/T in C/G and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'G', 'A', 'C', 'T', 'T', 0.42, -3.14, 3.56, "G/T in G/C and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'G', 'C', 'C', 'T', 'G', -0.51, -4.08, 3.57, "G/T in G/C and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'G', 'G', 'C', 'T', 'C', -0.24, -3.68, 3.44, "G/T in G/C and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'G', 'T', 'C', 'T', 'A', 0.15, -3.28, 3.43, "G/T in G/C and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'G', 'A', 'A', 'T', 'T', 0.77, -2.75, 3.52, "G/T in T/A and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'G', 'C', 'A', 'T', 'G', -0.16, -3.69, 3.53, "G/T in T/A and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'G', 'G', 'A', 'T', 'C', 0.11, -3.29, 3.40, "G/T in T/A and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'G', 'T', 'A', 'T', 'A', 0.50, -2.89, 3.39, "G/T in T/A and T/A flanks"),

	// T/C mismatches.
	santaLuciaHicksCompiledGaugeTriplet('A', 'T', 'A', 'T', 'C', 'T', 1.70, -1.46, 3.16, "T/C in A/T and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'T', 'C', 'T', 'C', 'G', 1.35, -2.18, 3.53, "T/C in A/T and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'T', 'G', 'T', 'C', 'C', 1.35, -2.33, 3.68, "T/C in A/T and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'T', 'T', 'T', 'C', 'A', 1.37, -1.88, 3.25, "T/C in A/T and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'T', 'A', 'G', 'C', 'T', 1.37, -1.86, 3.23, "T/C in C/G and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'T', 'C', 'G', 'C', 'G', 1.02, -2.58, 3.60, "T/C in C/G and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'T', 'G', 'G', 'C', 'C', 1.02, -2.73, 3.75, "T/C in C/G and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'T', 'T', 'G', 'C', 'A', 1.04, -2.28, 3.32, "T/C in C/G and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'T', 'A', 'C', 'C', 'T', 1.95, -2.02, 3.97, "T/C in G/C and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'T', 'C', 'C', 'C', 'G', 1.60, -2.74, 4.34, "T/C in G/C and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'T', 'G', 'C', 'C', 'C', 1.60, -2.89, 4.49, "T/C in G/C and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'T', 'T', 'C', 'C', 'A', 1.62, -2.44, 4.06, "T/C in G/C and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'T', 'A', 'A', 'C', 'T', 1.72, -1.58, 3.30, "T/C in T/A and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'T', 'C', 'A', 'C', 'G', 1.37, -2.30, 3.67, "T/C in T/A and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'T', 'G', 'A', 'C', 'C', 1.37, -2.45, 3.82, "T/C in T/A and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'T', 'T', 'A', 'C', 'A', 1.39, -2.00, 3.39, "T/C in T/A and T/A flanks"),

	// T/G mismatches.
	santaLuciaHicksCompiledGaugeTriplet('A', 'T', 'A', 'T', 'G', 'T', 0.50, -1.46, 1.96, "T/G in A/T and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'T', 'C', 'T', 'G', 'G', 0.15, -2.18, 2.33, "T/G in A/T and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'T', 'G', 'T', 'G', 'C', -0.40, -2.33, 1.93, "T/G in A/T and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'T', 'T', 'T', 'G', 'A', 0.78, -1.88, 2.66, "T/G in A/T and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'T', 'A', 'G', 'G', 'T', 0.11, -1.86, 1.97, "T/G in C/G and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'T', 'C', 'G', 'G', 'G', -0.24, -2.58, 2.34, "T/G in C/G and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'T', 'G', 'G', 'G', 'C', -0.79, -2.73, 1.94, "T/G in C/G and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'T', 'T', 'G', 'G', 'A', 0.39, -2.28, 2.67, "T/G in C/G and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'T', 'A', 'C', 'G', 'T', -0.16, -2.02, 1.86, "T/G in G/C and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'T', 'C', 'C', 'G', 'G', -0.51, -2.74, 2.23, "T/G in G/C and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'T', 'G', 'C', 'G', 'C', -1.06, -2.89, 1.83, "T/G in G/C and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'T', 'T', 'C', 'G', 'A', 0.12, -2.44, 2.56, "T/G in G/C and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'T', 'A', 'A', 'G', 'T', 0.77, -1.58, 2.35, "T/G in T/A and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'T', 'C', 'A', 'G', 'G', 0.42, -2.30, 2.72, "T/G in T/A and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'T', 'G', 'A', 'G', 'C', -0.13, -2.45, 2.32, "T/G in T/A and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'T', 'T', 'A', 'G', 'A', 1.05, -2.00, 3.05, "T/G in T/A and T/A flanks"),

	// T/T mismatches.
	santaLuciaHicksCompiledGaugeTriplet('A', 'T', 'A', 'T', 'T', 'T', 1.37, -1.46, 2.83, "T/T in A/T and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'T', 'C', 'T', 'T', 'G', 1.14, -2.18, 3.32, "T/T in A/T and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'T', 'G', 'T', 'T', 'C', 0.57, -2.33, 2.90, "T/T in A/T and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('A', 'T', 'T', 'T', 'T', 'A', 1.38, -1.88, 3.26, "T/T in A/T and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'T', 'A', 'G', 'T', 'T', 0.56, -1.86, 2.42, "T/T in C/G and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'T', 'C', 'G', 'T', 'G', 0.33, -2.58, 2.91, "T/T in C/G and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'T', 'G', 'G', 'T', 'C', -0.24, -2.73, 2.49, "T/T in C/G and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('C', 'T', 'T', 'G', 'T', 'A', 0.57, -2.28, 2.85, "T/T in C/G and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'T', 'A', 'C', 'T', 'T', 1.13, -2.02, 3.15, "T/T in G/C and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'T', 'C', 'C', 'T', 'G', 0.90, -2.74, 3.64, "T/T in G/C and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'T', 'G', 'C', 'T', 'C', 0.33, -2.89, 3.22, "T/T in G/C and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('G', 'T', 'T', 'C', 'T', 'A', 1.14, -2.44, 3.58, "T/T in G/C and T/A flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'T', 'A', 'A', 'T', 'T', 1.36, -1.58, 2.94, "T/T in T/A and A/T flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'T', 'C', 'A', 'T', 'G', 1.13, -2.30, 3.43, "T/T in T/A and C/G flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'T', 'G', 'A', 'T', 'C', 0.56, -2.45, 3.01, "T/T in T/A and G/C flanks"),
	santaLuciaHicksCompiledGaugeTriplet('T', 'T', 'T', 'A', 'T', 'A', 1.37, -2.00, 3.37, "T/T in T/A and T/A flanks"),
}

func init() {
	for _, p := range CuratedMismatchTriplets {
		DeltaGTriplet[p.Key] = p.DeltaDeltaG37kcal
		DeltaGTripletSource[p.Key] = p.Source
		DeltaGTripletParameterSet[p.Key] = p.ParameterSet
		DeltaGTripletCitation[p.Key] = p.Citation
		DeltaGTripletNote[p.Key] = p.Note
	}
}
