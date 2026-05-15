package thermo

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

const (
	// MismatchPolicyPerfect identifies an all-Watson-Crick duplex.
	MismatchPolicyPerfect = "nn-perfect"

	// MismatchPolicyImperfectV1 identifies the first condition-aware imperfect
	// duplex model. It anchors on the perfect primer/complement NN duplex, then
	// applies context-aware mismatch terms at the configured annealing conditions.
	MismatchPolicyImperfectV1 = "nn-imperfect-v1"

	// MismatchPolicyImperfectHeuristicFallback identifies an imperfect-duplex
	// result that had to use the current pair/context ΔΔG fallback for at least
	// one mismatch because no curated triplet parameter was available.
	MismatchPolicyImperfectHeuristicFallback = "nn-imperfect-v1-with-heuristic-ddg-fallback"

	// MismatchPolicyImperfectDefaultFallback identifies an imperfect-duplex result
	// that encountered an unsupported mismatch context and used the conservative
	// default ΔTm fallback.
	MismatchPolicyImperfectDefaultFallback = "nn-imperfect-v1-with-default-fallback"
)

const (
	defaultFivePrimeMismatchWindow  = 3
	defaultThreePrimeMismatchWindow = 3
	defaultFivePrimeMismatchWeight  = 1.5
	defaultThreePrimeMismatchWeight = 2.0
	defaultMismatchDeltaTmC         = 4.0
)

// ImperfectDuplexOptions controls the positional weighting used by the current
// imperfect-duplex model. Positions are primer 5'→3' indexes.
type ImperfectDuplexOptions struct {
	FivePrimeWindow        int
	ThreePrimeWindow       int
	FivePrimeMultiplier    float64
	ThreePrimeMultiplier   float64
	DefaultMismatchDeltaTm float64
}

// DefaultImperfectDuplexOptions returns the positional weighting historically
// used by ipcr-thermo: 3' mismatches are strongest, then 5' mismatches, then
// internal mismatches.
func DefaultImperfectDuplexOptions() ImperfectDuplexOptions {
	return ImperfectDuplexOptions{
		FivePrimeWindow:        defaultFivePrimeMismatchWindow,
		ThreePrimeWindow:       defaultThreePrimeMismatchWindow,
		FivePrimeMultiplier:    defaultFivePrimeMismatchWeight,
		ThreePrimeMultiplier:   defaultThreePrimeMismatchWeight,
		DefaultMismatchDeltaTm: defaultMismatchDeltaTmC,
	}
}

func (o ImperfectDuplexOptions) withDefaults() ImperfectDuplexOptions {
	d := DefaultImperfectDuplexOptions()
	if o.FivePrimeWindow < 0 {
		o.FivePrimeWindow = 0
	} else if o.FivePrimeWindow == 0 {
		o.FivePrimeWindow = d.FivePrimeWindow
	}
	if o.ThreePrimeWindow < 0 {
		o.ThreePrimeWindow = 0
	} else if o.ThreePrimeWindow == 0 {
		o.ThreePrimeWindow = d.ThreePrimeWindow
	}
	if o.FivePrimeMultiplier == 0 {
		o.FivePrimeMultiplier = d.FivePrimeMultiplier
	}
	if o.ThreePrimeMultiplier == 0 {
		o.ThreePrimeMultiplier = d.ThreePrimeMultiplier
	}
	if o.DefaultMismatchDeltaTm == 0 {
		o.DefaultMismatchDeltaTm = d.DefaultMismatchDeltaTm
	}
	return o
}

func (o ImperfectDuplexOptions) posMultiplier(i, n int) float64 {
	o = o.withDefaults()
	if o.ThreePrimeWindow > 0 && i >= n-o.ThreePrimeWindow {
		return o.ThreePrimeMultiplier
	}
	if o.FivePrimeWindow > 0 && i < o.FivePrimeWindow {
		return o.FivePrimeMultiplier
	}
	return 1
}

// MismatchContribution describes one non-Watson-Crick primer-template column.
type MismatchContribution struct {
	Pos                int
	PrimerBase         byte
	TargetBase         byte
	P5                 byte
	P3                 byte
	T5                 byte
	T3                 byte
	Source             MismatchLookupSource
	RawDeltaTmC        float64
	WeightedDeltaTmC   float64
	DeltaGPenaltyKcal  float64
	PositionMultiplier float64
	FivePrimeWindow    bool
	ThreePrimeWindow   bool
	FivePrimeTerminal  bool
	ThreePrimeTerminal bool
}

// ImperfectDuplexResult reports an approximate condition-aware imperfect
// primer-template duplex. The base nearest-neighbor terms come from the perfect
// primer/complement duplex; mismatch terms adjust Tm, ΔG(Tanneal), and margin.
type ImperfectDuplexResult struct {
	DuplexResult
	MismatchPenaltyC        float64
	DeltaGPenaltyKcal       float64
	MismatchCount           int
	FivePrimeMismatchCount  int
	ThreePrimeMismatchCount int
	TerminalMismatchCount   int
	TripletTmCount          int
	TripletDeltaGCount      int
	HeuristicFallbackCount  int
	DefaultFallbackCount    int
	HasNonWatsonCrick       bool
	UsedHeuristicAdjust     bool
	MismatchPolicy          string
	Contributions           []MismatchContribution
}

// ImperfectDuplex computes an imperfect primer-template duplex using default
// positional weighting.
func ImperfectDuplex(primer5to3, target3to5 string, cond Conditions) (ImperfectDuplexResult, error) {
	return ImperfectDuplexWithOptions(primer5to3, target3to5, cond, DefaultImperfectDuplexOptions())
}

// ImperfectDuplexWithOptions computes condition-aware primer-template duplex
// quantities for equal-length A/C/G/T primers against A/C/G/T/N target sites.
// The target strand must be supplied 3'→5' in primer-aligned coordinates.
func ImperfectDuplexWithOptions(primer5to3, target3to5 string, cond Conditions, opts ImperfectDuplexOptions) (ImperfectDuplexResult, error) {
	var out ImperfectDuplexResult
	p := strings.ToUpper(strings.TrimSpace(primer5to3))
	t := strings.ToUpper(strings.TrimSpace(target3to5))
	if len(p) == 0 || len(t) == 0 || len(p) != len(t) {
		return out, errors.New("ImperfectDuplex: sequences must be equal length and non-empty")
	}
	for i := 0; i < len(p); i++ {
		if !isACGT(p[i]) {
			return out, fmt.Errorf("ImperfectDuplex: non-ACGT base in primer at pos %d", i)
		}
		if !isNT(t[i]) {
			return out, fmt.Errorf("ImperfectDuplex: unsupported target base at pos %d", i)
		}
	}

	perfectTarget, ok := compStrict(p)
	if !ok {
		return out, errors.New("ImperfectDuplex: non-ACGT base in primer")
	}
	base, err := PerfectDuplex(p, perfectTarget, cond)
	if err != nil {
		return out, err
	}

	denom := math.Abs(base.EffectiveDenomCalK)
	if math.IsNaN(denom) || math.IsInf(denom, 0) || denom == 0 {
		denom = 200.0
	}

	n := len(p)
	opts = opts.withDefaults()
	penaltyC := 0.0
	for i := 0; i < n; i++ {
		if wc(p[i], t[i]) {
			continue
		}
		p5, pC, p3 := mismatchAt(p, i-1), p[i], mismatchAt(p, i+1)
		t5, tC, t3 := mismatchAt(t, i-1), t[i], mismatchAt(t, i+1)
		rawTm := 0.0
		source := MismatchSourceDefaultDeltaTm
		deltaG := 0.0

		if dTm, src, ok := LookupDeltaTmDetail(p5, pC, p3, t5, tC, t3); ok {
			rawTm = dTm
			source = src
			out.TripletTmCount++
		} else if dG, src, ok := LookupDeltaGDetail(p5, pC, p3, t5, tC, t3); ok {
			deltaG = dG
			rawTm = DeltaGToDeltaTm(dG, denom)
			source = src
			switch src {
			case MismatchSourceTripletDeltaG:
				out.TripletDeltaGCount++
			case MismatchSourceHeuristicDeltaG:
				out.HeuristicFallbackCount++
			}
		} else {
			rawTm = opts.DefaultMismatchDeltaTm
			out.DefaultFallbackCount++
		}

		mult := opts.posMultiplier(i, n)
		weighted := rawTm * mult
		if weighted < 0 {
			// Preserve the historical confidence cap: a mismatch should not make
			// the imperfect duplex better than the perfect complement.
			weighted = 0
		}
		penaltyC += weighted

		fiveWindow := opts.FivePrimeWindow > 0 && i < opts.FivePrimeWindow
		threeWindow := opts.ThreePrimeWindow > 0 && i >= n-opts.ThreePrimeWindow
		if fiveWindow {
			out.FivePrimeMismatchCount++
		}
		if threeWindow {
			out.ThreePrimeMismatchCount++
		}
		if i == 0 || i == n-1 {
			out.TerminalMismatchCount++
		}
		out.MismatchCount++
		out.Contributions = append(out.Contributions, MismatchContribution{
			Pos:                i,
			PrimerBase:         pC,
			TargetBase:         tC,
			P5:                 p5,
			P3:                 p3,
			T5:                 t5,
			T3:                 t3,
			Source:             source,
			RawDeltaTmC:        rawTm,
			WeightedDeltaTmC:   weighted,
			DeltaGPenaltyKcal:  deltaG * mult,
			PositionMultiplier: mult,
			FivePrimeWindow:    fiveWindow,
			ThreePrimeWindow:   threeWindow,
			FivePrimeTerminal:  i == 0,
			ThreePrimeTerminal: i == n-1,
		})
	}

	adjusted := base
	if penaltyC < 0 {
		penaltyC = 0
	}
	deltaGPenalty := penaltyC * denom / 1000.0
	adjusted.TmC = base.TmC - penaltyC
	adjusted.AnnealMarginC = adjusted.TmC - cond.WithDefaults().AnnealC
	adjusted.DeltaGAtAnnealKcal = base.DeltaGAtAnnealKcal + deltaGPenalty
	adjusted.EffectiveDenomCalK = denom

	policy := MismatchPolicyPerfect
	if out.MismatchCount > 0 {
		policy = MismatchPolicyImperfectV1
		if out.DefaultFallbackCount > 0 {
			policy = MismatchPolicyImperfectDefaultFallback
		} else if out.HeuristicFallbackCount > 0 {
			policy = MismatchPolicyImperfectHeuristicFallback
		}
	}

	out.DuplexResult = adjusted
	out.MismatchPenaltyC = penaltyC
	out.DeltaGPenaltyKcal = deltaGPenalty
	out.HasNonWatsonCrick = out.MismatchCount > 0
	out.UsedHeuristicAdjust = out.HeuristicFallbackCount > 0 || out.DefaultFallbackCount > 0
	out.MismatchPolicy = policy
	return out, nil
}

func mismatchAt(s string, idx int) byte {
	if idx < 0 || idx >= len(s) {
		return 'N'
	}
	return s[idx]
}
