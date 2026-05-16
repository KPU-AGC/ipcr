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

func TestImperfectDuplexMismatchLowersTmAndReportsTripletMetadata(t *testing.T) {
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
	if got.TripletDeltaGCount != 1 || got.CuratedPairCount != 0 || got.HeuristicFallbackCount != 0 || got.UsedHeuristicAdjust {
		t.Fatalf("expected exact triplet metadata without heuristic fallback, got %+v", got)
	}
	if got.MismatchPolicy != MismatchPolicyImperfectTriplet {
		t.Fatalf("expected triplet mismatch policy, got %+v", got)
	}
	if len(got.Contributions) != 1 {
		t.Fatalf("expected one mismatch contribution, got %+v", got.Contributions)
	}
	contrib := got.Contributions[0]
	if contrib.ParameterSet != MismatchParameterSetSantaLuciaHicks2004CompiledDimerGaugeV1 {
		t.Fatalf("expected triplet parameter set, got %+v", contrib)
	}
	if contrib.Citation == "" || contrib.ParameterNote == "" {
		t.Fatalf("expected citation and curation note, got %+v", contrib)
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
	if !(three.MismatchPenaltyC > five.MismatchPenaltyC && internal.MismatchPenaltyC > 0) {
		t.Fatalf("unexpected position penalties, got 3'=%g 5'=%g internal=%g", three.MismatchPenaltyC, five.MismatchPenaltyC, internal.MismatchPenaltyC)
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
	contrib := got.Contributions[0]
	if contrib.TerminalSource != TerminalMismatchSourceHeuristicPenalty || contrib.TerminalParameterSet != TerminalMismatchParameterSetHeuristicV1 {
		t.Fatalf("expected named terminal mismatch fallback provenance, got %+v", contrib)
	}
	if contrib.TerminalCitation == "" || contrib.TerminalParameterNote == "" {
		t.Fatalf("expected terminal mismatch citation and note, got %+v", contrib)
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

func TestLookupTerminalMismatchHeuristicParameter(t *testing.T) {
	key := TerminalMismatchKey{
		PrimerEnd: TerminalMismatchPrimer3Prime,
		P:         'G',
		T:         'A',
		PNeighbor: 'G',
		TNeighbor: 'C',
	}
	param, ok := LookupTerminalMismatchParameterWithFallback(key, DefaultImperfectDuplexOptions())
	if !ok {
		t.Fatalf("expected terminal mismatch fallback parameter for %+v", key)
	}
	if param.Source != TerminalMismatchSourceHeuristicPenalty || param.ParameterSet != TerminalMismatchParameterSetHeuristicV1 {
		t.Fatalf("unexpected terminal mismatch provenance: %+v", param)
	}
	if !param.HasDeltaTm || param.DeltaTmC != defaultThreePrimeTerminalPenalty || param.HasDeltaDeltaG37 {
		t.Fatalf("unexpected terminal mismatch values: %+v", param)
	}
	if param.Citation == "" || param.Note == "" {
		t.Fatalf("expected terminal mismatch citation/note: %+v", param)
	}

	if _, ok := LookupTerminalMismatchParameter(key); ok {
		t.Fatalf("did not expect a curated terminal mismatch table entry before literature-backed values are added")
	}

	perfect := TerminalMismatchKey{PrimerEnd: TerminalMismatchPrimer3Prime, P: 'G', T: 'C', PNeighbor: 'G', TNeighbor: 'C'}
	if _, ok := LookupTerminalMismatchParameterWithFallback(perfect, DefaultImperfectDuplexOptions()); ok {
		t.Fatalf("Watson-Crick terminal pair should not produce a terminal mismatch parameter")
	}

	unknownTarget := TerminalMismatchKey{PrimerEnd: TerminalMismatchPrimer5Prime, P: 'A', T: 'N', PNeighbor: 'C', TNeighbor: 'G'}
	param, ok = LookupTerminalMismatchParameterWithFallback(unknownTarget, DefaultImperfectDuplexOptions())
	if !ok || param.DeltaTmC != defaultFivePrimeTerminalPenalty {
		t.Fatalf("expected heuristic terminal fallback for target N, ok=%v param=%+v", ok, param)
	}
}

func TestTerminalMismatchKeyForPosition(t *testing.T) {
	key, ok := TerminalMismatchKeyForPosition("ACGT", "TGCG", 3)
	if !ok {
		t.Fatal("expected 3' terminal mismatch key")
	}
	want := TerminalMismatchKey{PrimerEnd: TerminalMismatchPrimer3Prime, P: 'T', T: 'G', PNeighbor: 'G', TNeighbor: 'C'}
	if key != want {
		t.Fatalf("unexpected terminal key: got %+v want %+v", key, want)
	}

	if _, ok := TerminalMismatchKeyForPosition("ACGT", "TGGA", 2); ok {
		t.Fatal("internal mismatch/key position should not produce a terminal mismatch key")
	}
	if _, ok := TerminalMismatchKeyForPosition("ACGT", "TGCA", 3); ok {
		t.Fatal("Watson-Crick terminal pair should not produce a terminal mismatch key")
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
		DanglingEndContext{ThreePrimeBase: 'A'},
	)
	if err != nil {
		t.Fatalf("dangling ImperfectDuplex: %v", err)
	}
	if got.DanglingEndCount != 1 || got.DanglingEndAdjustmentC <= 0 || got.DanglingEndDeltaGKcal >= 0 {
		t.Fatalf("expected favorable dangling-end adjustment, got %+v", got)
	}
	if len(got.DanglingContributions) != 1 {
		t.Fatalf("expected one dangling-end contribution, got %+v", got.DanglingContributions)
	}
	c := got.DanglingContributions[0]
	if c.ParameterSet != DanglingEndParameterSetSantaLuciaHicks2004V1 || c.Source == "" || c.Citation == "" {
		t.Fatalf("expected SantaLucia-Hicks dangling-end provenance, got %+v", c)
	}
	if c.Side != "primer-3p" || c.DanglingStrandSide != DanglingEndStrand5Prime || c.Base != 'A' || c.TerminalPrimerBase != 'T' || c.TerminalTargetBase != 'A' {
		t.Fatalf("unexpected dangling-end key mapping: %+v", c)
	}
	if math.Abs(c.DeltaG37kcal-(-0.51)) > 1e-12 || math.Abs(c.DeltaHkcal-0.2) > 1e-12 {
		t.Fatalf("unexpected dangling-end thermodynamics: %+v", c)
	}
	if got.EndEffectPolicy != EndEffectPolicyTemplateDanglingV1 {
		t.Fatalf("unexpected end-effect policy: %q", got.EndEffectPolicy)
	}
	if !(got.AnnealMarginC > base.AnnealMarginC && got.DeltaGAtAnnealKcal < base.DeltaGAtAnnealKcal) {
		t.Fatalf("expected dangling context to stabilize endpoint: base=%+v got=%+v", base.DuplexResult, got.DuplexResult)
	}
}

func TestImperfectDuplexDanglingEndCanBeDestabilizing(t *testing.T) {
	primer := "ACGTACGTACGTACGA"
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
	if got.DanglingEndCount != 1 || len(got.DanglingContributions) != 1 {
		t.Fatalf("expected one dangling-end contribution, got %+v", got)
	}
	c := got.DanglingContributions[0]
	if c.Side != "primer-3p" || c.DanglingStrandSide != DanglingEndStrand5Prime || c.Base != 'G' || c.TerminalPrimerBase != 'A' || c.TerminalTargetBase != 'T' {
		t.Fatalf("unexpected dangling-end key mapping: %+v", c)
	}
	if math.Abs(c.DeltaG37kcal-0.48) > 1e-12 || c.AdjustmentC >= 0 || got.DanglingEndDeltaGKcal <= 0 {
		t.Fatalf("expected destabilizing dangling-end term, got contribution=%+v result=%+v", c, got)
	}
	if !(got.AnnealMarginC < base.AnnealMarginC && got.DeltaGAtAnnealKcal > base.DeltaGAtAnnealKcal) {
		t.Fatalf("expected dangling context to destabilize endpoint: base=%+v got=%+v", base.DuplexResult, got.DuplexResult)
	}
}

func TestLookupDanglingEndParameterSantaLuciaHicksTable3(t *testing.T) {
	if len(CuratedDanglingEnds) != 32 {
		t.Fatalf("curated dangling-end count: got %d want 32", len(CuratedDanglingEnds))
	}
	cases := []struct {
		name       string
		key        DanglingEndKey
		wantDH     float64
		wantDG37   float64
		wantMotif  string
		mappedSide string
	}{
		{
			name:      "5p_XT_A_positive",
			key:       DanglingEndKey{StrandEnd: DanglingEndStrand5Prime, DanglingBase: 'G', PairedBase: 'T', OppositeBase: 'A'},
			wantDH:    -4.2,
			wantDG37:  0.48,
			wantMotif: "GT/A",
		},
		{
			name:      "3p_AX_T_positive",
			key:       DanglingEndKey{StrandEnd: DanglingEndStrand3Prime, DanglingBase: 'C', PairedBase: 'A', OppositeBase: 'T'},
			wantDH:    4.7,
			wantDG37:  0.28,
			wantMotif: "AC/T",
		},
		{
			name:      "3p_GX_C_strong",
			key:       DanglingEndKey{StrandEnd: DanglingEndStrand3Prime, DanglingBase: 'A', PairedBase: 'G', OppositeBase: 'C'},
			wantDH:    -2.1,
			wantDG37:  -0.92,
			wantMotif: "GA/C",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := LookupDanglingEndParameter(tc.key)
			if !ok {
				t.Fatalf("missing dangling-end parameter for %+v", tc.key)
			}
			if got.ParameterSet != DanglingEndParameterSetSantaLuciaHicks2004V1 || got.Source == "" || got.Citation == "" {
				t.Fatalf("missing dangling-end provenance: %+v", got)
			}
			if got.Motif != tc.wantMotif || math.Abs(got.DeltaHkcal-tc.wantDH) > 1e-12 || math.Abs(got.DeltaG37kcal-tc.wantDG37) > 1e-12 {
				t.Fatalf("unexpected parameter: got %+v want DH=%g DG=%g motif=%s", got, tc.wantDH, tc.wantDG37, tc.wantMotif)
			}
		})
	}

	mapped, ok := LookupTemplateDanglingEndParameter("primer-3p", 'G', 'A', 'T')
	if !ok || mapped.Key.StrandEnd != DanglingEndStrand5Prime || mapped.Motif != "GT/A" || math.Abs(mapped.DeltaG37kcal-0.48) > 1e-12 {
		t.Fatalf("unexpected primer-3p target-dangling mapping: ok=%v param=%+v", ok, mapped)
	}
	mapped, ok = LookupTemplateDanglingEndParameter("primer-5p", 'C', 'T', 'A')
	if !ok || mapped.Key.StrandEnd != DanglingEndStrand3Prime || mapped.Motif != "AC/T" || math.Abs(mapped.DeltaG37kcal-0.28) > 1e-12 {
		t.Fatalf("unexpected primer-5p target-dangling mapping: ok=%v param=%+v", ok, mapped)
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
