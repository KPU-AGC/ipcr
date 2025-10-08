package thermovisitors

import "testing"

func TestDenomForPrimer_PositiveMagnitude(t *testing.T) {
	v := Score{Na_M: 0.05, PrimerConc_M: 2.5e-7}
	d := v.denomForPrimer("ACGTACGTAC")
	if d <= 0 {
		t.Fatalf("expected positive denom, got %g", d)
	}
	// Extremely unlikely to be exactly 200 unless we fell back.
	if d == 200.0 {
		t.Fatalf("unexpected fallback denom=200. got %g", d)
	}
	// Sanity bound (wide): should be a realistic magnitude
	if d > 2000 {
		t.Fatalf("implausibly large denom: %g", d)
	}
}

func TestAutoDenomInfluencesPenalty(t *testing.T) {
	pr := "GGGGGGGGGG"
	tgtPerfect := "CCCCCCCCCC" // 3'→5'
	// Introduce an internal mismatch so ΔΔG path is used
	tbad := []byte(tgtPerfect)
	tbad[4] = 'A' // G•A (harsher than GT)
	bad := string(tbad)

	// Fixed D path
	vFix := Score{Na_M: 0.05, PrimerConc_M: 2.5e-7}
	pFixed := vFix.Penalty(pr, bad, 200.0)

	// Auto D path (use the runtime D for this primer)
	dAuto := vFix.denomForPrimer(pr)
	pAuto := alignPenaltyC_contextualD_ss(pr, bad, false, dAuto, false)

	if dAuto == 200.0 {
		t.Fatalf("auto denom fell back to 200; dAuto=%g", dAuto)
	}
	if pAuto == pFixed {
		t.Fatalf("expected auto denom to change penalty: fixed=%.6f auto=%.6f (D=%.2f)", pFixed, pAuto, dAuto)
	}
}
