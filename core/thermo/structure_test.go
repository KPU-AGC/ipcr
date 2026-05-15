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

func TestBestCrossDimerV2ReportsBulgeCandidate(t *testing.T) {
	opts := DefaultStructureOptions(DefaultConditions())
	got, ok, err := BestCrossDimerV2("GCGCAAGCGC", "GCGCGCGC", opts)
	if err != nil {
		t.Fatalf("BestCrossDimerV2 error: %v", err)
	}
	if !ok {
		t.Fatal("expected gapped cross-dimer")
	}
	if got.Model != StructureModelStemLoopV2 {
		t.Fatalf("expected v2 model, got %+v", got)
	}
	if got.BulgeCount != 1 || got.InternalLoopCount != 0 || got.SegmentCount != 2 {
		t.Fatalf("expected one bulge in two-segment structure, got %+v", got)
	}
	if got.StemLen < 8 || got.BulgePenaltyKcal <= 0 {
		t.Fatalf("unexpected gapped-stem details: %+v", got)
	}
}

func TestBestHairpinV2ReportsInternalLoopCandidate(t *testing.T) {
	opts := DefaultStructureOptions(DefaultConditions())
	got, ok, err := bestHairpinGapped("GCGCAAGCGCTTTTGCGCTTGCGC", opts)
	if err != nil {
		t.Fatalf("bestHairpinGapped error: %v", err)
	}
	if !ok {
		t.Fatal("expected gapped hairpin")
	}
	if got.Model != StructureModelStemLoopV2 {
		t.Fatalf("expected v2 model, got %+v", got)
	}
	if got.InternalLoopCount != 1 || got.SegmentCount != 2 {
		t.Fatalf("expected one internal-loop two-segment hairpin, got %+v", got)
	}
	if got.InternalLoopPenaltyKcal <= 0 || got.LoopPenaltyKcal <= 0 {
		t.Fatalf("expected loop penalties, got %+v", got)
	}
}

func TestBestHairpinV2FindsBulgedStem(t *testing.T) {
	opts := DefaultStructureOptions(DefaultConditions())
	got, ok, err := BestHairpinV2("GCGCGCAAAAGCGTCGC", opts)
	if err != nil {
		t.Fatalf("BestHairpinV2 error: %v", err)
	}
	if !ok {
		t.Fatal("expected v2 bulged hairpin candidate")
	}
	if got.Model != StructureModelStemLoopV2 || got.BulgeCount+got.InternalLoopCount == 0 {
		t.Fatalf("expected v2 gapped hairpin metadata, got %+v", got)
	}
}

func TestBestCrossDimerV2FindsBulgedDimer(t *testing.T) {
	opts := DefaultStructureOptions(DefaultConditions())
	got, ok, err := BestCrossDimerV2("TTTTTGCGCG", "AAAAAGCGTCG", opts)
	if err != nil {
		t.Fatalf("BestCrossDimerV2 error: %v", err)
	}
	if !ok {
		t.Fatal("expected v2 bulged cross-dimer candidate")
	}
	if got.Model != StructureModelStemLoopV2 || got.StemLen < opts.MinStem || got.BulgeCount+got.InternalLoopCount == 0 {
		t.Fatalf("unexpected v2 cross-dimer: %+v", got)
	}
}

func TestBestStructureV2PreservesContiguousResult(t *testing.T) {
	opts := DefaultStructureOptions(DefaultConditions())
	v1, ok1, err1 := BestCrossDimer("TTTTTGCGC", "AAAAAGCGC", opts)
	v2, ok2, err2 := BestCrossDimerV2("TTTTTGCGC", "AAAAAGCGC", opts)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v %v", err1, err2)
	}
	if !ok1 || !ok2 {
		t.Fatalf("expected both v1 and v2 candidates: v1=%+v v2=%+v", v1, v2)
	}
	if v2.DeltaGAtAnnealKcal > v1.DeltaGAtAnnealKcal+1e-9 {
		t.Fatalf("v2 should preserve or improve v1 result: v1=%+v v2=%+v", v1, v2)
	}
}
