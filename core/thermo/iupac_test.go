package thermo

import "testing"

func TestExpandIUPACStrictAndDegenerate(t *testing.T) {
	got, capped, err := ExpandIUPAC("AR", 0)
	if err != nil {
		t.Fatalf("ExpandIUPAC: %v", err)
	}
	if capped || len(got) != 2 || got[0] != "AA" || got[1] != "AG" {
		t.Fatalf("unexpected expansion: got=%v capped=%v", got, capped)
	}
}

func TestExpandIUPACCap(t *testing.T) {
	got, capped, err := ExpandIUPAC("NNN", 5)
	if err != nil {
		t.Fatalf("ExpandIUPAC: %v", err)
	}
	if !capped || len(got) != 5 {
		t.Fatalf("expected capped 5 expansions, got len=%d capped=%v", len(got), capped)
	}
}

func TestParseIUPACThermoPolicyDefaultWorst(t *testing.T) {
	got, err := ParseIUPACThermoPolicy("")
	if err != nil {
		t.Fatalf("ParseIUPACThermoPolicy: %v", err)
	}
	if got != IUPACThermoPolicyWorst {
		t.Fatalf("got %q, want %q", got, IUPACThermoPolicyWorst)
	}
}
