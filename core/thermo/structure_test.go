package thermo

import "testing"

func TestBestHairpinFindsNearestNeighborStem(t *testing.T) {
	got, ok, err := BestHairpin("GCGCGCAAAAGCGCGC", DefaultStructureOptions(DefaultConditions()))
	if err != nil {
		t.Fatalf("BestHairpin error: %v", err)
	}
	if !ok {
		t.Fatal("expected hairpin")
	}
	if got.Kind != StructureHairpin || got.StemLen < 4 || got.LoopLen < 3 {
		t.Fatalf("unexpected hairpin result: %+v", got)
	}
}

func TestBestHairpinRejectsNoStemSequence(t *testing.T) {
	got, ok, err := BestHairpin("AAAAAAAAAAAA", DefaultStructureOptions(DefaultConditions()))
	if err != nil {
		t.Fatalf("BestHairpin error: %v", err)
	}
	if ok {
		t.Fatalf("unexpected hairpin: %+v", got)
	}
}

func TestBestSelfDimerReportsThreePrimeAnchoring(t *testing.T) {
	got, ok, err := BestSelfDimer("AAAAAGCGC", DefaultStructureOptions(DefaultConditions()))
	if err != nil {
		t.Fatalf("BestSelfDimer error: %v", err)
	}
	if !ok {
		t.Fatal("expected self-dimer")
	}
	if got.Kind != StructureSelfDimer || got.StemLen < 4 {
		t.Fatalf("unexpected self-dimer: %+v", got)
	}
	if !got.ThreePrimeAnchored {
		t.Fatalf("expected 3' anchoring, got %+v", got)
	}
}

func TestBestCrossDimerIsSymmetricInEnergy(t *testing.T) {
	opts := DefaultStructureOptions(DefaultConditions())
	ab, okAB, errAB := BestCrossDimer("TTTTTGCGC", "AAAAAGCGC", opts)
	ba, okBA, errBA := BestCrossDimer("AAAAAGCGC", "TTTTTGCGC", opts)
	if errAB != nil || errBA != nil {
		t.Fatalf("unexpected errors: %v %v", errAB, errBA)
	}
	if !okAB || !okBA {
		t.Fatalf("expected both orientations to find dimers: ab=%+v ba=%+v", ab, ba)
	}
	if diff := ab.DeltaGAtAnnealKcal - ba.DeltaGAtAnnealKcal; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("expected symmetric dimer energy, got %g vs %g", ab.DeltaGAtAnnealKcal, ba.DeltaGAtAnnealKcal)
	}
}
