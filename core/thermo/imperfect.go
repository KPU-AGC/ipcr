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

	// MismatchPolicyImperfectTriplet identifies an imperfect-duplex result
	// scored with exact triplet-level mismatch ΔΔG or ΔTm parameters.
	MismatchPolicyImperfectTriplet = "nn-imperfect-v1-with-triplet-ddg"

	// MismatchPolicyImperfectCuratedPair identifies an imperfect-duplex result
	// scored with the curated pair-family mismatch parameter registry.
	MismatchPolicyImperfectCuratedPair = "nn-imperfect-v1-with-curated-pair-ddg"

	// MismatchPolicyImperfectHeuristicFallback identifies an imperfect-duplex
	// result that had to use the current pair/context ΔΔG fallback for at least
	// one mismatch because no curated triplet/pair-family parameter was available.
	MismatchPolicyImperfectHeuristicFallback = "nn-imperfect-v1-with-heuristic-ddg-fallback"

	// MismatchPolicyImperfectDefaultFallback identifies an imperfect-duplex result
	// that encountered an unsupported mismatch context and used the conservative
	// default ΔTm fallback.
	MismatchPolicyImperfectDefaultFallback = "nn-imperfect-v1-with-default-fallback"

	// EndEffectPolicyNone identifies a duplex with no explicit terminal/dangling
	// correction beyond the ordinary end-window mismatch multiplier.
	EndEffectPolicyNone = "none"

	// EndEffectPolicyTerminalMismatchV1 identifies the exact-terminal mismatch
	// correction layer applied after ordinary 5'/3' end-window weighting.
	EndEffectPolicyTerminalMismatchV1 = "nn-terminal-mismatch-v1"

	// EndEffectPolicyTemplateDanglingV1 identifies the SantaLucia-Hicks
	// sequence-context model for a template base adjacent to the primer-template
	// duplex.
	EndEffectPolicyTemplateDanglingV1 = "nn-template-dangling-end-v1"

	// EndEffectPolicyTerminalAndDanglingV1 identifies rows where both v1 end-effect
	// layers were applied.
	EndEffectPolicyTerminalAndDanglingV1 = "nn-terminal-mismatch-template-dangling-v1"
)

const (
	defaultFivePrimeMismatchWindow   = 3
	defaultThreePrimeMismatchWindow  = 3
	defaultFivePrimeMismatchWeight   = 1.5
	defaultThreePrimeMismatchWeight  = 2.0
	defaultMismatchDeltaTmC          = 4.0
	defaultFivePrimeTerminalPenalty  = 0.5
	defaultThreePrimeTerminalPenalty = 1.5
)

// ImperfectDuplexOptions controls the positional weighting used by the current
// imperfect-duplex model. Positions are primer 5'→3' indexes.
type ImperfectDuplexOptions struct {
	FivePrimeWindow        int
	ThreePrimeWindow       int
	FivePrimeMultiplier    float64
	ThreePrimeMultiplier   float64
	DefaultMismatchDeltaTm float64

	// Exact terminal mismatch penalties are added after the ordinary 5'/3' window
	// multiplier. They are deliberately separate so diagnostics can distinguish a
	// literal terminal-base mismatch from a broader end-window mismatch.
	FivePrimeTerminalPenaltyC  float64
	ThreePrimeTerminalPenaltyC float64
}

// DefaultImperfectDuplexOptions returns the positional weighting historically
// used by ipcr-thermo, with an explicit extra term for literal terminal bases.
func DefaultImperfectDuplexOptions() ImperfectDuplexOptions {
	return ImperfectDuplexOptions{
		FivePrimeWindow:            defaultFivePrimeMismatchWindow,
		ThreePrimeWindow:           defaultThreePrimeMismatchWindow,
		FivePrimeMultiplier:        defaultFivePrimeMismatchWeight,
		ThreePrimeMultiplier:       defaultThreePrimeMismatchWeight,
		DefaultMismatchDeltaTm:     defaultMismatchDeltaTmC,
		FivePrimeTerminalPenaltyC:  defaultFivePrimeTerminalPenalty,
		ThreePrimeTerminalPenaltyC: defaultThreePrimeTerminalPenalty,
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
	if o.FivePrimeTerminalPenaltyC < 0 {
		o.FivePrimeTerminalPenaltyC = 0
	} else if o.FivePrimeTerminalPenaltyC == 0 {
		o.FivePrimeTerminalPenaltyC = d.FivePrimeTerminalPenaltyC
	}
	if o.ThreePrimeTerminalPenaltyC < 0 {
		o.ThreePrimeTerminalPenaltyC = 0
	} else if o.ThreePrimeTerminalPenaltyC == 0 {
		o.ThreePrimeTerminalPenaltyC = d.ThreePrimeTerminalPenaltyC
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

func (o ImperfectDuplexOptions) terminalPenalty(i, n int) float64 {
	o = o.withDefaults()
	switch {
	case n <= 0:
		return 0
	case i == n-1:
		return o.ThreePrimeTerminalPenaltyC
	case i == 0:
		return o.FivePrimeTerminalPenaltyC
	default:
		return 0
	}
}

// MismatchContribution describes one non-Watson-Crick primer-template column.
type MismatchContribution struct {
	Pos                   int
	PrimerBase            byte
	TargetBase            byte
	P5                    byte
	P3                    byte
	T5                    byte
	T3                    byte
	Source                MismatchLookupSource
	ParameterSet          string
	Citation              string
	ParameterNote         string
	RawDeltaTmC           float64
	WeightedDeltaTmC      float64
	TerminalPenaltyC      float64
	TerminalSource        string
	TerminalParameterSet  string
	TerminalCitation      string
	TerminalParameterNote string
	DeltaGPenaltyKcal     float64
	PositionMultiplier    float64
	FivePrimeWindow       bool
	ThreePrimeWindow      bool
	FivePrimeTerminal     bool
	ThreePrimeTerminal    bool
}

// DanglingEndContext supplies target-strand bases adjacent to the duplex in
// primer-aligned coordinates. In PCR-product scoring, the primer 3' adjacent
// template base is usually available from the amplicon interior; the 5' outside
// flank generally is not carried in engine.Product.
type DanglingEndContext struct {
	FivePrimeBase  byte
	ThreePrimeBase byte
}

// DanglingEndContribution describes one table-backed template-adjacent dangling
// base correction. AdjustmentC is positive when the base stabilizes the endpoint,
// and negative for the few experimentally observed destabilizing dangling ends.
type DanglingEndContribution struct {
	Side               string
	Base               byte
	TerminalPrimerBase byte
	TerminalTargetBase byte
	DanglingStrandSide byte
	Motif              string
	DeltaHkcal         float64
	DeltaScalK         float64
	DeltaGKcal         float64
	DeltaG37kcal       float64
	AdjustmentC        float64
	Source             string
	ParameterSet       string
	Citation           string
	ParameterNote      string
}

// ImperfectDuplexResult reports an approximate condition-aware imperfect
// primer-template duplex. The base nearest-neighbor terms come from the perfect
// primer/complement duplex; mismatch and end-effect terms adjust Tm, ΔG(Tanneal),
// and margin.
type ImperfectDuplexResult struct {
	DuplexResult
	MismatchPenaltyC                   float64
	DeltaGPenaltyKcal                  float64
	TerminalMismatchPenaltyC           float64
	TerminalMismatchDeltaGKcal         float64
	DanglingEndAdjustmentC             float64
	DanglingEndDeltaGKcal              float64
	DanglingEndCount                   int
	MismatchCount                      int
	FivePrimeMismatchCount             int
	ThreePrimeMismatchCount            int
	FivePrimeTerminalMismatchCount     int
	ThreePrimeTerminalMismatchCount    int
	TerminalMismatchCount              int
	FivePrimeTerminalMismatchPenaltyC  float64
	ThreePrimeTerminalMismatchPenaltyC float64
	EndEffectPolicy                    string
	TripletTmCount                     int
	TripletDeltaGCount                 int
	CuratedPairCount                   int
	HeuristicFallbackCount             int
	DefaultFallbackCount               int
	HasNonWatsonCrick                  bool
	UsedHeuristicAdjust                bool
	MismatchPolicy                     string
	Contributions                      []MismatchContribution
	DanglingContributions              []DanglingEndContribution
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
	return ImperfectDuplexWithOptionsAndContext(primer5to3, target3to5, cond, opts, DanglingEndContext{})
}

// ImperfectDuplexWithOptionsAndContext computes condition-aware primer-template
// duplex quantities and applies optional template-adjacent dangling-end terms
// when flanking target bases are supplied.
func ImperfectDuplexWithOptionsAndContext(primer5to3, target3to5 string, cond Conditions, opts ImperfectDuplexOptions, ctx DanglingEndContext) (ImperfectDuplexResult, error) {
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
		parameterSet := ""
		citation := ""
		parameterNote := ""

		if dTm, src, ok := LookupDeltaTmDetail(p5, pC, p3, t5, tC, t3); ok {
			rawTm = dTm
			source = src
			out.TripletTmCount++
		} else if dG, src, ok := LookupDeltaGDetail(p5, pC, p3, t5, tC, t3); ok {
			deltaG = dG
			rawTm = DeltaGToDeltaTm(dG, denom)
			source = src
			if param, ok := LookupMismatchParameterInfoForContext(p5, pC, p3, t5, tC, t3); ok {
				parameterSet = param.ParameterSet
				citation = param.Citation
				parameterNote = param.Note
			}
			switch src {
			case MismatchSourceTripletDeltaG:
				out.TripletDeltaGCount++
			case MismatchSourceCuratedPairDeltaG:
				out.CuratedPairCount++
			case MismatchSourceHeuristicDeltaG:
				out.HeuristicFallbackCount++
			}
		} else {
			rawTm = opts.DefaultMismatchDeltaTm
			out.DefaultFallbackCount++
		}

		mult := opts.posMultiplier(i, n)
		terminalPenalty := 0.0
		terminalSource := ""
		terminalParameterSet := ""
		terminalCitation := ""
		terminalParameterNote := ""
		if terminalKey, ok := TerminalMismatchKeyForPosition(p, t, i); ok {
			if terminalParam, ok := LookupTerminalMismatchParameterWithFallback(terminalKey, opts); ok {
				terminalSource = terminalParam.Source
				terminalParameterSet = terminalParam.ParameterSet
				terminalCitation = terminalParam.Citation
				terminalParameterNote = terminalParam.Note
				if terminalParam.HasDeltaTm {
					terminalPenalty = terminalParam.DeltaTmC
				} else if terminalParam.HasDeltaDeltaG37 {
					terminalPenalty = DeltaGToDeltaTm(terminalParam.DeltaDeltaG37kcal, denom)
				}
			}
		}
		weighted := rawTm*mult + terminalPenalty
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
			out.TerminalMismatchPenaltyC += terminalPenalty
		}
		if i == 0 {
			out.FivePrimeTerminalMismatchCount++
			out.FivePrimeTerminalMismatchPenaltyC += terminalPenalty
		}
		if i == n-1 {
			out.ThreePrimeTerminalMismatchCount++
			out.ThreePrimeTerminalMismatchPenaltyC += terminalPenalty
		}
		out.MismatchCount++
		out.Contributions = append(out.Contributions, MismatchContribution{
			Pos:                   i,
			PrimerBase:            pC,
			TargetBase:            tC,
			P5:                    p5,
			P3:                    p3,
			T5:                    t5,
			T3:                    t3,
			Source:                source,
			ParameterSet:          parameterSet,
			Citation:              citation,
			ParameterNote:         parameterNote,
			RawDeltaTmC:           rawTm,
			WeightedDeltaTmC:      weighted,
			TerminalPenaltyC:      terminalPenalty,
			TerminalSource:        terminalSource,
			TerminalParameterSet:  terminalParameterSet,
			TerminalCitation:      terminalCitation,
			TerminalParameterNote: terminalParameterNote,
			DeltaGPenaltyKcal:     deltaG*mult + terminalPenalty*denom/1000.0,
			PositionMultiplier:    mult,
			FivePrimeWindow:       fiveWindow,
			ThreePrimeWindow:      threeWindow,
			FivePrimeTerminal:     i == 0,
			ThreePrimeTerminal:    i == n-1,
		})
	}

	adjusted := base
	if penaltyC < 0 {
		penaltyC = 0
	}
	deltaGPenalty := penaltyC * denom / 1000.0
	danglingAdjustmentC, danglingDeltaG, dangling := danglingEndAdjustment(ctx, p, t, denom)
	adjusted.TmC = base.TmC - penaltyC + danglingAdjustmentC
	adjusted.AnnealMarginC = adjusted.TmC - cond.WithDefaults().AnnealC
	adjusted.DeltaGAtAnnealKcal = base.DeltaGAtAnnealKcal + deltaGPenalty + danglingDeltaG
	adjusted.EffectiveDenomCalK = denom

	policy := MismatchPolicyPerfect
	if out.MismatchCount > 0 {
		policy = MismatchPolicyImperfectV1
		if out.DefaultFallbackCount > 0 {
			policy = MismatchPolicyImperfectDefaultFallback
		} else if out.HeuristicFallbackCount > 0 {
			policy = MismatchPolicyImperfectHeuristicFallback
		} else if out.TripletTmCount+out.TripletDeltaGCount > 0 {
			policy = MismatchPolicyImperfectTriplet
		} else if out.CuratedPairCount > 0 {
			policy = MismatchPolicyImperfectCuratedPair
		}
	}

	out.DuplexResult = adjusted
	out.MismatchPenaltyC = penaltyC
	out.DeltaGPenaltyKcal = deltaGPenalty
	out.TerminalMismatchDeltaGKcal = out.TerminalMismatchPenaltyC * denom / 1000.0
	out.DanglingEndAdjustmentC = danglingAdjustmentC
	out.DanglingEndDeltaGKcal = danglingDeltaG
	out.DanglingEndCount = len(dangling)
	out.DanglingContributions = dangling
	out.EndEffectPolicy = endEffectPolicy(out.TerminalMismatchPenaltyC > 0, len(dangling) > 0)
	out.HasNonWatsonCrick = out.MismatchCount > 0
	out.UsedHeuristicAdjust = out.HeuristicFallbackCount > 0 || out.DefaultFallbackCount > 0
	out.MismatchPolicy = policy
	return out, nil
}

func endEffectPolicy(hasTerminalMismatch, hasDangling bool) string {
	switch {
	case hasTerminalMismatch && hasDangling:
		return EndEffectPolicyTerminalAndDanglingV1
	case hasDangling:
		return EndEffectPolicyTemplateDanglingV1
	case hasTerminalMismatch:
		return EndEffectPolicyTerminalMismatchV1
	default:
		return EndEffectPolicyNone
	}
}

func danglingEndAdjustment(ctx DanglingEndContext, primer, target string, denom float64) (float64, float64, []DanglingEndContribution) {
	if denom <= 0 || math.IsNaN(denom) || math.IsInf(denom, 0) || len(primer) == 0 || len(target) == 0 {
		return 0, 0, nil
	}
	contribs := make([]DanglingEndContribution, 0, 2)
	add := func(side string, base, terminalPrimer, terminalTarget byte) {
		param, ok := LookupTemplateDanglingEndParameter(side, base, terminalPrimer, terminalTarget)
		if !ok {
			return
		}
		dg := param.DeltaG37kcal
		adjC := -dg * 1000.0 / denom
		contribs = append(contribs, DanglingEndContribution{
			Side:               side,
			DanglingStrandSide: param.Key.StrandEnd,
			Base:               param.Key.DanglingBase,
			TerminalPrimerBase: param.Key.OppositeBase,
			TerminalTargetBase: param.Key.PairedBase,
			Motif:              param.Motif,
			DeltaHkcal:         param.DeltaHkcal,
			DeltaScalK:         param.DeltaScalK,
			DeltaGKcal:         dg,
			DeltaG37kcal:       dg,
			AdjustmentC:        adjC,
			Source:             param.Source,
			ParameterSet:       param.ParameterSet,
			Citation:           param.Citation,
			ParameterNote:      param.Note,
		})
	}
	// Target/template 3' dangling bases sit beside the primer 5' end.
	add("primer-5p", ctx.FivePrimeBase, primer[0], target[0])
	// Target/template 5' dangling bases sit beside the primer 3' end.
	add("primer-3p", ctx.ThreePrimeBase, primer[len(primer)-1], target[len(target)-1])

	adjustmentC := 0.0
	deltaG := 0.0
	for _, c := range contribs {
		adjustmentC += c.AdjustmentC
		deltaG += c.DeltaGKcal
	}
	return adjustmentC, deltaG, contribs
}

func normalizeBase(b byte) byte {
	switch b {
	case 'a', 'A':
		return 'A'
	case 'c', 'C':
		return 'C'
	case 'g', 'G':
		return 'G'
	case 't', 'T':
		return 'T'
	default:
		return 'N'
	}
}

func mismatchAt(s string, idx int) byte {
	if idx < 0 || idx >= len(s) {
		return 'N'
	}
	return s[idx]
}
