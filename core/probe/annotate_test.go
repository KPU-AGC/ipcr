package probe

import "testing"

func TestAnnotateAmplicon(t *testing.T) {
	amp := "ACGTACGTACGT"
	probe := "GTAC" // occurs at pos 2 and 6 and 10

	a := AnnotateAmplicon(amp, probe, 0)
	if !a.Found || a.MM != 0 || a.Pos != 2 || a.Site != "GTAC" {
		t.Fatalf("unexpected: %+v", a)
	}

	// Reverse complement should also match
	a2 := AnnotateAmplicon(amp, "GTGC", 1) // RC is GCAC; allow 1 mm
	if !a2.Found {
		t.Fatalf("expected rc match with mismatches")
	}
}
