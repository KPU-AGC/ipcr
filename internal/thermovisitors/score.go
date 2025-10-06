package thermovisitors

import (
	"math"
	"strings"
	"sync"
	"unicode"

	"ipcr-core/engine"
	"ipcr-core/oligo"
	"ipcr-core/thermo"
	"ipcr-core/thermoaddons"
)

/*
ipcr-thermo scoring (thermo-only):

- Contextual ΔTm for mismatches + position-aware single-nt gaps
- Probe-aware blend (unchanged)
- NEW: soft-knee length penalty, hairpin/dimer (3'-weighted), and a kinetics-lite extension term
  via ExtensionProb(Logit), all added only in this thermo path.
*/

type Score struct {
	AnnealTempC  float64
	Na_M         float64
	PrimerConc_M float64
	AllowIndels  bool
	LengthBiasOn bool

	// Probe
	ProbeSeq    string
	ProbeMaxMM  int
	ProbeWeight float64 // [0..1]

	// NEW thermoaddons knobs (thermo-only)
	ExtAlpha      float64 // slope for ExtensionProb
	LenKneeBP     int
	LenSteep      float64
	LenMaxPenC    float64
	StructHairpin bool
	StructDimer   bool
	StructScale   float64
	BindWeight    float64 // keep available for future occupancy; not used heavily yet
	ExtWeight     float64 // weight for extension logit term

	denomOnce sync.Once
	denomInit map[string]float64 // primer -> denominator for ΔΔG→ΔTm
}

const (
	K5 = 3
	K3 = 3

	PEN_GAP_1NT = 12.0

	MIN_USEFUL_TMC = -200.0
	MAX_USEFUL_TMC = 200.0

	PROBE_NOT_FOUND_PEN = 10.0
)

func (v *Score) Visit(p engine.Product) (bool, engine.Product, error) {
	if p.Seq == "" || p.FwdPrimer == "" || p.RevPrimer == "" {
		return true, p, nil
	}
	alen, blen := len(p.FwdPrimer), len(p.RevPrimer)
	if alen == 0 || blen == 0 || len(p.Seq) < alen+blen {
		return true, p, nil
	}
	v.denomOnce.Do(func() { v.denomInit = make(map[string]float64, 8) })

	// ends of amplicon (+ strand)
	leftPlus := toUpperACGTAllowN(p.Seq[:alen])
	rightPlus := toUpperACGTAllowN(p.Seq[len(p.Seq)-blen:])

	leftTgt3to5 := complement3to5AllowN(leftPlus)
	rightTgt3to5 := complement3to5AllowN(rightPlus)

	// Base Tm (perfect) for reference
	tmMaxF, _ := baseTmForPrimer(p.FwdPrimer, v.Na_M, v.PrimerConc_M)
	tmMaxR, _ := baseTmForPrimer(p.RevPrimer, v.Na_M, v.PrimerConc_M)

	// Effective denominators D per primer (for ΔΔG→ΔTm fallback)
	dF := v.effectiveDenominator(p.FwdPrimer)
	dR := v.effectiveDenominator(p.RevPrimer)

	// Contextual ΔTm penalties by DP, with position-aware gap & ΔΔG fallback
	deltaF := alignPenaltyC_contextualD(p.FwdPrimer, leftTgt3to5, v.AllowIndels, dF)
	deltaR := alignPenaltyC_contextualD(p.RevPrimer, rightTgt3to5, v.AllowIndels, dR)

	// Adjusted per-end Tm
	tmAdjF := tmMaxF - deltaF
	tmAdjR := tmMaxR - deltaR
	margPrimers := math.Min(tmAdjF, tmAdjR) - v.AnnealTempC

	// Probe margin (optional; unchanged)
	margProbe := math.Inf(1)
	if v.ProbeSeq != "" {
		dP := v.effectiveDenominator(v.ProbeSeq)
		margProbe = probeMarginC_D(p.Seq, v.ProbeSeq, v.ProbeMaxMM, v.Na_M, v.PrimerConc_M, dP) - v.AnnealTempC
		if math.IsInf(margProbe, 1) {
			margProbe = (tmAdjF - v.AnnealTempC) - PROBE_NOT_FOUND_PEN
		}
	}
	score := blendMargin(margPrimers, margProbe, clamp01(v.ProbeWeight))

	// NEW: kinetics-lite extension term (logit of ExtensionProb on the limiting margin)
	if v.ExtAlpha == 0 {
		v.ExtAlpha = 0.45
	}
	if v.ExtWeight == 0 {
		v.ExtWeight = 1.0
	}
	extLogit := thermoaddons.Logit(thermoaddons.ExtensionProb(score, v.ExtAlpha))
	score += v.ExtWeight * extLogit

	// NEW: soft-knee length penalty (°C-equivalent), subtract directly
	if v.LengthBiasOn {
		if v.LenKneeBP == 0 {
			v.LenKneeBP = 550
		}
		if v.LenSteep == 0 {
			v.LenSteep = 0.003
		}
		if v.LenMaxPenC == 0 {
			v.LenMaxPenC = 10.0
		}
		lenPen := thermoaddons.LengthPenalty(p.Length, v.LenKneeBP, v.LenSteep, v.LenMaxPenC)
		score -= lenPen
	}

	// NEW: secondary-structure penalties (3'-weighted heuristics), thermo-only
	if v.StructScale == 0 {
		v.StructScale = 1.0
	}
	if v.StructHairpin {
		score -= v.StructScale * (thermoaddons.HairpinPenalty(p.FwdPrimer) + thermoaddons.HairpinPenalty(p.RevPrimer))
	}
	if v.StructDimer {
		score -= v.StructScale * thermoaddons.DimerPenalty(p.FwdPrimer, p.RevPrimer)
	}

	// clamp absurd
	if score > MAX_USEFUL_TMC {
		score = MAX_USEFUL_TMC
	} else if score < -MAX_USEFUL_TMC {
		score = -MAX_USEFUL_TMC
	}
	p.Score = score
	return true, p, nil
}

func (v *Score) effectiveDenominator(primer string) float64 {
	if v.denomInit == nil {
		v.denomInit = make(map[string]float64, 8)
	}
	if d, ok := v.denomInit[primer]; ok {
		return d
	}
	primACGT := toUpperACGT(primer)
	if primACGT == "" || v.PrimerConc_M <= 0 || v.Na_M <= 0 {
		v.denomInit[primer] = 200.0
		return 200.0
	}
	tgt3to5 := complement3to5(primACGT)
	res, err := thermo.Tm(primACGT, tgt3to5, thermo.TmInput{CT: v.PrimerConc_M, Na: v.Na_M, X: 4})
	if err != nil {
		v.denomInit[primer] = 200.0
		return 200.0
	}
	D := res.DS_Na + thermo.Rcal*math.Log(v.PrimerConc_M/4.0)
	if D <= 0 {
		D = 200.0
	}
	v.denomInit[primer] = D
	return D
}

func probeMarginC_D(ampliconPlus, probe5to3 string, maxMM int, na, ct, denom float64) float64 {
	h := oligo.BestHit(ampliconPlus, probe5to3, maxMM)
	if !h.Found || len(h.Site) != len(probe5to3) {
		return math.Inf(1)
	}
	tmBase, _ := baseTmForPrimer(probe5to3, na, ct)
	site3to5 := complement3to5AllowN(h.Site)
	delta := alignPenaltyC_contextualD(probe5to3, site3to5, false, denom)
	return tmBase - delta
}

func alignPenaltyC_contextualD(primer5to3, tgt3to5 string, allowGap bool, denom float64) float64 {
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

	for i := 0; i <= n; i++ {
		for j := 0; j <= m; j++ {
			for g := 0; g <= gapAllowed; g++ {
				cur := dp[i][j][g]
				if cur >= INF {
					continue
				}
				if i < n && j < m {
					pen := 0.0
					if !wcACGT(P[i], T[j]) {
						p5, pC, p3 := at(P, i-1), P[i], at(P, i+1)
						t5, tC, t3 := at(T, j-1), T[j], at(T, j+1)
						if dTm, ok := thermo.LookupDeltaTm(p5, pC, p3, t5, tC, t3); ok {
							pen = dTm
						} else if dG, ok := thermo.LookupDeltaG(p5, pC, p3, t5, tC, t3); ok {
							pen = thermo.DeltaGToDeltaTm(dG, denom)
						} else {
							pen = 4.0
						}
						pen *= posMultiplier(i, n)
					}
					if cur+pen < dp[i+1][j+1][g] {
						dp[i+1][j+1][g] = cur + pen
					}
				}
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
	return best
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

// Base Tm against perfect complement; Wallace fallback.
func baseTmForPrimer(primer string, na, ct float64) (float64, bool) {
	primACGT := toUpperACGT(primer)
	if primACGT != "" && ct > 0 && na > 0 {
		tgt3to5 := complement3to5(primACGT)
		res, err := thermo.Tm(primACGT, tgt3to5, thermo.TmInput{CT: ct, Na: na, X: 4})
		if err == nil && res.TmC > MIN_USEFUL_TMC && res.TmC < MAX_USEFUL_TMC {
			return res.TmC, true
		}
	}
	a, c, g, t := 0, 0, 0, 0
	for _, r := range strings.ToUpper(primer) {
		switch r {
		case 'A':
			a++
		case 'C':
			c++
		case 'G':
			g++
		case 'T':
			t++
		}
	}
	return 2.0*float64(a+t) + 4.0*float64(c+g), false
}

// util
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
func complement3to5(plus5to3 string) string {
	out := make([]byte, len(plus5to3))
	for i := 0; i < len(plus5to3); i++ {
		switch plus5to3[i] {
		case 'A':
			out[i] = 'T'
		case 'C':
			out[i] = 'G'
		case 'G':
			out[i] = 'C'
		case 'T':
			out[i] = 'A'
		default:
			out[i] = 'N'
		}
	}
	return string(out)
}
func complement3to5AllowN(plus5to3 string) string {
	out := make([]byte, len(plus5to3))
	for i := 0; i < len(plus5to3); i++ {
		switch plus5to3[i] {
		case 'A':
			out[i] = 'T'
		case 'C':
			out[i] = 'G'
		case 'G':
			out[i] = 'C'
		case 'T':
			out[i] = 'A'
		case 'N':
			out[i] = 'N'
		default:
			out[i] = 'N'
		}
	}
	return string(out)
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
func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
func blendMargin(mPrimers, mProbe, w float64) float64 {
	if w <= 0 || math.IsInf(mProbe, 1) {
		return mPrimers
	}
	if w >= 1 {
		if mProbe < mPrimers {
			return mProbe
		}
		return mPrimers
	}
	minM := mPrimers
	if mProbe < minM {
		minM = mProbe
	}
	return (1.0-w)*mPrimers + w*minM
}
