package thermo

import (
	"errors"
	"math"
	"strings"
)

const (
	StructureHairpin    = "hairpin"
	StructureSelfDimer  = "self-dimer"
	StructureCrossDimer = "cross-dimer"
)

// StructureOptions configures the v1 secondary-structure evaluator.
type StructureOptions struct {
	Conditions Conditions
	MinStem    int
	MinLoop    int
}

// StructureResult describes the strongest contiguous nearest-neighbor structure
// found for a primer or primer pair. Coordinates are 0-based in the submitted
// 5'→3' sequence(s).
type StructureResult struct {
	Kind                 string
	DeltaGAtAnnealKcal   float64
	TmC                  float64
	AnnealMarginC        float64
	StemLen              int
	LoopLen              int
	AStart               int
	AEnd                 int
	BStart               int
	BEnd                 int
	ThreePrimeAnchored   bool
	BothThreePrimeAnchor bool
}

func DefaultStructureOptions(cond Conditions) StructureOptions {
	return StructureOptions{Conditions: cond.WithDefaults(), MinStem: 4, MinLoop: 3}
}

func BestHairpin(seq5to3 string, opts StructureOptions) (StructureResult, bool, error) {
	opts = normalizeStructureOptions(opts)
	seq, ok := normalizeACGTStructure(seq5to3)
	if !ok {
		return StructureResult{}, false, errors.New("hairpin: sequence must be A/C/G/T")
	}
	if len(seq) < 2*opts.MinStem+opts.MinLoop {
		return StructureResult{}, false, nil
	}

	var best StructureResult
	for i := 0; i <= len(seq)-opts.MinStem-opts.MinLoop-opts.MinStem; i++ {
		for j := i + opts.MinStem + opts.MinLoop; j <= len(seq)-opts.MinStem; j++ {
			maxStem := len(seq) - j
			if byLoop := j - i - opts.MinLoop; byLoop < maxStem {
				maxStem = byLoop
			}
			for stem := opts.MinStem; stem <= maxStem; stem++ {
				if !hairpinStemWC(seq, i, j, stem) {
					continue
				}
				loopLen := j - (i + stem)
				top := seq[i : i+stem]
				target3 := reverseStructure(seq[j : j+stem])
				cand, err := stemThermo(StructureHairpin, top, target3, opts.Conditions, loopLen)
				if err != nil {
					continue
				}
				cand.AStart, cand.AEnd = i, i+stem
				cand.BStart, cand.BEnd = j, j+stem
				cand.ThreePrimeAnchored = cand.AEnd == len(seq) || cand.BEnd == len(seq)
				cand.BothThreePrimeAnchor = false
				if betterStructureResult(cand, best) {
					best = cand
				}
			}
		}
	}
	return best, best.StemLen > 0, nil
}

func BestSelfDimer(seq5to3 string, opts StructureOptions) (StructureResult, bool, error) {
	return bestDimer(seq5to3, seq5to3, opts, StructureSelfDimer)
}

func BestCrossDimer(a5to3, b5to3 string, opts StructureOptions) (StructureResult, bool, error) {
	return bestDimer(a5to3, b5to3, opts, StructureCrossDimer)
}

func bestDimer(a5to3, b5to3 string, opts StructureOptions, kind string) (StructureResult, bool, error) {
	opts = normalizeStructureOptions(opts)
	a, okA := normalizeACGTStructure(a5to3)
	b, okB := normalizeACGTStructure(b5to3)
	if !okA || !okB {
		return StructureResult{}, false, errors.New("dimer: sequences must be A/C/G/T")
	}
	if len(a) < opts.MinStem || len(b) < opts.MinStem {
		return StructureResult{}, false, nil
	}

	bTarget3 := reverseStructure(b)
	var best StructureResult
	for i := 0; i <= len(a)-opts.MinStem; i++ {
		for j := 0; j <= len(bTarget3)-opts.MinStem; j++ {
			run := 0
			for i+run < len(a) && j+run < len(bTarget3) && wc(a[i+run], bTarget3[j+run]) {
				run++
			}
			if run < opts.MinStem {
				continue
			}
			cand, err := stemThermo(kind, a[i:i+run], bTarget3[j:j+run], opts.Conditions, 0)
			if err != nil {
				continue
			}
			cand.AStart, cand.AEnd = i, i+run
			cand.BStart, cand.BEnd = len(b)-(j+run), len(b)-j
			a3 := cand.AEnd == len(a)
			b3 := cand.BEnd == len(b)
			cand.ThreePrimeAnchored = a3 || b3
			cand.BothThreePrimeAnchor = a3 && b3
			if betterStructureResult(cand, best) {
				best = cand
			}
		}
	}
	return best, best.StemLen > 0, nil
}

func stemThermo(kind, top5to3, target3to5 string, cond Conditions, loopLen int) (StructureResult, error) {
	if len(top5to3) != len(target3to5) || len(top5to3) == 0 {
		return StructureResult{}, errors.New("structure stem: invalid stem")
	}
	cond = cond.WithDefaults()
	local := cond
	local.SelfComplementary = kind == StructureSelfDimer
	res, err := Tm(top5to3, target3to5, local.TmInput())
	if err != nil {
		return StructureResult{}, err
	}
	in := local.TmInput()
	denom := res.DS_Na + Rcal*math.Log(in.CT/float64(in.X))
	if kind == StructureHairpin {
		denom = res.DS_Na
	}
	loopPenalty := 0.0
	if kind == StructureHairpin {
		loopPenalty = hairpinLoopPenaltyKcal(loopLen)
	}
	dg := res.DH_kcal - (cond.AnnealC+273.15)*denom/1000.0 + loopPenalty
	tmC := res.TmC
	if kind == StructureHairpin && denom != 0 {
		tmC = ((res.DH_kcal + loopPenalty) * 1000.0 / denom) - 273.15
	}
	return StructureResult{
		Kind:               kind,
		DeltaGAtAnnealKcal: dg,
		TmC:                tmC,
		AnnealMarginC:      tmC - cond.AnnealC,
		StemLen:            len(top5to3),
		LoopLen:            loopLen,
	}, nil
}

func normalizeStructureOptions(opts StructureOptions) StructureOptions {
	if opts.MinStem <= 0 {
		opts.MinStem = 4
	}
	if opts.MinLoop <= 0 {
		opts.MinLoop = 3
	}
	opts.Conditions = opts.Conditions.WithDefaults()
	return opts
}

func normalizeACGTStructure(s string) (string, bool) {
	out := strings.ToUpper(strings.TrimSpace(s))
	if out == "" {
		return "", false
	}
	for i := 0; i < len(out); i++ {
		switch out[i] {
		case 'A', 'C', 'G', 'T':
		default:
			return "", false
		}
	}
	return out, true
}

func hairpinStemWC(seq string, leftStart, rightStart, stem int) bool {
	for k := 0; k < stem; k++ {
		if !wc(seq[leftStart+k], seq[rightStart+stem-1-k]) {
			return false
		}
	}
	return true
}

func hairpinLoopPenaltyKcal(loopLen int) float64 {
	if loopLen <= 0 {
		return math.Inf(1)
	}
	return 3.0 + 0.2*math.Log(float64(loopLen))
}

func reverseStructure(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

func betterStructureResult(cand, best StructureResult) bool {
	if cand.StemLen == 0 {
		return false
	}
	if best.StemLen == 0 {
		return true
	}
	if cand.DeltaGAtAnnealKcal < best.DeltaGAtAnnealKcal-1e-9 {
		return true
	}
	if math.Abs(cand.DeltaGAtAnnealKcal-best.DeltaGAtAnnealKcal) <= 1e-9 {
		if cand.BothThreePrimeAnchor != best.BothThreePrimeAnchor {
			return cand.BothThreePrimeAnchor
		}
		if cand.ThreePrimeAnchored != best.ThreePrimeAnchored {
			return cand.ThreePrimeAnchored
		}
		return cand.StemLen > best.StemLen
	}
	return false
}
