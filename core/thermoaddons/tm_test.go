package thermoaddons

import (
	"math"
	"testing"

	"ipcr-core/thermo"
)

func TestTmNearestNeighborDelegatesToCanonicalThermo(t *testing.T) {
	primer := "GCGCGATATCGC"
	cond := Conditions{NaM: 0.05, PrimerTotalM: 2.5e-7, SaltModel: thermo.SaltModelMonovalent}
	gotTm, gotDH, gotDS, err := TmNearestNeighbor(primer, cond)
	if err != nil {
		t.Fatalf("TmNearestNeighbor: %v", err)
	}
	target, _ := complement3to5(primer)
	want, err := thermo.Tm(primer, target, cond.WithDefaults().TmInput())
	if err != nil {
		t.Fatalf("thermo.Tm: %v", err)
	}
	if math.Abs(gotTm-want.TmC) > 1e-9 || math.Abs(gotDH-want.DH_kcal) > 1e-9 || math.Abs(gotDS-want.DS_Na) > 1e-9 {
		t.Fatalf("got Tm/DH/DS %.12g %.12g %.12g, want %.12g %.12g %.12g", gotTm, gotDH, gotDS, want.TmC, want.DH_kcal, want.DS_Na)
	}
	if gotTm > 120 || gotTm < -20 {
		t.Fatalf("TmNearestNeighbor returned implausible Celsius value: %g", gotTm)
	}
}
