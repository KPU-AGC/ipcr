package thermomodel

import "testing"

func TestParseDefaultsToLegacyHeuristic(t *testing.T) {
	got, err := Parse("")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if got != LegacyHeuristic {
		t.Fatalf("got %q, want %q", got, LegacyHeuristic)
	}
}

func TestParseKnownModes(t *testing.T) {
	for _, mode := range Known() {
		got, err := Parse(mode.String())
		if err != nil {
			t.Fatalf("Parse(%q): %v", mode, err)
		}
		if got != mode {
			t.Fatalf("Parse(%q) = %q", mode, got)
		}
	}
}

func TestImplementedModes(t *testing.T) {
	for _, mode := range []Mode{LegacyHeuristic, NNDuplexV1} {
		if !mode.Implemented() {
			t.Fatalf("%q should be implemented", mode)
		}
	}
	if NNStructureV1.Implemented() {
		t.Fatalf("%q should still be reserved, not implemented", NNStructureV1)
	}
}

func TestParseRejectsUnknownMode(t *testing.T) {
	if _, err := Parse("bogus"); err == nil {
		t.Fatal("expected unknown mode error")
	}
}
