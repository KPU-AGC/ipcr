// internal/thermovisitors/score_test.go
package thermovisitors

import (
	"ipcr-core/engine"
	"ipcr-core/thermo"
	"ipcr/internal/thermomodel"
	"math"
	"strings"
	"testing"
)

// Small wrapper for tests: forward to the denom-aware DP with a constant D and ssDNA=false.
// D=200.0 matches the conservative fallback used historically.
func dpPenalty(pr, tgt string, allowGap bool) float64 {
	return alignPenaltyC_contextualD_ss(pr, tgt, allowGap, 200.0, false)
}

func TestAlignPenalty_PositionEffects(t *testing.T) {
	// Use a 10-mer so we have a real internal region (K5=3, K3=3 → internal idx 3..6).
	pr := "ACGTACGTAC"
	tgtPerfect := "TGCATGCATG" // 3'→5'

	if p := dpPenalty(pr, tgtPerfect, false); p != 0 {
		t.Fatalf("perfect align penalty = %.2f, want 0", p)
	}
	// 3' mismatch at last base (index 9)
	t3 := []byte(tgtPerfect)
	if t3[len(t3)-1] == 'A' {
		t3[len(t3)-1] = 'G'
	} else {
		t3[len(t3)-1] = 'A'
	}
	p3 := dpPenalty(pr, string(t3), false)

	// 5' mismatch at first base (index 0)
	t5 := []byte(tgtPerfect)
	if t5[0] == 'T' {
		t5[0] = 'A'
	} else {
		t5[0] = 'T'
	}
	p5 := dpPenalty(pr, string(t5), false)

	// Internal mismatch at index 4
	ti := []byte(tgtPerfect)
	switch ti[4] {
	case 'A':
		ti[4] = 'G'
	default:
		ti[4] = 'A'
	}
	pIn := dpPenalty(pr, string(ti), false)

	// Terminal-window weighting should still make the same terminal mismatch more
	// severe at the 3' end than at the 5' end. Internal mismatch magnitude is now
	// context dependent because exact SantaLucia-Hicks triplet overrides are used.
	if !(p3 > p5 && pIn > 0) {
		t.Fatalf("unexpected position penalties: 3' %.2f, 5' %.2f, internal %.2f", p3, p5, pIn)
	}
}

func TestAlignPenalty_UsesExactGTTripletOverride(t *testing.T) {
	// Primer of G's gives an internal G/T triplet key of:
	// 5'-GGG-3' / 3'-CTC-5'. The curated SantaLucia-Hicks compiled-gauge
	// ΔΔG°37 for that context is 3.44 kcal/mol, so D=200 gives 17.20 °C.
	pr := "GGGGGGGGGG"
	tgt := []byte("CCCCCCCCCC") // 3'→5'
	tgt[4] = 'T'

	pGT := dpPenalty(pr, string(tgt), false)
	want := thermo.DeltaGToDeltaTm(3.44, 200.0)
	if math.Abs(pGT-want) > 1e-9 {
		t.Fatalf("GT triplet penalty: got %.12g want %.12g", pGT, want)
	}
}

func rc5to3(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 'A':
			b[len(s)-1-i] = 'T'
		case 'C':
			b[len(s)-1-i] = 'G'
		case 'G':
			b[len(s)-1-i] = 'C'
		case 'T':
			b[len(s)-1-i] = 'A'
		default:
			b[len(s)-1-i] = 'N'
		}
	}
	return string(b)
}

func perfectAmplicon(fwd, rev string, length int) string {
	filler := length - len(fwd) - len(rev)
	if filler < 0 {
		filler = 0
	}
	return fwd + strings.Repeat("A", filler) + rc5to3(rev)
}

func TestScore_ImprovesWithPerfectEnds(t *testing.T) {
	fwd := "ACGTAC"
	rev := "GGTACC"
	amp := fwd + "AAAA" + rc5to3(rev) // perfect ends
	p := engine.Product{
		FwdPrimer: fwd, RevPrimer: rev,
		Seq: amp, Length: len(amp), Type: "forward",
	}

	v := Score{AnnealTempC: 60, Na_M: 0.05, PrimerConc_M: 2.5e-7, AllowIndels: true, LengthBiasOn: false}
	_, p1, _ := v.Visit(p)

	// 3' mismatch on forward end: mutate last base of left end
	ampBad := []byte(amp)
	if ampBad[len(fwd)-1] == 'A' {
		ampBad[len(fwd)-1] = 'C'
	} else {
		ampBad[len(fwd)-1] = 'A'
	}
	pBad := p
	pBad.Seq = string(ampBad)
	_, p2, _ := v.Visit(pBad)

	if !(p1.Score > p2.Score) {
		t.Fatalf("expected score to drop with 3' mismatch: perfect=%.2f, bad=%.2f", p1.Score, p2.Score)
	}
	if math.IsNaN(p1.Score) || math.IsNaN(p2.Score) {
		t.Fatalf("NaN scores")
	}
}

func TestScore_DefaultModelMatchesExplicitNNStructure(t *testing.T) {
	fwd := "ACGTAC"
	rev := "GGTACC"
	amp := fwd + "AAAA" + rc5to3(rev)
	p := engine.Product{
		FwdPrimer: fwd, RevPrimer: rev,
		Seq: amp, Length: len(amp), Type: "forward",
	}

	base := Score{AnnealTempC: 60, Na_M: 0.05, PrimerConc_M: 2.5e-7, AllowIndels: true}
	_, gotDefault, err := base.Visit(p)
	if err != nil {
		t.Fatalf("default Visit returned error: %v", err)
	}

	base.Model = thermomodel.NNStructureV1
	_, gotExplicit, err := base.Visit(p)
	if err != nil {
		t.Fatalf("explicit NNStructure Visit returned error: %v", err)
	}

	if gotDefault.Score != gotExplicit.Score {
		t.Fatalf("default model mismatch: default=%g explicit=%g", gotDefault.Score, gotExplicit.Score)
	}
}

func TestScore_NNDuplexModelProducesThermoComponents(t *testing.T) {
	fwd := "ACGTACGTACGTACGTACGT"
	rev := "ACGTACGTACGTACGTACGT"
	amp := fwd + "AAAA" + rc5to3(rev)
	p := engine.Product{
		FwdPrimer: fwd, RevPrimer: rev,
		Seq: amp, Length: len(amp), Type: "forward",
	}

	v := Score{Model: thermomodel.NNDuplexV1, AnnealTempC: 60, Na_M: 0.05, PrimerConc_M: 2.5e-7}
	_, got, err := v.Visit(p)
	if err != nil {
		t.Fatalf("NNDuplex Visit returned error: %v", err)
	}
	if got.Thermo == nil {
		t.Fatal("expected thermo components")
	}
	if got.Thermo.Model != thermomodel.NNDuplexV1.String() {
		t.Fatalf("got model %q", got.Thermo.Model)
	}
	if got.Score != got.Thermo.ScoreC {
		t.Fatalf("score/component mismatch: %g vs %g", got.Score, got.Thermo.ScoreC)
	}
	if got.Thermo.Fwd.MismatchPenaltyC != 0 || got.Thermo.Rev.MismatchPenaltyC != 0 {
		t.Fatalf("perfect duplex should not have mismatch penalties: %+v", got.Thermo)
	}
}

func TestScore_NNDuplexAnnealTemperatureChangesScore(t *testing.T) {
	fwd := "ACGTACGTACGTACGTACGT"
	rev := "ACGTACGTACGTACGTACGT"
	amp := fwd + "AAAA" + rc5to3(rev)
	p := engine.Product{FwdPrimer: fwd, RevPrimer: rev, Seq: amp}

	low := Score{Model: thermomodel.NNDuplexV1, AnnealTempC: 55, Na_M: 0.05, PrimerConc_M: 2.5e-7}
	high := Score{Model: thermomodel.NNDuplexV1, AnnealTempC: 70, Na_M: 0.05, PrimerConc_M: 2.5e-7}
	_, pLow, err := low.Visit(p)
	if err != nil {
		t.Fatalf("low anneal Visit: %v", err)
	}
	_, pHigh, err := high.Visit(p)
	if err != nil {
		t.Fatalf("high anneal Visit: %v", err)
	}
	if !(pLow.Score > pHigh.Score) {
		t.Fatalf("expected lower anneal temp to produce higher margin score: low=%g high=%g", pLow.Score, pHigh.Score)
	}
}

func TestScore_NNDuplexMismatchUsesFallbackAndLowersScore(t *testing.T) {
	fwd := "ACGTACGTACGTACGTACGT"
	rev := "ACGTACGTACGTACGTACGT"
	amp := fwd + "AAAA" + rc5to3(rev)
	p := engine.Product{FwdPrimer: fwd, RevPrimer: rev, Seq: amp}
	v := Score{Model: thermomodel.NNDuplexV1, AnnealTempC: 60, Na_M: 0.05, PrimerConc_M: 2.5e-7}
	_, perfect, err := v.Visit(p)
	if err != nil {
		t.Fatalf("perfect Visit: %v", err)
	}

	badAmp := []byte(amp)
	badAmp[len(fwd)-1] = 'A'
	if amp[len(fwd)-1] == 'A' {
		badAmp[len(fwd)-1] = 'C'
	}
	p.Seq = string(badAmp)
	_, mismatched, err := v.Visit(p)
	if err != nil {
		t.Fatalf("mismatch Visit: %v", err)
	}
	if !(perfect.Score > mismatched.Score) {
		t.Fatalf("expected mismatch to lower NN score: perfect=%g mismatched=%g", perfect.Score, mismatched.Score)
	}
	if mismatched.Thermo == nil || !mismatched.Thermo.Fwd.HasNonWatsonCrick || mismatched.Thermo.Fwd.MismatchCuratedPairCount != 1 || mismatched.Thermo.Fwd.UsedHeuristicAdjust {
		t.Fatalf("expected fwd curated pair-family mismatch details, got %+v", mismatched.Thermo)
	}
	if mismatched.Thermo.Fwd.MismatchCount != 1 || mismatched.Thermo.Fwd.ThreePrimeMismatchCount != 1 {
		t.Fatalf("expected one 3' mismatch to be reported, got %+v", mismatched.Thermo.Fwd)
	}
	if mismatched.Thermo.Fwd.MismatchPolicy != thermo.MismatchPolicyImperfectCuratedPair {
		t.Fatalf("unexpected mismatch policy: %+v", mismatched.Thermo.Fwd)
	}
}

func TestScore_NNStructureModelAddsStructureComponents(t *testing.T) {
	fwd := "GCGCGCGC"
	rev := "GCGCGCGC"
	amp := fwd + "AAAA" + rc5to3(rev)
	p := engine.Product{FwdPrimer: fwd, RevPrimer: rev, Seq: amp}

	duplex := Score{Model: thermomodel.NNDuplexV1, AnnealTempC: 60, Na_M: 0.05, PrimerConc_M: 2.5e-7}
	_, base, err := duplex.Visit(p)
	if err != nil {
		t.Fatalf("NNDuplex Visit: %v", err)
	}

	structure := Score{
		Model:         thermomodel.NNStructureV1,
		AnnealTempC:   60,
		Na_M:          0.05,
		PrimerConc_M:  2.5e-7,
		StructHairpin: true,
		StructDimer:   true,
		StructScale:   1.0,
	}
	_, got, err := structure.Visit(p)
	if err != nil {
		t.Fatalf("NNStructure Visit: %v", err)
	}
	if got.Thermo == nil || got.Thermo.Model != thermomodel.NNStructureV1.String() {
		t.Fatalf("expected nn-structure-v1 details, got %+v", got.Thermo)
	}
	if got.Thermo.CrossDimer == nil {
		t.Fatalf("expected cross-dimer component, got %+v", got.Thermo)
	}
	if got.Thermo.StructurePenaltyC <= 0 {
		t.Fatalf("expected positive structure penalty, got %+v", got.Thermo)
	}
	if !(got.Score < base.Score) {
		t.Fatalf("expected structure-aware score to be lower than duplex-only score: structure=%g duplex=%g", got.Score, base.Score)
	}
}

func TestScore_NNDuplexBaseScoreMatchesFinalScore(t *testing.T) {
	fwd := "ACGTACGTACGTACGTACGT"
	rev := "ACGTACGTACGTACGTACGT"
	amp := fwd + "AAAA" + rc5to3(rev)
	p := engine.Product{FwdPrimer: fwd, RevPrimer: rev, Seq: amp, Type: "forward"}

	v := Score{Model: thermomodel.NNDuplexV1, AnnealTempC: 60, Na_M: 0.05, PrimerConc_M: 2.5e-7}
	_, got, err := v.Visit(p)
	if err != nil {
		t.Fatalf("NNDuplex Visit returned error: %v", err)
	}
	if got.Thermo == nil {
		t.Fatal("expected thermo details")
	}
	if got.Thermo.BaseScoreC != got.Thermo.ScoreC || got.Score != got.Thermo.BaseScoreC {
		t.Fatalf("expected duplex base/final score parity, got score=%g thermo=%+v", got.Score, got.Thermo)
	}
}

func TestScore_NNStructurePanelCrossDimerPenalty(t *testing.T) {
	fwd := "AAAACGCGCGCGCGCG"
	rev := "TTTTATATATATATAT"
	partner := "TTTTCGCGCGCGCGCG"
	amp := fwd + "AAAA" + rc5to3(rev)
	p := engine.Product{FwdPrimer: fwd, RevPrimer: rev, Seq: amp, Type: "forward"}

	v := Score{
		Model:         thermomodel.NNStructureV1,
		AnnealTempC:   60,
		Na_M:          0.05,
		PrimerConc_M:  2.5e-7,
		StructHairpin: false,
		StructDimer:   true,
		StructScale:   1,
		PanelPrimers: []PrimerRef{
			{ID: "current-fwd", Seq: fwd},
			{ID: "current-rev", Seq: rev},
			{ID: "panel_partner", Seq: partner},
		},
	}
	_, got, err := v.Visit(p)
	if err != nil {
		t.Fatalf("NNStructure Visit returned error: %v", err)
	}
	if got.Thermo == nil {
		t.Fatal("expected thermo details")
	}
	if got.Thermo.PanelCrossDimer == nil {
		t.Fatalf("expected panel cross-dimer details, got %+v", got.Thermo)
	}
	if got.Thermo.PanelCrossDimerPenaltyC <= 0 || got.Thermo.PanelCrossDimerBurdenC <= 0 || got.Thermo.PanelCrossDimerCount <= 0 {
		t.Fatalf("expected positive panel cross-dimer penalty/burden/count, got %+v", got.Thermo)
	}
	if got.Thermo.PanelCrossDimer.QueryB != "panel_partner" {
		t.Fatalf("expected panel partner label, got %+v", got.Thermo.PanelCrossDimer)
	}
	if !(got.Score < got.Thermo.BaseScoreC) {
		t.Fatalf("expected panel dimer penalty to lower score: score=%g base=%g", got.Score, got.Thermo.BaseScoreC)
	}
}

func TestScore_GelProfileAddsAmpliconObservableTerms(t *testing.T) {
	// Xiong-style Salmonella multiplex primers. The binding-only NN score ranks
	// the short, high-Tm product highest; gel-observable ranking should be able
	// to include amplicon mass and extension penalties as explicit components.
	o1 := "ATGTCTATAAGCACCACAATG"
	o2 := "TCATTTCAATAATGATTCAAGC"
	o3 := "CATTCTGACCTTTAAGCCGGTCAATGAG"
	o4 := "CCAAAAAGCGAGACCTCAAACTTACTCAG"
	o5 := "GCGGACGTCATTGTCACTAACCCGACG"
	o6 := "TCTAAAGTGGGAACCCGATGTTCAGCG"

	p155 := engine.Product{ExperimentID: "O5+O6", FwdPrimer: o5, RevPrimer: o6, Seq: perfectAmplicon(o5, o6, 155), Length: 155}
	p339 := engine.Product{ExperimentID: "O3+O4", FwdPrimer: o3, RevPrimer: o4, Seq: perfectAmplicon(o3, o4, 339), Length: 339}
	p882 := engine.Product{ExperimentID: "O1+O2", FwdPrimer: o1, RevPrimer: o2, Seq: perfectAmplicon(o1, o2, 882), Length: 882}

	binding := Score{Model: thermomodel.NNDuplexV1, AnnealTempC: 60, Na_M: 0.05, PrimerConc_M: 2.5e-7}
	_, b155, err := binding.Visit(p155)
	if err != nil {
		t.Fatalf("binding 155 Visit: %v", err)
	}
	_, b339, err := binding.Visit(p339)
	if err != nil {
		t.Fatalf("binding 339 Visit: %v", err)
	}
	if !(b155.Score > b339.Score) {
		t.Fatalf("expected binding-only score to prefer short high-Tm product: 155=%g 339=%g", b155.Score, b339.Score)
	}

	gel := Score{
		Model:          thermomodel.NNStructureV1,
		AnnealTempC:    60,
		Na_M:           0.05,
		PrimerConc_M:   2.5e-7,
		StructHairpin:  true,
		StructDimer:    true,
		StructScale:    1,
		ScoreProfile:   scoreProfileGel,
		ExtAlpha:       0.45,
		ExtWeight:      1,
		LenKneeBP:      550,
		LenSteep:       0.003,
		LenMaxPenC:     10,
		BandMassWeight: 15,
	}
	_, g155, err := gel.Visit(p155)
	if err != nil {
		t.Fatalf("gel 155 Visit: %v", err)
	}
	_, g339, err := gel.Visit(p339)
	if err != nil {
		t.Fatalf("gel 339 Visit: %v", err)
	}
	_, g882, err := gel.Visit(p882)
	if err != nil {
		t.Fatalf("gel 882 Visit: %v", err)
	}

	if g339.Thermo == nil || g339.Thermo.ScoreProfile != scoreProfileGel {
		t.Fatalf("expected gel thermo details, got %+v", g339.Thermo)
	}
	if g339.Thermo.BandMassBonusC <= g155.Thermo.BandMassBonusC {
		t.Fatalf("expected longer visible product to get larger band-mass term: 339=%g 155=%g", g339.Thermo.BandMassBonusC, g155.Thermo.BandMassBonusC)
	}
	if g882.Thermo.LengthPenaltyC <= 0 {
		t.Fatalf("expected long product extension/length penalty, got %+v", g882.Thermo)
	}
	if !(g339.Score > g882.Score && g882.Score > g155.Score) {
		t.Fatalf("expected gel profile rank 339 > 882 > 155; got 339=%g 882=%g 155=%g", g339.Score, g882.Score, g155.Score)
	}
}

func TestScore_NNDuplexReportsEndEffectComponents(t *testing.T) {
	fwd := "ACGTACGTACGTACGTACGT"
	rev := "ACGTACGTACGTACGTACGT"
	amp := fwd + "GAAA" + rc5to3(rev)
	p := engine.Product{FwdPrimer: fwd, RevPrimer: rev, Seq: amp, Length: len(amp), Type: "forward"}
	v := Score{Model: thermomodel.NNDuplexV1, AnnealTempC: 60, Na_M: 0.05, PrimerConc_M: 2.5e-7}

	_, got, err := v.Visit(p)
	if err != nil {
		t.Fatalf("NNDuplex Visit: %v", err)
	}
	if got.Thermo == nil {
		t.Fatal("expected thermo details")
	}
	if got.Thermo.Fwd.DanglingEndCount != 1 || got.Thermo.Fwd.DanglingEndAdjustmentC <= 0 {
		t.Fatalf("expected fwd dangling-end diagnostics, got %+v", got.Thermo.Fwd)
	}
	if got.Thermo.Fwd.EndEffectPolicy != thermo.EndEffectPolicyTemplateDanglingV1 {
		t.Fatalf("unexpected fwd end-effect policy: %+v", got.Thermo.Fwd)
	}

	badAmp := []byte(amp)
	badAmp[len(fwd)-1] = 'A'
	if amp[len(fwd)-1] == 'A' {
		badAmp[len(fwd)-1] = 'C'
	}
	p.Seq = string(badAmp)
	_, mismatched, err := v.Visit(p)
	if err != nil {
		t.Fatalf("mismatched NNDuplex Visit: %v", err)
	}
	if mismatched.Thermo.Fwd.ThreePrimeTerminalMismatchCount != 1 || mismatched.Thermo.Fwd.TerminalMismatchPenaltyC <= 0 {
		t.Fatalf("expected explicit 3' terminal mismatch diagnostics, got %+v", mismatched.Thermo.Fwd)
	}
	if mismatched.Thermo.Fwd.EndEffectPolicy != thermo.EndEffectPolicyTerminalMismatchV1 {
		t.Fatalf("unexpected terminal end-effect policy: %+v", mismatched.Thermo.Fwd)
	}
}

func TestScore_IUPACThermoPolicyWorstBestMeanEnumerate(t *testing.T) {
	fwdDegenerate := "ACGRACGTACGT"
	fwdBestVariant := "ACGAACGTACGT"
	rev := "ACGTACGTACGT"
	amp := fwdBestVariant + strings.Repeat("A", 12) + rc5to3(rev)
	p := engine.Product{FwdPrimer: fwdDegenerate, RevPrimer: rev, Seq: amp, Length: len(amp), Type: "forward"}
	base := Score{Model: thermomodel.NNDuplexV1, AnnealTempC: 60, Na_M: 0.05, PrimerConc_M: 2.5e-7}

	worstVisitor := base
	worstVisitor.IUPACThermoPolicy = thermo.IUPACThermoPolicyWorst
	_, worst, err := worstVisitor.Visit(p)
	if err != nil {
		t.Fatalf("worst Visit: %v", err)
	}
	bestVisitor := base
	bestVisitor.IUPACThermoPolicy = thermo.IUPACThermoPolicyBest
	_, best, err := bestVisitor.Visit(p)
	if err != nil {
		t.Fatalf("best Visit: %v", err)
	}
	meanVisitor := base
	meanVisitor.IUPACThermoPolicy = thermo.IUPACThermoPolicyMean
	_, mean, err := meanVisitor.Visit(p)
	if err != nil {
		t.Fatalf("mean Visit: %v", err)
	}
	enumVisitor := base
	enumVisitor.IUPACThermoPolicy = thermo.IUPACThermoPolicyEnumerate
	_, enumerated, err := enumVisitor.Visit(p)
	if err != nil {
		t.Fatalf("enumerate Visit: %v", err)
	}

	if best.Score <= worst.Score {
		t.Fatalf("expected best score > worst score; best=%g worst=%g", best.Score, worst.Score)
	}
	if !(mean.Score > worst.Score && mean.Score < best.Score) {
		t.Fatalf("expected mean between worst and best; mean=%g worst=%g best=%g", mean.Score, worst.Score, best.Score)
	}
	if worst.Thermo == nil || best.Thermo == nil || mean.Thermo == nil || enumerated.Thermo == nil {
		t.Fatal("expected thermo details for all IUPAC policies")
	}
	if worst.Thermo.IUPACThermoPolicy != thermo.IUPACThermoPolicyWorst || worst.Thermo.IUPACExpansionCount != 2 {
		t.Fatalf("unexpected worst metadata: %+v", worst.Thermo)
	}
	if worst.Thermo.IUPACEffectiveVariant == best.Thermo.IUPACEffectiveVariant {
		t.Fatalf("expected worst and best to select different variants; both=%q", worst.Thermo.IUPACEffectiveVariant)
	}
	if !strings.Contains(worst.Thermo.IUPACEffectiveVariant, "fwd=") || !strings.Contains(best.Thermo.IUPACEffectiveVariant, "fwd=") {
		t.Fatalf("expected concrete effective variants; worst=%q best=%q", worst.Thermo.IUPACEffectiveVariant, best.Thermo.IUPACEffectiveVariant)
	}
	if enumerated.Thermo.IUPACThermoPolicy != thermo.IUPACThermoPolicyEnumerate || len(enumerated.Thermo.IUPACVariants) != 2 {
		t.Fatalf("expected enumerated variants, got %+v", enumerated.Thermo)
	}
}

func TestScore_IUPACThermoExpansionCap(t *testing.T) {
	fwd := "NNNN"
	rev := "ACGT"
	amp := "AAAA" + strings.Repeat("A", 12) + rc5to3(rev)
	p := engine.Product{FwdPrimer: fwd, RevPrimer: rev, Seq: amp, Length: len(amp), Type: "forward"}
	v := Score{Model: thermomodel.NNDuplexV1, AnnealTempC: 60, Na_M: 0.05, PrimerConc_M: 2.5e-7, IUPACThermoPolicy: thermo.IUPACThermoPolicyWorst, IUPACThermoMaxExpansions: 3}
	_, got, err := v.Visit(p)
	if err != nil {
		t.Fatalf("Visit: %v", err)
	}
	if got.Thermo == nil || got.Thermo.IUPACExpansionCount != 3 || !got.Thermo.IUPACExpansionCapped {
		t.Fatalf("expected capped IUPAC metadata, got %+v", got.Thermo)
	}
}

func TestScore_ProbeThermoGateKeepsFoundProbeAndAnnotates(t *testing.T) {
	fwd := "ACGTACGTACGTACGTACGT"
	rev := "TGCATGCATGCATGCATGCA"
	probe := "GATTACAGATTACAGATTAC"
	amp := fwd + "AAAA" + probe + "AAAA" + rc5to3(rev)
	p := engine.Product{FwdPrimer: fwd, RevPrimer: rev, Seq: amp, Length: len(amp), Type: "forward"}

	v := Score{
		Model:           thermomodel.NNDuplexV1,
		AnnealTempC:     60,
		Na_M:            0.05,
		PrimerConc_M:    2.5e-7,
		ProbeSeq:        probe,
		ProbeName:       "p1",
		ProbeThermo:     true,
		ProbeScoreMode:  probeScoreModeGate,
		ProbeMinMarginC: -100,
	}
	keep, got, err := v.Visit(p)
	if err != nil {
		t.Fatalf("Visit returned error: %v", err)
	}
	if !keep {
		t.Fatal("expected probe-gated product to be kept")
	}
	if got.Thermo == nil || got.Thermo.Probe == nil {
		t.Fatalf("expected probe thermo details, got %+v", got.Thermo)
	}
	pr := got.Thermo.Probe
	if !pr.Found || pr.Name != "p1" || pr.Seq != probe || pr.ScoreMode != probeScoreModeGate {
		t.Fatalf("unexpected probe annotation: %+v", pr)
	}
	if pr.TmC == 0 || pr.IUPACExpansionCount != 1 || pr.IUPACEffectiveVariant != probe {
		t.Fatalf("expected populated probe thermo fields, got %+v", pr)
	}
}

func TestScore_ProbeThermoGateDropsMissingProbe(t *testing.T) {
	fwd := "ACGTACGTACGTACGTACGT"
	rev := "TGCATGCATGCATGCATGCA"
	amp := fwd + strings.Repeat("A", 40) + rc5to3(rev)
	p := engine.Product{FwdPrimer: fwd, RevPrimer: rev, Seq: amp, Length: len(amp), Type: "forward"}

	v := Score{
		Model:          thermomodel.NNDuplexV1,
		AnnealTempC:    60,
		Na_M:           0.05,
		PrimerConc_M:   2.5e-7,
		ProbeSeq:       "GATTACAGATTACAGATTAC",
		ProbeThermo:    true,
		ProbeScoreMode: probeScoreModeGate,
	}
	keep, _, err := v.Visit(p)
	if err != nil {
		t.Fatalf("Visit returned error: %v", err)
	}
	if keep {
		t.Fatal("expected gate mode to drop products missing the probe")
	}
}

func TestScore_ProbeThermoBlendPenalizesMissingProbe(t *testing.T) {
	fwd := "ACGTACGTACGTACGTACGT"
	rev := "TGCATGCATGCATGCATGCA"
	amp := fwd + strings.Repeat("A", 40) + rc5to3(rev)
	p := engine.Product{FwdPrimer: fwd, RevPrimer: rev, Seq: amp, Length: len(amp), Type: "forward"}

	base := Score{Model: thermomodel.NNDuplexV1, AnnealTempC: 60, Na_M: 0.05, PrimerConc_M: 2.5e-7}
	_, withoutProbe, err := base.Visit(p)
	if err != nil {
		t.Fatalf("base Visit returned error: %v", err)
	}
	withProbe := base
	withProbe.ProbeSeq = "GATTACAGATTACAGATTAC"
	withProbe.ProbeThermo = true
	withProbe.ProbeScoreMode = probeScoreModeBlend
	keep, got, err := withProbe.Visit(p)
	if err != nil {
		t.Fatalf("probe Visit returned error: %v", err)
	}
	if !keep {
		t.Fatal("blend mode should not drop missing-probe products")
	}
	if !(got.Score < withoutProbe.Score) {
		t.Fatalf("expected missing probe to penalize score: base=%g got=%g", withoutProbe.Score, got.Score)
	}
	if got.Thermo == nil || got.Thermo.Probe == nil || got.Thermo.Probe.ScoreContributionC >= 0 {
		t.Fatalf("expected negative probe score contribution, got %+v", got.Thermo)
	}
}
