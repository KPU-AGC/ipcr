// internal/thermovisitors/score.go
package thermovisitors

import (
	"math"
	"os"
	"strings"
	"unicode"

	"ipcr-core/engine"
	"ipcr-core/thermo"
	"ipcr-core/thermoaddons"
)

const (
	PEN_GAP_1NT         = 6.0
	MIN_USEFUL_TMC      = -10.0
	MAX_USEFUL_TMC      = 120.0
	K5                  = 3 // 5' end is harsher across first K5 bases
	K3                  = 3 // 3' end is harshest across last K3 bases
	PROBE_NOT_FOUND_PEN = 12.0
)

// Score is the thermo-scoring visitor config.
type Score struct {
	AnnealTempC    float64
	Na_M           float64
	PrimerConc_M   float64
	AllowIndels    bool
	LengthBiasOn   bool
	SingleStranded bool // env toggle actually used; keep for compatibility
	StructScale    float64
}

// Public helper used by tests/tools.
func (v *Score) Penalty(primer5to3, tgt3to5 string, denom float64) float64 {
	return alignPenaltyC_contextualD(primer5to3, tgt3to5, v.AllowIndels, denom)
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

// Env-based ssDNA toggle.
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

// ---------------- DP with contextual mismatch penalties ----------------
//
// denom = effective denominator D (cal/K/mol) used only for ΔΔG→ΔTm fallback.
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
				if cur >= INF {
					continue
				}
				// match / mismatch
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

	// ssDNA adjustments (env toggle)
	if singleStrandedMode() && n > 0 {
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

// Visit implements the appcore visitor for ipcr-thermo.
// It computes a small penalty for the forward end (and conservatively for the reverse end),
// then sets Score = -penalty so that higher is better.
func (v Score) Visit(p engine.Product) (bool, engine.Product, error) {
	const denom = 200.0 // safe, matches tests’ fallback

	pen := 0.0

	// Forward end (use leftmost |F| bases of the amplicon)
	if f := toUpperACGT(p.FwdPrimer); f != "" && len(p.Seq) >= len(f) {
		left := p.Seq[:len(f)]
		t3 := comp5to3(left)
		pen += alignPenaltyC_contextualD(f, t3, v.AllowIndels, denom)
	}

	// Reverse end (conservative: compare primer vs complement of rightmost |R| bases)
	if r := toUpperACGT(p.RevPrimer); r != "" && len(p.Seq) >= len(r) {
		right := p.Seq[len(p.Seq)-len(r):]
		t3 := comp5to3(right)
		pen += alignPenaltyC_contextualD(r, t3, v.AllowIndels, denom)
	}

	// Final score: higher is better.
	p.Score = -pen
	return true, p, nil
}
