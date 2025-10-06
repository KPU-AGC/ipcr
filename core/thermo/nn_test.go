package thermo

import (
	"math"
	"strings"
	"testing"
)

// --- local helpers (test-only) ---------------------------------------------

// comp returns the Watson–Crick complement of s (no reverse).
func comp(s string) string {
	b := []byte(s)
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case 'A', 'a':
			b[i] = 'T'
		case 'T', 't', 'U', 'u':
			b[i] = 'A'
		case 'C', 'c':
			b[i] = 'G'
		case 'G', 'g':
			b[i] = 'C'
		default:
			b[i] = 'N'
		}
	}
	return string(b)
}

// revComp returns the reverse-complement of s.
func revComp(s string) string {
	n := len(s)
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		switch s[i] {
		case 'A', 'a':
			out[n-1-i] = 'T'
		case 'T', 't', 'U', 'u':
			out[n-1-i] = 'A'
		case 'C', 'c':
			out[n-1-i] = 'G'
		case 'G', 'g':
			out[n-1-i] = 'C'
		default:
			out[n-1-i] = 'N'
		}
	}
	return string(out)
}

// newInp builds a TmInput with typical values; tweak fields per test.
func newInp() TmInput {
	return TmInput{
		CT: 5e-7, // 0.5 µM total strand concentration
		Na: 0.05, // 50 mM monovalent
		X:  4,    // non-self (default)
	}
}

// --- tests ------------------------------------------------------------------

// Validates input checks and error messages order.
func TestTm_InputValidation(t *testing.T) {
	t.Run("empty sequences", func(t *testing.T) {
		_, err := Tm("", "", newInp())
		if err == nil || !strings.Contains(err.Error(), "sequences must be equal length and non-empty") {
			t.Fatalf("expected empty/len error, got: %v", err)
		}
	})
	t.Run("length mismatch", func(t *testing.T) {
		_, err := Tm("AA", "TTT", newInp())
		if err == nil || !strings.Contains(err.Error(), "sequences must be equal length and non-empty") {
			t.Fatalf("expected length error, got: %v", err)
		}
	})
	t.Run("CT must be > 0", func(t *testing.T) {
		in := newInp()
		in.CT = 0
		_, err := Tm("AA", "TT", in)
		if err == nil || !strings.Contains(err.Error(), "CT must be > 0") {
			t.Fatalf("expected CT>0 error, got: %v", err)
		}
	})
	t.Run("[Na+] must be > 0", func(t *testing.T) {
		in := newInp()
		in.Na = 0
		_, err := Tm("AA", "TT", in)
		if err == nil || !strings.Contains(err.Error(), "[Na+] must be > 0") {
			t.Fatalf("expected [Na+]>0 error, got: %v", err)
		}
	})
	t.Run("non-ACGT target", func(t *testing.T) {
		in := newInp()
		_, err := Tm("AA", "TN", in) // 'N' in target
		if err == nil || !strings.Contains(err.Error(), "non-ACGT base in target") {
			t.Fatalf("expected non-ACGT target error, got: %v", err)
		}
	})
	t.Run("non-WC pair at index", func(t *testing.T) {
		in := newInp()
		_, err := Tm("AA", "AA", in) // not WC
		if err == nil || !strings.Contains(err.Error(), "non-WC pair at pos") {
			t.Fatalf("expected non-WC error, got: %v", err)
		}
	})
}

// Salt monotonicity: Tm should be non-decreasing with [Na+] and
// strictly higher across a wide range.
func TestTm_MonotonicWithSalt(t *testing.T) {
	// GC-only sequence avoids NN-table edge cases and keeps focus on salt.
	primer := strings.ToUpper("GGGGCCCCGGGGCCCCGGGGCCCC")
	target3to5 := comp(primer)

	in := newInp()

	// Wide spread to avoid rounding plateaus.
	salts := []float64{1e-6, 1e-3, 1e-1, 1.0}

	const eps = 1e-9
	var last = -math.MaxFloat64
	var first float64
	for i, na := range salts {
		in.Na = na
		res, err := Tm(primer, target3to5, in)
		if err != nil {
			t.Fatalf("Tm at %g M Na+: %v", na, err)
		}
		if i == 0 {
			first = res.TmC
		}
		if res.TmC < last-eps {
			t.Fatalf("Tm should be non-decreasing with salt: got %g < %g at %g M Na+", res.TmC, last, na)
		}
		last = res.TmC
	}
	// Across the full range we expect a net increase.
	if (last - first) <= eps {
		t.Fatalf("Tm should increase across salt range: Δ=%g °C (first=%g, last=%g)", last-first, first, last)
	}
}

// Concentration monotonicity: Tm should be non-decreasing with Ct
// and strictly higher across a wide range (for non-self X=4).
func TestTm_MonotonicWithCt(t *testing.T) {
	primer := strings.ToUpper("GGGGCCCCGGGGCCCCGGGGCCCC")
	target3to5 := comp(primer)

	in := newInp()
	in.Na = 0.05 // fix salt

	cts := []float64{5e-8, 5e-7, 5e-6} // 50 nM → 0.5 µM → 5 µM

	const eps = 1e-9
	var last = -math.MaxFloat64
	var first float64
	for i, ct := range cts {
		in.CT = ct
		res, err := Tm(primer, target3to5, in)
		if err != nil {
			t.Fatalf("Tm at Ct=%g: %v", ct, err)
		}
		if i == 0 {
			first = res.TmC
		}
		if res.TmC < last-eps {
			t.Fatalf("Tm should be non-decreasing with Ct: got %g < %g at Ct=%g", res.TmC, last, ct)
		}
		last = res.TmC
	}
	if (last - first) <= eps {
		t.Fatalf("Tm should increase across Ct range: Δ=%g °C (first=%g, last=%g)", last-first, first, last)
	}
}

// Self-complement handling: for a self-complementary primer,
// X=1 should yield a (slightly) higher Tm than forcing X=4 (non-self).
func TestTm_SelfComplement_XEffect(t *testing.T) {
	primer := strings.ToUpper("ATGCAT") // self-complementary palindrome
	target3to5 := comp(primer)

	in := newInp()
	in.Na = 0.05
	in.CT = 1e-6

	in.X = 1
	resSelf, err := Tm(primer, target3to5, in)
	if err != nil {
		t.Fatalf("Tm self (X=1): %v", err)
	}

	in.X = 4
	resForcedNonSelf, err := Tm(primer, target3to5, in)
	if err != nil {
		t.Fatalf("Tm forced non-self (X=4): %v", err)
	}

	if !(resSelf.TmC > resForcedNonSelf.TmC) {
		t.Fatalf("expected self X=1 Tm > forced non-self X=4 Tm; got %g vs %g",
			resSelf.TmC, resForcedNonSelf.TmC)
	}
}

// Orientation requirement: passing reverse-complement as target (instead of 3'→5' WC)
// should fail the per-index WC check.
func TestTm_Orientation_MustBe3to5WC(t *testing.T) {
	primer := strings.ToUpper("GCGCGATATCGC")
	targetWrong := revComp(primer) // wrong: RC instead of 3'→5' WC
	in := newInp()
	_, err := Tm(primer, targetWrong, in)
	if err == nil || !strings.Contains(err.Error(), "non-WC pair at pos") {
		t.Fatalf("expected non-WC error for wrong orientation, got: %v", err)
	}
}
