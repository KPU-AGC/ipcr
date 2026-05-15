package thermo

import (
	"math"
	"testing"
)

func TestImperfectDuplexPerfectMatchesPerfectDuplex(t *testing.T) {
	primer := "ACGTACGTACGTACGT"
	cond := DefaultConditions()
	perfect, err := PerfectDuplex(primer, comp(primer), cond)
	if err != nil {
		t.Fatalf("PerfectDuplex: %v", err)
	}
	got, err := ImperfectDuplex(primer, comp(primer), cond)
	if err != nil {
		t.Fatalf("ImperfectDuplex: %v", err)
	}
	if got.MismatchCount != 0 || got.MismatchPenaltyC != 0 || got.MismatchPolicy != MismatchPolicyPerfect {
		t.Fatalf("expected perfect mismatch summary, got %+v", got)
	}
	if math.Abs(got.TmC-perfect.TmC) > 1e-9 || math.Abs(got.DeltaGAtAnnealKcal-perfect.DeltaGAtAnnealKcal) > 1e-9 {
		t.Fatalf("perfect/imperfect mismatch: perfect=%+v got=%+v", perfect, got.DuplexResult)
	}
}

func TestImperfectDuplexMismatchLowersTmAndReportsFallback(t *testing.T) {
	primer := "ACGTACGTACGTACGT"
	target := []byte(comp(primer))
	target[6] = 'A'
	if target[6] == comp(primer)[6] {
		target[6] = 'C'
	}

	got, err := ImperfectDuplex(primer, string(target), DefaultConditions())
	if err != nil {
		t.Fatalf("ImperfectDuplex: %v", err)
	}
	if got.MismatchCount != 1 || !got.HasNonWatsonCrick {
		t.Fatalf("expected one mismatch, got %+v", got)
	}
	if got.MismatchPenaltyC <= 0 || got.DeltaGPenaltyKcal <= 0 {
		t.Fatalf("expected positive mismatch penalty, got %+v", got)
	}
	if got.HeuristicFallbackCount != 1 || !got.UsedHeuristicAdjust || got.MismatchPolicy != MismatchPolicyImperfectHeuristicFallback {
		t.Fatalf("expected explicit heuristic fallback metadata, got %+v", got)
	}
	base, err := PerfectDuplex(primer, comp(primer), DefaultConditions())
	if err != nil {
		t.Fatalf("PerfectDuplex: %v", err)
	}
	if !(got.TmC < base.TmC && got.AnnealMarginC < base.AnnealMarginC) {
		t.Fatalf("expected mismatch to lower Tm/margin: base=%+v got=%+v", base, got.DuplexResult)
	}
}

func TestImperfectDuplexThreePrimeMismatchIsWeightedMoreStrongly(t *testing.T) {
	primer := "GGGGGGGGGG"
	perfect := []byte(comp(primer))

	makeMismatch := func(pos int) string {
		t := append([]byte(nil), perfect...)
		t[pos] = 'A' // G·A mismatch at each position
		return string(t)
	}
	internal, err := ImperfectDuplex(primer, makeMismatch(4), DefaultConditions())
	if err != nil {
		t.Fatalf("internal mismatch: %v", err)
	}
	five, err := ImperfectDuplex(primer, makeMismatch(0), DefaultConditions())
	if err != nil {
		t.Fatalf("5' mismatch: %v", err)
	}
	three, err := ImperfectDuplex(primer, makeMismatch(len(primer)-1), DefaultConditions())
	if err != nil {
		t.Fatalf("3' mismatch: %v", err)
	}
	if !(three.MismatchPenaltyC > five.MismatchPenaltyC && five.MismatchPenaltyC > internal.MismatchPenaltyC) {
		t.Fatalf("expected 3' > 5' > internal penalties, got 3'=%g 5'=%g internal=%g", three.MismatchPenaltyC, five.MismatchPenaltyC, internal.MismatchPenaltyC)
	}
	if three.ThreePrimeMismatchCount != 1 || five.FivePrimeMismatchCount != 1 {
		t.Fatalf("terminal window counts missing: 3'=%+v 5'=%+v", three, five)
	}
}
