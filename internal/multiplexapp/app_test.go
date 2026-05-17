package multiplexapp

import "testing"

func TestValidatePrimerPoolNormalizesAndDeduplicates(t *testing.T) {
	got, err := validatePrimerPool("--forward", []string{" acg ", "ACG", "try"})
	if err != nil {
		t.Fatalf("validatePrimerPool returned error: %v", err)
	}
	want := []string{"ACG", "TRY"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestValidatePrimerPoolRejectsInvalidBase(t *testing.T) {
	if _, err := validatePrimerPool("--reverse", []string{"ACGX"}); err == nil {
		t.Fatal("expected invalid primer error")
	}
}
