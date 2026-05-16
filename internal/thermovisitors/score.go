// internal/thermovisitors/score.go
package thermovisitors

import (
	"fmt"
	"ipcr-core/engine"
	probeanno "ipcr-core/probe"
	"ipcr-core/thermo"
	"ipcr-core/thermoaddons"
	"ipcr/internal/thermomodel"
	"math"
	"os"
	"strings"
	"unicode"
)

const (
	PEN_GAP_1NT         = 6.0
	MIN_USEFUL_TMC      = -10.0
	MAX_USEFUL_TMC      = 120.0
	K5                  = 3 // 5' end is harsher across first K5 bases
	K3                  = 3 // 3' end is harshest across last K3 bases
	PROBE_NOT_FOUND_PEN = 12.0

	iupacPolicyStrictACGT           = "strict-acgt"
	mismatchPolicyNNPerfect         = thermo.MismatchPolicyPerfect
	mismatchPolicyHeuristicFallback = thermo.MismatchPolicyImperfectHeuristicFallback
	mismatchPolicyMixed             = "nn-perfect-or-nn-imperfect-v1"
	structurePolicyNNStemV1         = thermo.StructureModelContiguousStemV1
	structurePolicyNNStemLoopV2     = thermo.StructureModelStemLoopV2

	scoreProfileBinding = "binding"
	scoreProfilePCR     = "pcr"
	scoreProfileGel     = "gel"

	probeScoreModeAnnotate = "annotate"
	probeScoreModeGate     = "gate"
	probeScoreModeBlend    = "blend"

	defaultBandMassWeightC = 15.0
	bandMassRefBP          = 100.0
)

// PrimerRef identifies one primer in the current panel/pool for panel-wide
// dimer competition checks. Seq is expected in 5′→3′ orientation.
type PrimerRef struct {
	ID  string
	Seq string
}

// Score is the thermo-scoring visitor config.
type Score struct {
	Model      thermomodel.Mode
	Conditions thermo.Conditions

	AnnealTempC    float64
	Na_M           float64
	PrimerConc_M   float64
	AllowIndels    bool
	LengthBiasOn   bool
	SingleStranded bool // read (OR'd with env) to enable ssDNA tweaks
	StructHairpin  bool
	StructDimer    bool
	StructScale    float64
	PanelPrimers   []PrimerRef

	IUPACThermoPolicy        string
	IUPACThermoMaxExpansions int

	// ScoreProfile controls whether NN model scores remain pure primer-template
	// binding margins or include PCR/gel-observable amplicon-level terms.
	ScoreProfile   string
	ExtAlpha       float64
	ExtWeight      float64
	LenKneeBP      int
	LenSteep       float64
	LenMaxPenC     float64
	BindWeight     float64
	BandMassWeight float64

	ProbeSeq        string
	ProbeName       string
	ProbeMaxMM      int
	ProbeThermo     bool
	ProbeScoreMode  string
	ProbeMinMarginC float64
	ProbeWeight     float64

	// Opt-in: compute ΔΔG→ΔTm denominator from solution conditions.
	// Default false keeps the historical fixed D=200 path.
	UseAutoDenom bool
}

func (v Score) conditions() thermo.Conditions {
	c := v.Conditions
	if c.AnnealC == 0 {
		c.AnnealC = v.AnnealTempC
	}
	if c.NaM == 0 {
		c.NaM = v.Na_M
	}
	if c.PrimerTotalM == 0 {
		c.PrimerTotalM = v.PrimerConc_M
	}
	if c.SaltModel == "" {
		c.SaltModel = thermo.SaltModelMonovalent
	}
	return c.WithDefaults()
}

func (v Score) scoreProfile() string {
	switch strings.ToLower(strings.TrimSpace(v.ScoreProfile)) {
	case "", scoreProfileBinding:
		return scoreProfileBinding
	case scoreProfilePCR:
		return scoreProfilePCR
	case scoreProfileGel:
		return scoreProfileGel
	default:
		return scoreProfileBinding
	}
}

func (v Score) iupacThermoPolicy() string {
	policy, err := thermo.ParseIUPACThermoPolicy(v.IUPACThermoPolicy)
	if err != nil {
		return thermo.IUPACThermoPolicyWorst
	}
	return policy
}

func (v Score) iupacThermoMaxExpansions() int {
	if v.IUPACThermoMaxExpansions < 1 {
		return 256
	}
	return v.IUPACThermoMaxExpansions
}

func (v Score) extAlpha() float64 {
	if v.ExtAlpha == 0 {
		return 0.45
	}
	if v.ExtAlpha < 0 {
		return 0
	}
	return v.ExtAlpha
}

func (v Score) extWeight() float64 {
	if v.ExtWeight == 0 {
		return 1
	}
	return v.ExtWeight
}

func (v Score) lenKneeBP() int {
	if v.LenKneeBP <= 0 {
		return 550
	}
	return v.LenKneeBP
}

func (v Score) lenSteep() float64 {
	if v.LenSteep == 0 {
		return 0.003
	}
	if v.LenSteep < 0 {
		return 0
	}
	return v.LenSteep
}

func (v Score) lenMaxPenC() float64 {
	if v.LenMaxPenC == 0 {
		return 10
	}
	if v.LenMaxPenC < 0 {
		return 0
	}
	return v.LenMaxPenC
}

func (v Score) bindWeight() float64 {
	if v.BindWeight == 0 {
		return 1
	}
	return v.BindWeight
}

func (v Score) bandMassWeight() float64 {
	if v.BandMassWeight == 0 {
		return defaultBandMassWeightC
	}
	return v.BandMassWeight
}

func (v Score) probeName() string {
	name := strings.TrimSpace(v.ProbeName)
	if name == "" {
		return "probe"
	}
	return name
}

func (v Score) probeScoreMode() string {
	switch strings.ToLower(strings.TrimSpace(v.ProbeScoreMode)) {
	case probeScoreModeAnnotate:
		return probeScoreModeAnnotate
	case "", probeScoreModeGate:
		return probeScoreModeGate
	case probeScoreModeBlend:
		return probeScoreModeBlend
	default:
		return probeScoreModeGate
	}
}

func (v Score) probeWeight() float64 {
	if v.ProbeWeight < 0 {
		return 0
	}
	if v.ProbeWeight > 1 {
		return 1
	}
	return v.ProbeWeight
}

func (v Score) probeThermoEnabled() bool {
	return strings.TrimSpace(v.ProbeSeq) != "" && v.ProbeThermo
}

// Public helper used by tests/tools.
func (v *Score) Penalty(primer5to3, tgt3to5 string, denom float64) float64 {
	ssOn := v.SingleStranded || singleStrandedMode()
	return alignPenaltyC_contextualD_ss(primer5to3, tgt3to5, v.AllowIndels, denom, ssOn)
}

func toUpperACGT(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := unicode.ToUpper(rune(s[i]))
		switch c {
		case 'A', 'C', 'G', 'T':
			b = append(b, byte(c))
		default:
			return ""
		}
	}
	return string(b)
}

func toUpperACGTAllowN(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := unicode.ToUpper(rune(s[i]))
		switch c {
		case 'A', 'C', 'G', 'T', 'N':
			b = append(b, byte(c))
		default:
			return ""
		}
	}
	return string(b)
}

func wcACGT(a, b byte) bool {
	switch a {
	case 'A':
		return b == 'T'
	case 'C':
		return b == 'G'
	case 'G':
		return b == 'C'
	case 'T':
		return b == 'A'
	default:
		return false
	}
}

func posMultiplier(i, n int) float64 {
	if i >= n-K3 {
		return 2.0
	}
	if i < K5 {
		return 1.5
	}
	return 1.0
}

func at(s string, idx int) byte {
	if idx < 0 || idx >= len(s) {
		return 'N'
	}
	return s[idx]
}

func rev(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

func compBase(b byte) byte {
	switch b {
	case 'A':
		return 'T'
	case 'T':
		return 'A'
	case 'C':
		return 'G'
	case 'G':
		return 'C'
	default:
		return 'N'
	}
}

// complement (5'→3' top → 3'→5' bottom), per position (no reverse).
func comp5to3(top string) string {
	out := make([]byte, len(top))
	for i := 0; i < len(top); i++ {
		out[i] = compBase(top[i])
	}
	return string(out)
}

func absFiniteOrFallback(x, fallback float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) || x == 0 {
		return fallback
	}
	if x < 0 {
		return -x
	}
	return x
}

// Env-based ssDNA toggle (kept for backwards compatibility).
func singleStrandedMode() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("IPCR_SINGLE_STRANDED")))
	return v == "1" || v == "true" || v == "yes"
}

// Tiny, bounded “dangling-end” bonus per end.
func ssEndBonusApprox(top, bot byte) float64 {
	switch {
	case (top == 'A' && bot == 'T') || (top == 'T' && bot == 'A'):
		return 0.10
	case (top == 'G' && bot == 'C') || (top == 'C' && bot == 'G'):
		return 0.20
	default:
		return 0.0
	}
}

// -------------- DP with contextual mismatch penalties --------------
//
// denom = effective denominator D (cal/K/mol) used only for ΔΔG→ΔTm fallback.

func alignPenaltyC_contextualD_ss(primer5to3, tgt3to5 string, allowGap bool, denom float64, ssOn bool) float64 {
	P := toUpperACGT(primer5to3)
	T := toUpperACGTAllowN(tgt3to5)
	n, m := len(P), len(T)
	if n == 0 || m == 0 {
		return 0
	}
	gapAllowed := 0
	if allowGap {
		gapAllowed = 1
	}
	const INF = 1e9
	dp := make([][][2]float64, n+1)
	for i := 0; i <= n; i++ {
		dp[i] = make([][2]float64, m+1)
		for j := 0; j <= m; j++ {
			dp[i][j][0], dp[i][j][1] = INF, INF
		}
	}
	dp[0][0][0] = 0

	singleMM := func(i, j int) float64 {
		p5, pC, p3 := at(P, i-1), P[i], at(P, i+1)
		t5, tC, t3 := at(T, j-1), T[j], at(T, j+1)
		pen := 0.0
		if dTm, ok := thermo.LookupDeltaTm(p5, pC, p3, t5, tC, t3); ok {
			pen = dTm
		} else if dG, ok := thermo.LookupDeltaG(p5, pC, p3, t5, tC, t3); ok {
			pen = thermo.DeltaGToDeltaTm(dG, denom)
		} else {
			pen = 4.0
		}
		return pen * posMultiplier(i, n)
	}

	for i := 0; i <= n; i++ {
		for j := 0; j <= m; j++ {
			for g := 0; g <= gapAllowed; g++ {
				cur := dp[i][j][g]
				if cur >= INF/2 {
					continue
				}
				// match/mismatch
				if i < n && j < m {
					pen := 0.0
					if !wcACGT(P[i], T[j]) {
						pen = singleMM(i, j)
					}
					if cur+pen < dp[i+1][j+1][g] {
						dp[i+1][j+1][g] = cur + pen
					}
				}
				// 1-nt gap (once)
				if g == 0 && i < n {
					w := posMultiplier(i, n)
					val := cur + PEN_GAP_1NT*w
					if val < dp[i+1][j][1] {
						dp[i+1][j][1] = val
					}
				}
				if g == 0 && j < m {
					w := posMultiplier(i, n)
					val := cur + PEN_GAP_1NT*w
					if val < dp[i][j+1][1] {
						dp[i][j+1][1] = val
					}
				}
			}
		}
	}
	best := math.Min(dp[n][m][0], dp[n][m][1])
	if best >= INF/2 {
		return 0
	}

	// ssDNA adjustments (data-driven)
	if ssOn && n > 0 {
		leftTop := P[0]
		rightTop := P[n-1]
		leftBot := compBase(leftTop)
		rightBot := compBase(rightTop)
		endBonus := ssEndBonusApprox(leftTop, leftBot) + ssEndBonusApprox(rightTop, rightBot)

		// Target hairpin hindrance on local ssDNA: T is 3'→5', reverse for 5'→3'
		target5to3 := rev(T)
		hairpinPen := thermoaddons.HairpinPenalty(target5to3)

		best = best - endBonus + hairpinPen
	}

	// Confidence cap: never “better than perfect”.
	if best < 0 {
		best = 0
	}
	return best
}

// denomForPrimer computes D = ΔS_Na + R·ln(CT/X) using NN Tm on the
// primer vs its perfect complement. X=4 (non-self) unless the primer is self-compl.
// denomForPrimer computes |ΔS_Na + R·ln(CT/X)| using NN Tm on the
// primer vs its perfect complement. X=4 (non-self) unless the primer is self-compl.
// We return the absolute value so ΔΔG→ΔTm scaling is a positive magnitude.
func (v Score) denomForPrimer(primer5to3 string) float64 {
	p := toUpperACGT(primer5to3)
	cond := v.conditions()
	if p == "" || cond.NaM <= 0 || cond.PrimerTotalM <= 0 {
		return 200.0
	}
	// Build 3'→5' complement for Tm().
	t3 := comp5to3(p)

	// Self-compl check: rc == p (5'→3').
	rc := rev(comp5to3(p))
	cond.SelfComplementary = rc == p
	in := cond.TmInput()

	res, err := thermo.Tm(p, t3, in)
	if err != nil {
		return 200.0
	}
	D := res.DS_Na + thermo.Rcal*math.Log(in.CT/float64(in.X))
	// Go 1.22-safe "finite" check, then take magnitude.
	if math.IsNaN(D) || math.IsInf(D, 0) || D == 0 {
		return 200.0
	}
	if D < 0 {
		D = -D
	}
	return D
}

func endpointFromDuplex(side string, d thermo.DuplexResult, mismatchPenaltyC float64, policy string, hasNonWC, heuristic bool) engine.ThermoEndpoint {
	return engine.ThermoEndpoint{
		Side:                side,
		TmC:                 d.TmC,
		AnnealMarginC:       d.AnnealMarginC,
		DeltaGAtAnnealKcal:  d.DeltaGAtAnnealKcal,
		MismatchPenaltyC:    mismatchPenaltyC,
		EffectiveDenomCalK:  absFiniteOrFallback(d.EffectiveDenomCalK, 200.0),
		MismatchPolicy:      policy,
		EndEffectPolicy:     thermo.EndEffectPolicyNone,
		HasNonWatsonCrick:   hasNonWC,
		UsedHeuristicAdjust: heuristic,
	}
}

func endpointFromImperfect(side string, d thermo.ImperfectDuplexResult) engine.ThermoEndpoint {
	return engine.ThermoEndpoint{
		Side:                               side,
		TmC:                                d.TmC,
		AnnealMarginC:                      d.AnnealMarginC,
		DeltaGAtAnnealKcal:                 d.DeltaGAtAnnealKcal,
		MismatchPenaltyC:                   d.MismatchPenaltyC,
		MismatchDeltaGKcal:                 d.DeltaGPenaltyKcal,
		TerminalMismatchPenaltyC:           d.TerminalMismatchPenaltyC,
		TerminalMismatchDeltaGKcal:         d.TerminalMismatchDeltaGKcal,
		DanglingEndAdjustmentC:             d.DanglingEndAdjustmentC,
		DanglingEndDeltaGKcal:              d.DanglingEndDeltaGKcal,
		DanglingEndCount:                   d.DanglingEndCount,
		MismatchCount:                      d.MismatchCount,
		FivePrimeMismatchCount:             d.FivePrimeMismatchCount,
		ThreePrimeMismatchCount:            d.ThreePrimeMismatchCount,
		FivePrimeTerminalMismatchCount:     d.FivePrimeTerminalMismatchCount,
		ThreePrimeTerminalMismatchCount:    d.ThreePrimeTerminalMismatchCount,
		TerminalMismatchCount:              d.TerminalMismatchCount,
		FivePrimeTerminalMismatchPenaltyC:  d.FivePrimeTerminalMismatchPenaltyC,
		ThreePrimeTerminalMismatchPenaltyC: d.ThreePrimeTerminalMismatchPenaltyC,
		MismatchFallbackCount:              d.HeuristicFallbackCount + d.DefaultFallbackCount,
		MismatchTripletCount:               d.TripletTmCount + d.TripletDeltaGCount,
		MismatchCuratedPairCount:           d.CuratedPairCount,
		EffectiveDenomCalK:                 absFiniteOrFallback(d.EffectiveDenomCalK, 200.0),
		MismatchPolicy:                     d.MismatchPolicy,
		EndEffectPolicy:                    d.EndEffectPolicy,
		HasNonWatsonCrick:                  d.HasNonWatsonCrick,
		UsedHeuristicAdjust:                d.UsedHeuristicAdjust,
	}
}

func (v Score) scoreNNDuplexEndpoint(side, primer5to3, target3to5 string, dangling thermo.DanglingEndContext) (engine.ThermoEndpoint, error) {
	primer := toUpperACGT(primer5to3)
	if primer == "" {
		return engine.ThermoEndpoint{}, fmt.Errorf("nn-duplex-v1 requires A/C/G/T primers; %s primer contains unsupported bases", side)
	}
	target := toUpperACGTAllowN(target3to5)
	if target == "" {
		return engine.ThermoEndpoint{}, fmt.Errorf("nn-duplex-v1 requires A/C/G/T/N target sites; %s target contains unsupported bases", side)
	}
	if len(primer) != len(target) {
		return engine.ThermoEndpoint{}, fmt.Errorf("nn-duplex-v1 %s endpoint length mismatch: primer=%d target=%d", side, len(primer), len(target))
	}

	cond := v.conditions()
	ssOn := v.SingleStranded || singleStrandedMode()
	if !v.AllowIndels && !ssOn {
		imperfect, err := thermo.ImperfectDuplexWithOptionsAndContext(primer, target, cond, thermo.DefaultImperfectDuplexOptions(), dangling)
		if err != nil {
			return engine.ThermoEndpoint{}, err
		}
		return endpointFromImperfect(side, imperfect), nil
	}

	// Gap-tolerant and ssDNA adjustments are not yet part of the NN imperfect
	// duplex core. Preserve the historical DP fallback for those opt-in modes and
	// label it explicitly.
	perfectTarget := comp5to3(primer)
	base, err := thermo.PerfectDuplex(primer, perfectTarget, cond)
	if err != nil {
		return engine.ThermoEndpoint{}, err
	}
	denom := absFiniteOrFallback(base.EffectiveDenomCalK, 200.0)
	penaltyC := alignPenaltyC_contextualD_ss(primer, target, v.AllowIndels, denom, ssOn)
	deltaGPenalty := penaltyC * denom / 1000.0

	adjusted := base
	adjusted.TmC = base.TmC - penaltyC
	adjusted.AnnealMarginC = adjusted.TmC - cond.AnnealC
	adjusted.DeltaGAtAnnealKcal = base.DeltaGAtAnnealKcal + deltaGPenalty
	adjusted.EffectiveDenomCalK = denom
	return endpointFromDuplex(side, adjusted, penaltyC, mismatchPolicyHeuristicFallback, true, true), nil
}

func (v Score) scoreNNDuplexComponents(p engine.Product) (engine.ThermoEndpoint, engine.ThermoEndpoint, float64, string, thermo.Conditions, error) {
	f := toUpperACGT(p.FwdPrimer)
	r := toUpperACGT(p.RevPrimer)
	if f == "" || r == "" {
		return engine.ThermoEndpoint{}, engine.ThermoEndpoint{}, 0, "", thermo.Conditions{}, fmt.Errorf("nn-duplex-v1 requires A/C/G/T primers; degenerate/IUPAC primer scoring is not implemented yet")
	}
	if len(p.Seq) < len(f) || len(p.Seq) < len(r) {
		return engine.ThermoEndpoint{}, engine.ThermoEndpoint{}, 0, "", thermo.Conditions{}, fmt.Errorf("nn-duplex-v1 requires product sequence long enough for both primer sites")
	}

	leftSite := toUpperACGTAllowN(p.Seq[:len(f)])
	rightSite := toUpperACGTAllowN(p.Seq[len(p.Seq)-len(r):])
	if leftSite == "" || rightSite == "" {
		return engine.ThermoEndpoint{}, engine.ThermoEndpoint{}, 0, "", thermo.Conditions{}, fmt.Errorf("nn-duplex-v1 requires A/C/G/T/N product sequence at primer sites")
	}

	// The left site is in the same 5'→3' orientation as the forward primer.
	// The right site is the reference-strand reverse complement of the reverse
	// primer, so reversing the site gives the primer-aligned target strand 3'→5'.
	fwdTarget3 := comp5to3(leftSite)
	revTarget3 := rev(rightSite)
	fwdDangling := thermo.DanglingEndContext{}
	revDangling := thermo.DanglingEndContext{}
	if len(p.Seq) > len(f) {
		// Forward primer binds the bottom/complement strand; the amplicon-interior
		// base after the forward site is converted to the aligned template base.
		fwdDangling.ThreePrimeBase = compBase(byte(unicode.ToUpper(rune(p.Seq[len(f)]))))
	}
	if len(p.Seq) > len(r) {
		idx := len(p.Seq) - len(r) - 1
		if idx >= 0 {
			// Reverse primer binds the top/reference strand; the amplicon-interior
			// base before the reverse site is already the aligned template base.
			revDangling.ThreePrimeBase = byte(unicode.ToUpper(rune(p.Seq[idx])))
		}
	}

	fwd, err := v.scoreNNDuplexEndpoint("fwd", f, fwdTarget3, fwdDangling)
	if err != nil {
		return engine.ThermoEndpoint{}, engine.ThermoEndpoint{}, 0, "", thermo.Conditions{}, err
	}
	revEnd, err := v.scoreNNDuplexEndpoint("rev", r, revTarget3, revDangling)
	if err != nil {
		return engine.ThermoEndpoint{}, engine.ThermoEndpoint{}, 0, "", thermo.Conditions{}, err
	}

	limitingSide := "fwd"
	score := fwd.AnnealMarginC
	if revEnd.AnnealMarginC < score {
		score = revEnd.AnnealMarginC
		limitingSide = "rev"
	}
	return fwd, revEnd, score, limitingSide, v.conditions(), nil
}

func nnThermoDetails(model thermomodel.Mode, cond thermo.Conditions, fwd, revEnd engine.ThermoEndpoint, score float64, limitingSide string) *engine.ThermoDetails {
	return &engine.ThermoDetails{
		Model:          model.String(),
		SaltModel:      cond.SaltModel.String(),
		NaM:            cond.NaM,
		MgM:            cond.MgM,
		DntpM:          cond.DntpM,
		EffectiveNaM:   cond.EffectiveNaM(),
		FreeMgM:        cond.FreeMgM(),
		AnnealTempC:    cond.AnnealC,
		IUPACPolicy:    iupacPolicyStrictACGT,
		MismatchPolicy: mismatchPolicyMixed,
		ScoreProfile:   scoreProfileBinding,
		ScoreC:         score,
		BaseScoreC:     score,
		LimitingSide:   limitingSide,
		Fwd:            fwd,
		Rev:            revEnd,
	}
}

func ampliconBandMassBonusC(bp int, weightC float64) float64 {
	if bp <= 0 || weightC == 0 {
		return 0
	}
	ratio := float64(bp) / bandMassRefBP
	if ratio <= 0 {
		return 0
	}
	return weightC * math.Log2(ratio)
}

func (v Score) applyAmpliconProfile(p engine.Product, details *engine.ThermoDetails, score float64) float64 {
	if details == nil {
		return score
	}
	profile := v.scoreProfile()
	details.ScoreProfile = profile
	if profile == scoreProfileBinding {
		details.ScoreC = score
		return score
	}

	limitingMargin := details.Fwd.AnnealMarginC
	if details.Rev.AnnealMarginC < limitingMargin {
		limitingMargin = details.Rev.AnnealMarginC
	}

	bindingAdjustment := score * (v.bindWeight() - 1)

	extProb := thermoaddons.ExtensionProb(limitingMargin, v.extAlpha())
	extLogit := thermoaddons.Logit(extProb)
	extBonus := v.extWeight() * extLogit
	lengthPenalty := thermoaddons.LengthPenalty(p.Length, v.lenKneeBP(), v.lenSteep(), v.lenMaxPenC())

	details.ExtensionLogit = extLogit
	details.ExtensionBonusC = extBonus
	details.LengthPenaltyC = lengthPenalty

	adjustment := bindingAdjustment + extBonus - lengthPenalty
	if profile == scoreProfileGel {
		bandBonus := ampliconBandMassBonusC(p.Length, v.bandMassWeight())
		details.BandMassBonusC = bandBonus
		adjustment += bandBonus
	}
	details.AmpliconAdjustmentC = adjustment
	score += adjustment
	details.ScoreC = score
	return score
}

type primerPairVariant struct {
	Fwd string
	Rev string
}

type nnVariantScorer func(engine.Product) (bool, engine.Product, error)

func (v Score) primerPairVariants(p engine.Product) ([]primerPairVariant, bool, error) {
	policy := v.iupacThermoPolicy()
	if policy == thermo.IUPACThermoPolicyStrict {
		f := toUpperACGT(p.FwdPrimer)
		r := toUpperACGT(p.RevPrimer)
		if f == "" || r == "" {
			return nil, false, fmt.Errorf("NN thermodynamics with --iupac-thermo-policy strict requires A/C/G/T primers")
		}
		return []primerPairVariant{{Fwd: f, Rev: r}}, false, nil
	}

	maxExp := v.iupacThermoMaxExpansions()
	fwdExp, fwdCapped, err := thermo.ExpandIUPAC(p.FwdPrimer, maxExp)
	if err != nil {
		return nil, false, fmt.Errorf("forward primer IUPAC expansion: %w", err)
	}
	revExp, revCapped, err := thermo.ExpandIUPAC(p.RevPrimer, maxExp)
	if err != nil {
		return nil, false, fmt.Errorf("reverse primer IUPAC expansion: %w", err)
	}
	out := make([]primerPairVariant, 0, minInt(maxExp, len(fwdExp)*len(revExp)))
	capped := fwdCapped || revCapped
	for _, f := range fwdExp {
		for _, r := range revExp {
			if len(out) >= maxExp {
				return out, true, nil
			}
			out = append(out, primerPairVariant{Fwd: f, Rev: r})
		}
	}
	return out, capped, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func iupacVariantLabel(fwd, rev string) string {
	return "fwd=" + fwd + ";rev=" + rev
}

func thermoVariantSummary(p engine.Product) engine.ThermoVariant {
	out := engine.ThermoVariant{
		FwdPrimer: p.FwdPrimer,
		RevPrimer: p.RevPrimer,
		ScoreC:    p.Score,
	}
	if p.Thermo != nil {
		out.ScoreC = p.Thermo.ScoreC
		out.BaseScoreC = p.Thermo.BaseScoreC
		out.StructurePenaltyC = p.Thermo.StructurePenaltyC
		out.LimitingSide = p.Thermo.LimitingSide
		out.FwdTmC = p.Thermo.Fwd.TmC
		out.RevTmC = p.Thermo.Rev.TmC
		out.FwdMarginC = p.Thermo.Fwd.AnnealMarginC
		out.RevMarginC = p.Thermo.Rev.AnnealMarginC
	}
	return out
}

func copyThermoDetails(src *engine.ThermoDetails) *engine.ThermoDetails {
	if src == nil {
		return nil
	}
	out := *src
	if src.IUPACVariants != nil {
		out.IUPACVariants = append([]engine.ThermoVariant(nil), src.IUPACVariants...)
	}
	return &out
}

func averageIUPACProducts(scored []engine.Product) engine.Product {
	out := scored[0]
	out.Thermo = copyThermoDetails(scored[0].Thermo)
	n := float64(len(scored))
	out.Score = 0
	if out.Thermo != nil {
		out.Thermo.ScoreC = 0
		out.Thermo.BaseScoreC = 0
		out.Thermo.AmpliconAdjustmentC = 0
		out.Thermo.ExtensionLogit = 0
		out.Thermo.ExtensionBonusC = 0
		out.Thermo.LengthPenaltyC = 0
		out.Thermo.BandMassBonusC = 0
		out.Thermo.StructurePenaltyC = 0
		out.Thermo.Fwd.TmC = 0
		out.Thermo.Rev.TmC = 0
		out.Thermo.Fwd.AnnealMarginC = 0
		out.Thermo.Rev.AnnealMarginC = 0
		out.Thermo.Fwd.DeltaGAtAnnealKcal = 0
		out.Thermo.Rev.DeltaGAtAnnealKcal = 0
	}
	for _, p := range scored {
		out.Score += p.Score
		if out.Thermo != nil && p.Thermo != nil {
			out.Thermo.ScoreC += p.Thermo.ScoreC
			out.Thermo.BaseScoreC += p.Thermo.BaseScoreC
			out.Thermo.AmpliconAdjustmentC += p.Thermo.AmpliconAdjustmentC
			out.Thermo.ExtensionLogit += p.Thermo.ExtensionLogit
			out.Thermo.ExtensionBonusC += p.Thermo.ExtensionBonusC
			out.Thermo.LengthPenaltyC += p.Thermo.LengthPenaltyC
			out.Thermo.BandMassBonusC += p.Thermo.BandMassBonusC
			out.Thermo.StructurePenaltyC += p.Thermo.StructurePenaltyC
			out.Thermo.Fwd.TmC += p.Thermo.Fwd.TmC
			out.Thermo.Rev.TmC += p.Thermo.Rev.TmC
			out.Thermo.Fwd.AnnealMarginC += p.Thermo.Fwd.AnnealMarginC
			out.Thermo.Rev.AnnealMarginC += p.Thermo.Rev.AnnealMarginC
			out.Thermo.Fwd.DeltaGAtAnnealKcal += p.Thermo.Fwd.DeltaGAtAnnealKcal
			out.Thermo.Rev.DeltaGAtAnnealKcal += p.Thermo.Rev.DeltaGAtAnnealKcal
		}
	}
	out.Score /= n
	if out.Thermo != nil {
		out.Thermo.ScoreC /= n
		out.Thermo.BaseScoreC /= n
		out.Thermo.AmpliconAdjustmentC /= n
		out.Thermo.ExtensionLogit /= n
		out.Thermo.ExtensionBonusC /= n
		out.Thermo.LengthPenaltyC /= n
		out.Thermo.BandMassBonusC /= n
		out.Thermo.StructurePenaltyC /= n
		out.Thermo.Fwd.TmC /= n
		out.Thermo.Rev.TmC /= n
		out.Thermo.Fwd.AnnealMarginC /= n
		out.Thermo.Rev.AnnealMarginC /= n
		out.Thermo.Fwd.DeltaGAtAnnealKcal /= n
		out.Thermo.Rev.DeltaGAtAnnealKcal /= n
		out.Thermo.LimitingSide = "mean"
	}
	return out
}

func annotateIUPACThermo(p *engine.Product, policy string, count int, capped bool, effective string, variants []engine.ThermoVariant) {
	if p.Thermo == nil {
		return
	}
	p.Thermo.IUPACThermoPolicy = policy
	p.Thermo.IUPACExpansionCount = count
	p.Thermo.IUPACExpansionCapped = capped
	p.Thermo.IUPACEffectiveVariant = effective
	p.Thermo.IUPACPolicy = "iupac-thermo-" + policy
	if policy == thermo.IUPACThermoPolicyEnumerate {
		p.Thermo.IUPACVariants = variants
	}
}

func (v Score) visitNNWithIUPAC(p engine.Product, scorer nnVariantScorer) (bool, engine.Product, error) {
	variants, capped, err := v.primerPairVariants(p)
	if err != nil {
		return false, p, err
	}
	if len(variants) == 0 {
		return false, p, fmt.Errorf("IUPAC thermo expansion produced no concrete primer variants")
	}
	policy := v.iupacThermoPolicy()
	scored := make([]engine.Product, 0, len(variants))
	summaries := make([]engine.ThermoVariant, 0, len(variants))
	for _, variant := range variants {
		q := p
		q.FwdPrimer = variant.Fwd
		q.RevPrimer = variant.Rev
		ok, got, err := scorer(q)
		if err != nil {
			return false, p, err
		}
		if !ok {
			continue
		}
		scored = append(scored, got)
		summaries = append(summaries, thermoVariantSummary(got))
	}
	if len(scored) == 0 {
		return false, p, nil
	}

	bestIdx := 0
	switch policy {
	case thermo.IUPACThermoPolicyBest:
		for i := 1; i < len(scored); i++ {
			if scored[i].Score > scored[bestIdx].Score {
				bestIdx = i
			}
		}
		out := scored[bestIdx]
		annotateIUPACThermo(&out, policy, len(scored), capped, iupacVariantLabel(out.FwdPrimer, out.RevPrimer), summaries)
		return true, out, nil
	case thermo.IUPACThermoPolicyMean, thermo.IUPACThermoPolicyEnumerate:
		out := averageIUPACProducts(scored)
		effective := "mean"
		if policy == thermo.IUPACThermoPolicyEnumerate {
			effective = "enumerate"
		}
		annotateIUPACThermo(&out, policy, len(scored), capped, effective, summaries)
		return true, out, nil
	default:
		// worst is the default and the most conservative assay-design behavior.
		for i := 1; i < len(scored); i++ {
			if scored[i].Score < scored[bestIdx].Score {
				bestIdx = i
			}
		}
		out := scored[bestIdx]
		annotateIUPACThermo(&out, policy, len(scored), capped, iupacVariantLabel(out.FwdPrimer, out.RevPrimer), summaries)
		return true, out, nil
	}
}

func (v Score) visitNNDuplexV1Strict(p engine.Product) (bool, engine.Product, error) {
	fwd, revEnd, score, limitingSide, cond, err := v.scoreNNDuplexComponents(p)
	if err != nil {
		return false, p, err
	}
	details := nnThermoDetails(thermomodel.NNDuplexV1, cond, fwd, revEnd, score, limitingSide)
	score = v.applyAmpliconProfile(p, details, score)
	p.Score = score
	p.Thermo = details
	return true, p, nil
}

func (v Score) visitNNDuplexV1(p engine.Product) (bool, engine.Product, error) {
	return v.visitNNWithIUPAC(p, v.visitNNDuplexV1Strict)
}

func structureFromResult(src thermo.StructureResult, penaltyC float64) *engine.ThermoStructure {
	return structureFromResultWithLabels(src, penaltyC, "", "")
}

func structureFromResultWithLabels(src thermo.StructureResult, penaltyC float64, queryA, queryB string) *engine.ThermoStructure {
	if src.StemLen == 0 {
		return nil
	}
	return &engine.ThermoStructure{
		Kind:                        src.Kind,
		Model:                       src.Model,
		QueryA:                      queryA,
		QueryB:                      queryB,
		DeltaGAtAnnealKcal:          src.DeltaGAtAnnealKcal,
		TmC:                         src.TmC,
		AnnealMarginC:               src.AnnealMarginC,
		StemLen:                     src.StemLen,
		LoopLen:                     src.LoopLen,
		AStart:                      src.AStart,
		AEnd:                        src.AEnd,
		BStart:                      src.BStart,
		BEnd:                        src.BEnd,
		ThreePrimeAnchored:          src.ThreePrimeAnchored,
		BothThreePrimeAnchor:        src.BothThreePrimeAnchor,
		SegmentCount:                src.SegmentCount,
		BulgeCount:                  src.BulgeCount,
		InternalLoopCount:           src.InternalLoopCount,
		DanglingEndCount:            src.DanglingEndCount,
		LoopPenaltyKcal:             src.LoopPenaltyKcal,
		BulgePenaltyKcal:            src.BulgePenaltyKcal,
		InternalLoopPenaltyKcal:     src.InternalLoopPenaltyKcal,
		StructureDanglingDeltaGKcal: src.DanglingAdjustmentKcal,
		PenaltyC:                    penaltyC,
	}
}

func structureCompetitionPenaltyC(src thermo.StructureResult, binding engine.ThermoEndpoint) float64 {
	if src.StemLen == 0 || math.IsNaN(src.DeltaGAtAnnealKcal) || math.IsInf(src.DeltaGAtAnnealKcal, 0) {
		return 0
	}
	// Positive when the structure is close enough to compete with the relevant
	// primer-template endpoint at annealing temperature. 3' anchored dimers get a
	// larger competition window because they can seed extension.
	windowKcal := 1.0
	if src.Kind != thermo.StructureHairpin && src.ThreePrimeAnchored {
		windowKcal = 2.0
	}
	if src.BothThreePrimeAnchor {
		windowKcal = 3.0
	}
	competitiveKcal := binding.DeltaGAtAnnealKcal - src.DeltaGAtAnnealKcal + windowKcal
	if competitiveKcal <= 0 {
		return 0
	}
	denom := absFiniteOrFallback(binding.EffectiveDenomCalK, 200.0)
	penalty := competitiveKcal * 1000.0 / denom
	if math.IsNaN(penalty) || math.IsInf(penalty, 0) || penalty < 0 {
		return 0
	}
	if penalty > 30 {
		return 30
	}
	return penalty
}

func chooseWorseStructure(cur, cand *engine.ThermoStructure) *engine.ThermoStructure {
	if cand == nil || cand.PenaltyC <= 0 {
		return cur
	}
	if cur == nil || cand.PenaltyC > cur.PenaltyC {
		return cand
	}
	if cand.PenaltyC == cur.PenaltyC && cand.DeltaGAtAnnealKcal < cur.DeltaGAtAnnealKcal {
		return cand
	}
	return cur
}

func (v Score) normalizePanelPrimers() []PrimerRef {
	out := make([]PrimerRef, 0, len(v.PanelPrimers))
	seen := map[string]struct{}{}
	maxExp := v.iupacThermoMaxExpansions()
	for _, ref := range v.PanelPrimers {
		id := strings.TrimSpace(ref.ID)
		if id == "" {
			id = strings.ToUpper(ref.Seq)
		}
		expanded, _, err := thermo.ExpandIUPAC(ref.Seq, maxExp)
		if err != nil {
			continue
		}
		for _, seq := range expanded {
			key := seq
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			label := id
			if len(expanded) > 1 {
				label = id + "[" + seq + "]"
			}
			out = append(out, PrimerRef{ID: label, Seq: seq})
		}
	}
	return out
}

func samePrimerSeq(a, b string) bool {
	a = toUpperACGT(a)
	b = toUpperACGT(b)
	return a != "" && a == b
}

type panelCrossDimerHit struct {
	Result    thermo.StructureResult
	PenaltyC  float64
	BurdenC   float64
	Count     int
	QueryID   string
	PartnerID string
}

func (v Score) bestPanelCrossDimer(fwdPrimer, revPrimer string, fwd, revEnd engine.ThermoEndpoint, cond thermo.Conditions) panelCrossDimerHit {
	if len(v.PanelPrimers) == 0 {
		return panelCrossDimerHit{}
	}
	queries := []struct {
		ID      string
		Seq     string
		Binding engine.ThermoEndpoint
	}{
		{ID: "fwd", Seq: fwdPrimer, Binding: fwd},
		{ID: "rev", Seq: revPrimer, Binding: revEnd},
	}
	panel := v.normalizePanelPrimers()
	seen := map[string]struct{}{}
	var best panelCrossDimerHit
	for _, q := range queries {
		qSeq := toUpperACGT(q.Seq)
		if qSeq == "" {
			continue
		}
		for _, partner := range panel {
			if samePrimerSeq(partner.Seq, fwdPrimer) || samePrimerSeq(partner.Seq, revPrimer) {
				continue
			}
			key := q.ID + "\x00" + qSeq + "\x00" + partner.ID + "\x00" + partner.Seq
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			res, ok, err := thermo.BestCrossDimerV2(qSeq, partner.Seq, thermo.DefaultStructureOptions(cond))
			if err != nil || !ok {
				continue
			}
			pen := structureCompetitionPenaltyC(res, q.Binding)
			if pen <= 0 {
				continue
			}
			best.Count++
			best.BurdenC += pen
			if pen > best.PenaltyC || (pen == best.PenaltyC && res.DeltaGAtAnnealKcal < best.Result.DeltaGAtAnnealKcal) {
				best.Result = res
				best.PenaltyC = pen
				best.QueryID = q.ID
				best.PartnerID = partner.ID
			}
		}
	}
	return best
}

func (v Score) visitNNStructureV1Strict(p engine.Product) (bool, engine.Product, error) {
	fwd, revEnd, baseScore, limitingSide, cond, err := v.scoreNNDuplexComponents(p)
	if err != nil {
		return false, p, err
	}

	f := toUpperACGT(p.FwdPrimer)
	r := toUpperACGT(p.RevPrimer)
	scale := v.StructScale
	if scale < 0 {
		scale = 0
	}

	details := nnThermoDetails(thermomodel.NNStructureV1, cond, fwd, revEnd, baseScore, limitingSide)
	details.StructurePolicy = structurePolicyNNStemLoopV2
	details.BaseScoreC = baseScore

	totalPenalty := 0.0
	if v.StructHairpin {
		if hp, ok, err := thermo.BestHairpinV2(f, thermo.DefaultStructureOptions(cond)); err == nil && ok {
			pen := structureCompetitionPenaltyC(hp, fwd)
			details.WorstHairpin = chooseWorseStructure(details.WorstHairpin, structureFromResultWithLabels(hp, pen, "fwd", "fwd"))
			totalPenalty += pen
		}
		if hp, ok, err := thermo.BestHairpinV2(r, thermo.DefaultStructureOptions(cond)); err == nil && ok {
			pen := structureCompetitionPenaltyC(hp, revEnd)
			details.WorstHairpin = chooseWorseStructure(details.WorstHairpin, structureFromResultWithLabels(hp, pen, "rev", "rev"))
			totalPenalty += pen
		}
	}

	if v.StructDimer {
		if sd, ok, err := thermo.BestSelfDimerV2(f, thermo.DefaultStructureOptions(cond)); err == nil && ok {
			pen := structureCompetitionPenaltyC(sd, fwd)
			details.WorstSelfDimer = chooseWorseStructure(details.WorstSelfDimer, structureFromResultWithLabels(sd, pen, "fwd", "fwd"))
			totalPenalty += pen
		}
		if sd, ok, err := thermo.BestSelfDimerV2(r, thermo.DefaultStructureOptions(cond)); err == nil && ok {
			pen := structureCompetitionPenaltyC(sd, revEnd)
			details.WorstSelfDimer = chooseWorseStructure(details.WorstSelfDimer, structureFromResultWithLabels(sd, pen, "rev", "rev"))
			totalPenalty += pen
		}
		if xd, ok, err := thermo.BestCrossDimerV2(f, r, thermo.DefaultStructureOptions(cond)); err == nil && ok {
			pen := math.Max(structureCompetitionPenaltyC(xd, fwd), structureCompetitionPenaltyC(xd, revEnd))
			details.CrossDimer = chooseWorseStructure(details.CrossDimer, structureFromResultWithLabels(xd, pen, "fwd", "rev"))
			totalPenalty += pen
		}
		panel := v.bestPanelCrossDimer(f, r, fwd, revEnd, cond)
		if panel.PenaltyC > 0 {
			details.PanelCrossDimer = structureFromResultWithLabels(panel.Result, panel.PenaltyC, panel.QueryID, panel.PartnerID)
			details.PanelCrossDimerPenaltyC = panel.PenaltyC
			details.PanelCrossDimerBurdenC = panel.BurdenC
			details.PanelCrossDimerCount = panel.Count
			totalPenalty += panel.PenaltyC
		}
	}

	totalPenalty *= scale
	if details.WorstHairpin != nil {
		details.WorstHairpin.PenaltyC *= scale
	}
	if details.WorstSelfDimer != nil {
		details.WorstSelfDimer.PenaltyC *= scale
	}
	if details.CrossDimer != nil {
		details.CrossDimer.PenaltyC *= scale
	}
	if details.PanelCrossDimer != nil {
		details.PanelCrossDimer.PenaltyC *= scale
		details.PanelCrossDimerPenaltyC *= scale
		details.PanelCrossDimerBurdenC *= scale
	}

	score := baseScore - totalPenalty
	details.StructurePenaltyC = totalPenalty
	score = v.applyAmpliconProfile(p, details, score)
	p.Score = score
	p.Thermo = details
	return true, p, nil
}

func (v Score) visitNNStructureV1(p engine.Product) (bool, engine.Product, error) {
	return v.visitNNWithIUPAC(p, v.visitNNStructureV1Strict)
}

type scoredProbeVariant struct {
	Variant string
	Result  thermo.ImperfectDuplexResult
}

func probeTarget3to5(strand, site string) string {
	site = strings.ToUpper(site)
	if strand == "-" {
		return rev(site)
	}
	return comp5to3(site)
}

func probeVariantDetails(base engine.ProbeThermoDetails, chosen scoredProbeVariant, count int, capped bool, effective string) engine.ProbeThermoDetails {
	res := chosen.Result
	base.IUPACExpansionCount = count
	base.IUPACExpansionCapped = capped
	base.IUPACEffectiveVariant = effective
	base.TmC = res.TmC
	base.AnnealMarginC = res.AnnealMarginC
	base.DeltaGAtAnnealKcal = res.DeltaGAtAnnealKcal
	base.MismatchPenaltyC = res.MismatchPenaltyC
	base.MismatchDeltaGKcal = res.DeltaGPenaltyKcal
	base.MismatchCount = res.MismatchCount
	base.MismatchFallbackCount = res.HeuristicFallbackCount + res.DefaultFallbackCount
	base.MismatchTripletCount = res.TripletTmCount + res.TripletDeltaGCount
	base.MismatchPolicy = res.MismatchPolicy
	base.HasNonWatsonCrick = res.HasNonWatsonCrick
	base.UsedHeuristicAdjust = res.UsedHeuristicAdjust
	return base
}

func meanProbeDetails(base engine.ProbeThermoDetails, scored []scoredProbeVariant, capped bool, effective string) engine.ProbeThermoDetails {
	base.IUPACExpansionCount = len(scored)
	base.IUPACExpansionCapped = capped
	base.IUPACEffectiveVariant = effective
	if len(scored) == 0 {
		return base
	}
	n := float64(len(scored))
	for _, s := range scored {
		res := s.Result
		base.TmC += res.TmC
		base.AnnealMarginC += res.AnnealMarginC
		base.DeltaGAtAnnealKcal += res.DeltaGAtAnnealKcal
		base.MismatchPenaltyC += res.MismatchPenaltyC
		base.MismatchDeltaGKcal += res.DeltaGPenaltyKcal
		if res.MismatchCount > base.MismatchCount {
			base.MismatchCount = res.MismatchCount
		}
		fallbacks := res.HeuristicFallbackCount + res.DefaultFallbackCount
		if fallbacks > base.MismatchFallbackCount {
			base.MismatchFallbackCount = fallbacks
		}
		triplets := res.TripletTmCount + res.TripletDeltaGCount
		if triplets > base.MismatchTripletCount {
			base.MismatchTripletCount = triplets
		}
		if res.HasNonWatsonCrick {
			base.HasNonWatsonCrick = true
		}
		if res.UsedHeuristicAdjust {
			base.UsedHeuristicAdjust = true
		}
	}
	base.TmC /= n
	base.AnnealMarginC /= n
	base.DeltaGAtAnnealKcal /= n
	base.MismatchPenaltyC /= n
	base.MismatchDeltaGKcal /= n
	if base.MismatchFallbackCount > 0 {
		base.MismatchPolicy = thermo.MismatchPolicyImperfectHeuristicFallback
	} else if base.MismatchCount > 0 {
		base.MismatchPolicy = thermo.MismatchPolicyImperfectV1
	} else {
		base.MismatchPolicy = thermo.MismatchPolicyPerfect
	}
	return base
}

func (v Score) scoreProbeThermoDetails(p engine.Product) (engine.ProbeThermoDetails, error) {
	probeSeq := strings.ToUpper(strings.TrimSpace(v.ProbeSeq))
	details := engine.ProbeThermoDetails{
		Name:              v.probeName(),
		Seq:               probeSeq,
		ScoreMode:         v.probeScoreMode(),
		MinMarginC:        v.ProbeMinMarginC,
		IUPACThermoPolicy: v.iupacThermoPolicy(),
	}
	ann := probeanno.AnnotateAmplicon(p.Seq, probeSeq, v.ProbeMaxMM)
	details.Found = ann.Found
	details.Strand = ann.Strand
	details.Pos = ann.Pos
	details.MM = ann.MM
	details.Site = ann.Site
	if !ann.Found {
		return details, nil
	}

	policy := v.iupacThermoPolicy()
	expanded := []string{probeSeq}
	capped := false
	if policy == thermo.IUPACThermoPolicyStrict {
		if !thermo.IsStrictACGT(probeSeq) {
			return details, fmt.Errorf("--probe with --iupac-thermo-policy strict requires A/C/G/T probe sequence")
		}
	} else {
		var err error
		expanded, capped, err = thermo.ExpandIUPAC(probeSeq, v.iupacThermoMaxExpansions())
		if err != nil {
			return details, fmt.Errorf("--probe %q: %v", probeSeq, err)
		}
	}
	if len(expanded) == 0 {
		return details, fmt.Errorf("--probe IUPAC expansion produced no concrete probe variants")
	}

	target := probeTarget3to5(ann.Strand, ann.Site)
	cond := v.conditions()
	scored := make([]scoredProbeVariant, 0, len(expanded))
	for _, variant := range expanded {
		res, err := thermo.ImperfectDuplexWithOptions(variant, target, cond, thermo.DefaultImperfectDuplexOptions())
		if err != nil {
			return details, err
		}
		scored = append(scored, scoredProbeVariant{Variant: variant, Result: res})
	}

	bestIdx := 0
	switch policy {
	case thermo.IUPACThermoPolicyBest:
		for i := 1; i < len(scored); i++ {
			if scored[i].Result.AnnealMarginC > scored[bestIdx].Result.AnnealMarginC {
				bestIdx = i
			}
		}
		return probeVariantDetails(details, scored[bestIdx], len(scored), capped, scored[bestIdx].Variant), nil
	case thermo.IUPACThermoPolicyMean:
		return meanProbeDetails(details, scored, capped, "mean"), nil
	case thermo.IUPACThermoPolicyEnumerate:
		return meanProbeDetails(details, scored, capped, "enumerate"), nil
	default:
		// worst is the default for assay-design conservatism.
		for i := 1; i < len(scored); i++ {
			if scored[i].Result.AnnealMarginC < scored[bestIdx].Result.AnnealMarginC {
				bestIdx = i
			}
		}
		return probeVariantDetails(details, scored[bestIdx], len(scored), capped, scored[bestIdx].Variant), nil
	}
}

func (v Score) applyProbeThermo(p engine.Product) (bool, engine.Product, error) {
	if !v.probeThermoEnabled() || p.Thermo == nil {
		return true, p, nil
	}
	details, err := v.scoreProbeThermoDetails(p)
	if err != nil {
		return false, p, err
	}

	oldScore := p.Score
	switch details.ScoreMode {
	case probeScoreModeGate:
		if !details.Found {
			details.GatePenaltyC = PROBE_NOT_FOUND_PEN
			p.Thermo.Probe = &details
			return false, p, nil
		}
		if details.AnnealMarginC < details.MinMarginC {
			details.GatePenaltyC = details.MinMarginC - details.AnnealMarginC
			p.Thermo.Probe = &details
			return false, p, nil
		}
	case probeScoreModeBlend:
		if !details.Found {
			p.Score -= PROBE_NOT_FOUND_PEN
		} else {
			w := v.probeWeight()
			limiting := math.Min(oldScore, details.AnnealMarginC)
			p.Score = (1-w)*oldScore + w*limiting
		}
		details.ScoreContributionC = p.Score - oldScore
	case probeScoreModeAnnotate:
		// Deliberately leave the primer-derived score untouched.
	}
	if p.Thermo != nil {
		p.Thermo.Probe = &details
		p.Thermo.ScoreC = p.Score
	}
	return true, p, nil
}

// Visit implements the appcore visitor for ipcr-thermo.
// It computes a small penalty for the forward end (and conservatively for the reverse end),
// then sets Score = -penalty so that higher is better.
func (v Score) Visit(p engine.Product) (bool, engine.Product, error) {
	mode := v.Model
	if mode == "" {
		mode = thermomodel.Default()
	}
	var (
		ok  bool
		out engine.Product
		err error
	)
	switch mode {
	case thermomodel.LegacyHeuristic:
		return v.visitLegacyHeuristic(p)
	case thermomodel.NNDuplexV1:
		ok, out, err = v.visitNNDuplexV1(p)
	case thermomodel.NNStructureV1:
		ok, out, err = v.visitNNStructureV1(p)
	default:
		return false, p, fmt.Errorf("thermo model %q is not implemented", mode)
	}
	if err != nil || !ok {
		return ok, out, err
	}
	return v.applyProbeThermo(out)
}

func (v Score) visitLegacyHeuristic(p engine.Product) (bool, engine.Product, error) {
	// Default conservative fixed denominator
	denomF, denomR := 200.0, 200.0

	ssOn := v.SingleStranded || singleStrandedMode()
	pen := 0.0

	// Forward end (use leftmost |F| bases of the amplicon)
	if f := toUpperACGT(p.FwdPrimer); f != "" && len(p.Seq) >= len(f) {
		if v.UseAutoDenom {
			denomF = v.denomForPrimer(f)
		}
		left := p.Seq[:len(f)]
		t3 := comp5to3(left)
		pen += alignPenaltyC_contextualD_ss(f, t3, v.AllowIndels, denomF, ssOn)
	}

	// Reverse end (conservative: compare primer vs complement of rightmost |R| bases)
	if r := toUpperACGT(p.RevPrimer); r != "" && len(p.Seq) >= len(r) {
		if v.UseAutoDenom {
			denomR = v.denomForPrimer(r)
		}
		right := p.Seq[len(p.Seq)-len(r):]
		t3 := comp5to3(right)
		pen += alignPenaltyC_contextualD_ss(r, t3, v.AllowIndels, denomR, ssOn)
	}

	// Final score: higher is better.
	p.Score = -pen
	return true, p, nil
}
