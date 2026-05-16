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

func TestImperfectDuplexMismatchLowersTmAndReportsCuratedPairMetadata(t *testing.T) {
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
	if got.CuratedPairCount != 1 || got.HeuristicFallbackCount != 0 || got.UsedHeuristicAdjust {
		t.Fatalf("expected curated pair-family metadata without heuristic fallback, got %+v", got)
	}
	if got.MismatchPolicy != MismatchPolicyImperfectCuratedPair {
		t.Fatalf("expected curated pair-family mismatch policy, got %+v", got)
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

func TestImperfectDuplexReportsExplicitTerminalMismatchPenalty(t *testing.T) {
	primer := "GGGGGGGGGG"
	perfect := []byte(comp(primer))
	terminalTarget := append([]byte(nil), perfect...)
	terminalTarget[len(terminalTarget)-1] = 'A'

	got, err := ImperfectDuplex(primer, string(terminalTarget), DefaultConditions())
	if err != nil {
		t.Fatalf("ImperfectDuplex: %v", err)
	}
	if got.ThreePrimeTerminalMismatchCount != 1 || got.TerminalMismatchCount != 1 {
		t.Fatalf("expected one explicit 3' terminal mismatch, got %+v", got)
	}
	if got.ThreePrimeTerminalMismatchPenaltyC <= 0 || got.TerminalMismatchDeltaGKcal <= 0 {
		t.Fatalf("expected explicit terminal penalty terms, got %+v", got)
	}
	if got.EndEffectPolicy != EndEffectPolicyTerminalMismatchV1 {
		t.Fatalf("unexpected end-effect policy: %q", got.EndEffectPolicy)
	}
	if len(got.Contributions) != 1 || !got.Contributions[0].ThreePrimeTerminal || got.Contributions[0].TerminalPenaltyC <= 0 {
		t.Fatalf("expected contribution-level terminal annotation, got %+v", got.Contributions)
	}

	internalTarget := append([]byte(nil), perfect...)
	internalTarget[4] = 'A'
	internal, err := ImperfectDuplex(primer, string(internalTarget), DefaultConditions())
	if err != nil {
		t.Fatalf("internal ImperfectDuplex: %v", err)
	}
	if internal.TerminalMismatchPenaltyC != 0 || internal.TerminalMismatchCount != 0 {
		t.Fatalf("internal mismatch should not report terminal terms, got %+v", internal)
	}
}

func TestImperfectDuplexDanglingEndContextRaisesMargin(t *testing.T) {
	primer := "ACGTACGTACGTACGT"
	target := comp(primer)
	cond := DefaultConditions()
	base, err := ImperfectDuplex(primer, target, cond)
	if err != nil {
		t.Fatalf("base ImperfectDuplex: %v", err)
	}
	got, err := ImperfectDuplexWithOptionsAndContext(
		primer,
		target,
		cond,
		DefaultImperfectDuplexOptions(),
		DanglingEndContext{ThreePrimeBase: 'G'},
	)
	if err != nil {
		t.Fatalf("dangling ImperfectDuplex: %v", err)
	}
	if got.DanglingEndCount != 1 || got.DanglingEndAdjustmentC <= 0 || got.DanglingEndDeltaGKcal >= 0 {
		t.Fatalf("expected favorable dangling-end adjustment, got %+v", got)
	}
	if got.EndEffectPolicy != EndEffectPolicyTemplateDanglingV1 {
		t.Fatalf("unexpected end-effect policy: %q", got.EndEffectPolicy)
	}
	if !(got.AnnealMarginC > base.AnnealMarginC && got.DeltaGAtAnnealKcal < base.DeltaGAtAnnealKcal) {
		t.Fatalf("expected dangling context to stabilize endpoint: base=%+v got=%+v", base.DuplexResult, got.DuplexResult)
	}
}

func TestLookupMismatchParameterInfoCuratedPair(t *testing.T) {
	param, ok := LookupMismatchParameterInfo(broadMismatchKey('G', 'T'))
	if !ok {
		t.Fatal("expected curated G/T pair-family parameter")
	}
	if param.Source != MismatchSourceCuratedPairDeltaG || param.ParameterSet != MismatchParameterSetPairFamilyV1 || param.DeltaDeltaGKcal <= 0 {
		t.Fatalf("unexpected curated parameter: %+v", param)
	}
}
