package primer

import "testing"

func TestValidateNormalizesLowercaseIUPAC(t *testing.T) {
	got, err := Validate(" acgtry swkmbdhvn ")
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if got != "ACGTRYSWKMBDHVN" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateRejectsInvalidBase(t *testing.T) {
	if _, err := Validate("ACGX"); err == nil {
		t.Fatal("expected invalid base error")
	}
}

func TestValidateRejectsUAtInputBoundary(t *testing.T) {
	if _, err := Validate("ACGU"); err == nil {
		t.Fatal("expected U to be rejected as non-DNA input")
	}
}
