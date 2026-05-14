// internal/thermovisitors/score.go
package thermovisitors

import (
	"fmt"
	"ipcr-core/engine"
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
	mismatchPolicyNNPerfect         = "nn-perfect"
	mismatchPolicyHeuristicFallback = "heuristic-ddg-fallback"
	mismatchPolicyMixed             = "nn-perfect-or-heuristic-ddg-fallback"
)

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
	StructScale    float64

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
		HasNonWatsonCrick:   hasNonWC,
		UsedHeuristicAdjust: heuristic,
	}
}

func (v Score) scoreNNDuplexEndpoint(side, primer5to3, target3to5 string) (engine.ThermoEndpoint, error) {
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
	if actual, err := thermo.PerfectDuplex(primer, target, cond); err == nil {
		return endpointFromDuplex(side, actual, 0, mismatchPolicyNNPerfect, false, false), nil
	}

	// Mismatched or N-containing target: anchor the thermodynamics in the perfect
	// primer/complement duplex, then apply the currently curated mismatch layer.
	// Triplet ΔTm/ΔΔG overrides are used when present; otherwise the existing
	// pair/context heuristic is used explicitly and reported in the output.
	perfectTarget := comp5to3(primer)
	base, err := thermo.PerfectDuplex(primer, perfectTarget, cond)
	if err != nil {
		return engine.ThermoEndpoint{}, err
	}
	denom := absFiniteOrFallback(base.EffectiveDenomCalK, 200.0)
	ssOn := v.SingleStranded || singleStrandedMode()
	penaltyC := alignPenaltyC_contextualD_ss(primer, target, v.AllowIndels, denom, ssOn)
	deltaGPenalty := penaltyC * denom / 1000.0

	adjusted := base
	adjusted.TmC = base.TmC - penaltyC
	adjusted.AnnealMarginC = adjusted.TmC - cond.AnnealC
	adjusted.DeltaGAtAnnealKcal = base.DeltaGAtAnnealKcal + deltaGPenalty
	adjusted.EffectiveDenomCalK = denom
	return endpointFromDuplex(side, adjusted, penaltyC, mismatchPolicyHeuristicFallback, true, true), nil
}

func (v Score) visitNNDuplexV1(p engine.Product) (bool, engine.Product, error) {
	f := toUpperACGT(p.FwdPrimer)
	r := toUpperACGT(p.RevPrimer)
	if f == "" || r == "" {
		return false, p, fmt.Errorf("nn-duplex-v1 requires A/C/G/T primers; degenerate/IUPAC primer scoring is not implemented yet")
	}
	if len(p.Seq) < len(f) || len(p.Seq) < len(r) {
		return false, p, fmt.Errorf("nn-duplex-v1 requires product sequence long enough for both primer sites")
	}

	leftSite := toUpperACGTAllowN(p.Seq[:len(f)])
	rightSite := toUpperACGTAllowN(p.Seq[len(p.Seq)-len(r):])
	if leftSite == "" || rightSite == "" {
		return false, p, fmt.Errorf("nn-duplex-v1 requires A/C/G/T/N product sequence at primer sites")
	}

	// The left site is in the same 5'→3' orientation as the forward primer.
	// The right site is the reference-strand reverse complement of the reverse
	// primer, so reversing the site gives the primer-aligned target strand 3'→5'.
	fwdTarget3 := comp5to3(leftSite)
	revTarget3 := rev(rightSite)

	fwd, err := v.scoreNNDuplexEndpoint("fwd", f, fwdTarget3)
	if err != nil {
		return false, p, err
	}
	revEnd, err := v.scoreNNDuplexEndpoint("rev", r, revTarget3)
	if err != nil {
		return false, p, err
	}

	limitingSide := "fwd"
	score := fwd.AnnealMarginC
	if revEnd.AnnealMarginC < score {
		score = revEnd.AnnealMarginC
		limitingSide = "rev"
	}

	cond := v.conditions()
	p.Score = score
	p.Thermo = &engine.ThermoDetails{
		Model:          thermomodel.NNDuplexV1.String(),
		SaltModel:      cond.SaltModel.String(),
		AnnealTempC:    cond.AnnealC,
		IUPACPolicy:    iupacPolicyStrictACGT,
		MismatchPolicy: mismatchPolicyMixed,
		ScoreC:         score,
		LimitingSide:   limitingSide,
		Fwd:            fwd,
		Rev:            revEnd,
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
	switch mode {
	case thermomodel.LegacyHeuristic:
		return v.visitLegacyHeuristic(p)
	case thermomodel.NNDuplexV1:
		return v.visitNNDuplexV1(p)
	default:
		return false, p, fmt.Errorf("thermo model %q is not implemented", mode)
	}
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
