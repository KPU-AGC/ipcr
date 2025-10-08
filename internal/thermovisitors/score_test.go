// internal/thermovisitors/score_test.go
package thermovisitors

import (
	"math"
	"testing"

	"ipcr-core/engine"
)

// Small wrapper for tests: forward to the denom-aware DP with a constant D and ssDNA=false.
// D=200.0 matches the conservative fallback used historically.
func dpPenalty(pr, tgt string, allowGap bool) float64 {
	return alignPenaltyC_contextualD_ss(pr, tgt, allowGap, 200.0, false)
}

func TestAlignPenalty_PositionEffects(t *testing.T) {
	// Use a 10-mer so we have a real internal region (K5=3, K3=3 → internal idx 3..6).
	pr := "ACGTACGTAC"
	tgtPerfect := "TGCATGCATG" // 3'→5'

	if p := dpPenalty(pr, tgtPerfect, false); p != 0 {
		t.Fatalf("perfect align penalty = %.2f, want 0", p)
	}
	// 3' mismatch at last base (index 9)
	t3 := []byte(tgtPerfect)
	if t3[len(t3)-1] == 'A' {
		t3[len(t3)-1] = 'G'
	} else {
		t3[len(t3)-1] = 'A'
	}
	p3 := dpPenalty(pr, string(t3), false)

	// 5' mismatch at first base (index 0)
	t5 := []byte(tgtPerfect)
	if t5[0] == 'T' {
		t5[0] = 'A'
	} else {
		t5[0] = 'T'
	}
	p5 := dpPenalty(pr, string(t5), false)

	// Internal mismatch at index 4
	ti := []byte(tgtPerfect)
	switch ti[4] {
	case 'A':
		ti[4] = 'G'
	default:
		ti[4] = 'A'
	}
	pIn := dpPenalty(pr, string(ti), false)

	// Assert relative ordering: 3' > 5' > internal (position multiplier × chemistry)
	if !(p3 > p5 && p5 > pIn) {
		t.Fatalf("position penalties not ordered as expected: 3' %.2f, 5' %.2f, internal %.2f", p3, p5, pIn)
	}
}

func TestAlignPenalty_Chemistry_GTvsGA(t *testing.T) {
	// Primer of G's so we can toggle the target at an internal position.
	pr := "GGGGGGGGGG"
	tgtPerfect := "CCCCCCCCCC" // 3'→5'

	// Internal index = 4 → compare chemistries for that column
	tGT := []byte(tgtPerfect)
	tGT[4] = 'T' // G•T wobble (milder)
	pGT := dpPenalty(pr, string(tGT), false)

	tGA := []byte(tgtPerfect)
	tGA[4] = 'A' // G•A (harsher than GT in our table)
	pGA := dpPenalty(pr, string(tGA), false)

	if !(pGT < pGA) {
		t.Fatalf("chemistry ordering failed: expected GT(%.2f) < GA(%.2f)", pGT, pGA)
	}
}

func rc5to3(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 'A':
			b[len(s)-1-i] = 'T'
		case 'C':
			b[len(s)-1-i] = 'G'
		case 'G':
			b[len(s)-1-i] = 'C'
		case 'T':
			b[len(s)-1-i] = 'A'
		default:
			b[len(s)-1-i] = 'N'
		}
	}
	return string(b)
}

func TestScore_ImprovesWithPerfectEnds(t *testing.T) {
	fwd := "ACGTAC"
	rev := "GGTACC"
	amp := fwd + "AAAA" + rc5to3(rev) // perfect ends
	p := engine.Product{
		FwdPrimer: fwd, RevPrimer: rev,
		Seq: amp, Length: len(amp), Type: "forward",
	}

	v := Score{AnnealTempC: 60, Na_M: 0.05, PrimerConc_M: 2.5e-7, AllowIndels: true, LengthBiasOn: false}
	_, p1, _ := v.Visit(p)

	// 3' mismatch on forward end: mutate last base of left end
	ampBad := []byte(amp)
	if ampBad[len(fwd)-1] == 'A' {
		ampBad[len(fwd)-1] = 'C'
	} else {
		ampBad[len(fwd)-1] = 'A'
	}
	pBad := p
	pBad.Seq = string(ampBad)
	_, p2, _ := v.Visit(pBad)

	if !(p1.Score > p2.Score) {
		t.Fatalf("expected score to drop with 3' mismatch: perfect=%.2f, bad=%.2f", p1.Score, p2.Score)
	}
	if math.IsNaN(p1.Score) || math.IsNaN(p2.Score) {
		t.Fatalf("NaN scores")
	}
}
