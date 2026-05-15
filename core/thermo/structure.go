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

	StructureModelContiguousStemV1 = "nn-contiguous-stem-v1"
	StructureModelStemLoopV2       = "nn-stem-loop-v2"
)

// StructureOptions configures the v1 secondary-structure evaluator.
type StructureOptions struct {
	Conditions Conditions
	MinStem    int
	MinLoop    int

	// V2 permits one interruption inside an otherwise Watson-Crick stem. Bulges
	// are one-sided interruptions; internal loops have unpaired bases on both
	// strands. The defaults keep the search conservative for primer-scale oligos.
	MaxBulge        int
	MaxInternalLoop int
}

// StructureResult describes the strongest contiguous nearest-neighbor structure
// found for a primer or primer pair. Coordinates are 0-based in the submitted
// 5'→3' sequence(s).
type StructureResult struct {
	Kind                 string
	Model                string
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

	SegmentCount            int
	BulgeCount              int
	InternalLoopCount       int
	DanglingEndCount        int
	LoopPenaltyKcal         float64
	BulgePenaltyKcal        float64
	InternalLoopPenaltyKcal float64
	DanglingAdjustmentKcal  float64
}

func DefaultStructureOptions(cond Conditions) StructureOptions {
	return StructureOptions{
		Conditions:      cond.WithDefaults(),
		MinStem:         4,
		MinLoop:         3,
		MaxBulge:        2,
		MaxInternalLoop: 2,
	}
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

func BestHairpinV2(seq5to3 string, opts StructureOptions) (StructureResult, bool, error) {
	v1, ok1, err := BestHairpin(seq5to3, opts)
	if err != nil {
		return StructureResult{}, false, err
	}
	if ok1 {
		v1.Model = StructureModelContiguousStemV1
	}
	v2, ok2, err := bestHairpinGapped(seq5to3, opts)
	if err != nil {
		return v1, ok1, err
	}
	if ok2 && betterStructureResult(v2, v1) {
		return v2, true, nil
	}
	return v1, ok1, nil
}

func BestSelfDimerV2(seq5to3 string, opts StructureOptions) (StructureResult, bool, error) {
	return bestDimerV2(seq5to3, seq5to3, opts, StructureSelfDimer)
}

func BestCrossDimerV2(a5to3, b5to3 string, opts StructureOptions) (StructureResult, bool, error) {
	return bestDimerV2(a5to3, b5to3, opts, StructureCrossDimer)
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
		Model:              StructureModelContiguousStemV1,
		DeltaGAtAnnealKcal: dg,
		TmC:                tmC,
		AnnealMarginC:      tmC - cond.AnnealC,
		StemLen:            len(top5to3),
		LoopLen:            loopLen,
		SegmentCount:       1,
		LoopPenaltyKcal:    loopPenalty,
	}, nil
}

type gappedStemCandidate struct {
	aStart int
	bStart int
	stem1  int
	gapA   int
	gapB   int
	stem2  int
}

func (c gappedStemCandidate) aEnd() int    { return c.aStart + c.stem1 + c.gapA + c.stem2 }
func (c gappedStemCandidate) bEnd() int    { return c.bStart + c.stem1 + c.gapB + c.stem2 }
func (c gappedStemCandidate) stemLen() int { return c.stem1 + c.stem2 }

func bestHairpinGapped(seq5to3 string, opts StructureOptions) (StructureResult, bool, error) {
	opts = normalizeStructureOptions(opts)
	seq, ok := normalizeACGTStructure(seq5to3)
	if !ok {
		return StructureResult{}, false, errors.New("hairpin: sequence must be A/C/G/T")
	}
	if len(seq) < 2*opts.MinStem+opts.MinLoop+1 {
		return StructureResult{}, false, nil
	}

	var best StructureResult
	for i := 0; i < len(seq); i++ {
		for j := i + opts.MinStem + opts.MinLoop; j < len(seq); j++ {
			rightArm := seq[j:]
			target3 := reverseStructure(rightArm)
			for _, cand := range enumerateGappedStemCandidates(seq, target3, i, 0, opts) {
				loopLen := j - cand.aEnd()
				if loopLen < opts.MinLoop || cand.bEnd() > len(target3) {
					continue
				}
				res, err := gappedStemThermo(StructureHairpin, seq, target3, cand, opts.Conditions, loopLen)
				if err != nil {
					continue
				}
				res.AStart, res.AEnd = cand.aStart, cand.aEnd()
				// target3 is reverse(rightArm); convert back to original sequence coordinates.
				res.BStart, res.BEnd = j+len(rightArm)-cand.bEnd(), j+len(rightArm)-cand.bStart
				res.ThreePrimeAnchored = res.AEnd == len(seq) || res.BEnd == len(seq)
				if betterStructureResult(res, best) {
					best = res
				}
			}
		}
	}
	return best, best.StemLen > 0, nil
}

func bestDimerV2(a5to3, b5to3 string, opts StructureOptions, kind string) (StructureResult, bool, error) {
	v1, ok1, err := bestDimer(a5to3, b5to3, opts, kind)
	if err != nil {
		return StructureResult{}, false, err
	}
	if ok1 {
		v1.Model = StructureModelContiguousStemV1
	}
	opts = normalizeStructureOptions(opts)
	a, okA := normalizeACGTStructure(a5to3)
	b, okB := normalizeACGTStructure(b5to3)
	if !okA || !okB {
		return StructureResult{}, false, errors.New("dimer: sequences must be A/C/G/T")
	}
	bTarget3 := reverseStructure(b)

	best := v1
	for i := 0; i < len(a); i++ {
		for j := 0; j < len(bTarget3); j++ {
			for _, cand := range enumerateGappedStemCandidates(a, bTarget3, i, j, opts) {
				if cand.aEnd() > len(a) || cand.bEnd() > len(bTarget3) {
					continue
				}
				res, err := gappedStemThermo(kind, a, bTarget3, cand, opts.Conditions, 0)
				if err != nil {
					continue
				}
				res.AStart, res.AEnd = cand.aStart, cand.aEnd()
				res.BStart, res.BEnd = len(b)-cand.bEnd(), len(b)-cand.bStart
				a3 := res.AEnd == len(a)
				b3 := res.BEnd == len(b)
				res.ThreePrimeAnchored = a3 || b3
				res.BothThreePrimeAnchor = a3 && b3
				if betterStructureResult(res, best) {
					best = res
				}
			}
		}
	}
	return best, best.StemLen > 0, nil
}

func enumerateGappedStemCandidates(top, target3 string, aStart, bStart int, opts StructureOptions) []gappedStemCandidate {
	minSeg := 2
	if opts.MinStem < 4 {
		minSeg = 1
	}
	out := make([]gappedStemCandidate, 0)
	maxA := len(top) - aStart
	maxB := len(target3) - bStart
	for stem1 := minSeg; stem1 <= maxA && stem1 <= maxB; stem1++ {
		if !segmentWC(top, target3, aStart, bStart, stem1) {
			break
		}
		for gapA := 0; gapA <= opts.MaxBulge; gapA++ {
			for gapB := 0; gapB <= opts.MaxInternalLoop; gapB++ {
				if gapA == 0 && gapB == 0 {
					continue
				}
				if gapA > 0 && gapB > 0 && (gapA > opts.MaxInternalLoop || gapB > opts.MaxInternalLoop) {
					continue
				}
				a2 := aStart + stem1 + gapA
				b2 := bStart + stem1 + gapB
				if a2 >= len(top) || b2 >= len(target3) {
					continue
				}
				for stem2 := minSeg; a2+stem2 <= len(top) && b2+stem2 <= len(target3); stem2++ {
					if !segmentWC(top, target3, a2, b2, stem2) {
						break
					}
					if stem1+stem2 < opts.MinStem {
						continue
					}
					out = append(out, gappedStemCandidate{aStart: aStart, bStart: bStart, stem1: stem1, gapA: gapA, gapB: gapB, stem2: stem2})
				}
			}
		}
	}
	return out
}

func segmentWC(top, target3 string, aStart, bStart, n int) bool {
	if aStart < 0 || bStart < 0 || aStart+n > len(top) || bStart+n > len(target3) {
		return false
	}
	for k := 0; k < n; k++ {
		if !wc(top[aStart+k], target3[bStart+k]) {
			return false
		}
	}
	return true
}

func gappedStemThermo(kind, top, target3 string, cand gappedStemCandidate, cond Conditions, loopLen int) (StructureResult, error) {
	seg1Top := top[cand.aStart : cand.aStart+cand.stem1]
	seg1Target := target3[cand.bStart : cand.bStart+cand.stem1]
	seg2TopStart := cand.aStart + cand.stem1 + cand.gapA
	seg2TargetStart := cand.bStart + cand.stem1 + cand.gapB
	seg2Top := top[seg2TopStart : seg2TopStart+cand.stem2]
	seg2Target := target3[seg2TargetStart : seg2TargetStart+cand.stem2]

	joinedTop := seg1Top + seg2Top
	joinedTarget := seg1Target + seg2Target
	res, err := stemThermo(kind, joinedTop, joinedTarget, cond, loopLen)
	if err != nil {
		return StructureResult{}, err
	}
	res.Model = StructureModelStemLoopV2
	res.StemLen = cand.stemLen()
	res.SegmentCount = 2
	res.LoopLen = loopLen

	gapPenalty := structureGapPenaltyKcal(cand.gapA, cand.gapB)
	danglingAdjustment := structureDanglingAdjustmentKcal(top, target3, cand)
	if cand.gapA > 0 && cand.gapB > 0 {
		res.InternalLoopCount = 1
		res.InternalLoopPenaltyKcal = gapPenalty
	} else {
		res.BulgeCount = 1
		res.BulgePenaltyKcal = gapPenalty
	}
	res.DanglingAdjustmentKcal = danglingAdjustment
	if danglingAdjustment != 0 {
		res.DanglingEndCount = 1
	}
	res.LoopPenaltyKcal += gapPenalty
	res.DeltaGAtAnnealKcal += gapPenalty + danglingAdjustment
	denom := structureDenomCalPerK(cond)
	res.TmC -= (gapPenalty + danglingAdjustment) * 1000.0 / denom
	res.AnnealMarginC = res.TmC - cond.WithDefaults().AnnealC
	return res, nil
}

func structureGapPenaltyKcal(gapA, gapB int) float64 {
	gapTotal := gapA + gapB
	if gapTotal <= 0 {
		return 0
	}
	if gapA > 0 && gapB > 0 {
		asymmetry := math.Abs(float64(gapA - gapB))
		return 1.4 + 0.35*math.Log(float64(gapTotal+1)) + 0.25*asymmetry
	}
	return 0.8 + 0.45*math.Log(float64(gapTotal+1))
}

func structureDanglingAdjustmentKcal(top, target3 string, cand gappedStemCandidate) float64 {
	adj := 0.0
	for _, b := range unpairedStructureBases(top, cand.aStart+cand.stem1, cand.gapA) {
		adj += danglingBaseAdjustmentKcal(b)
	}
	for _, b := range unpairedStructureBases(target3, cand.bStart+cand.stem1, cand.gapB) {
		adj += danglingBaseAdjustmentKcal(b)
	}
	// Bound the v2 dangling-end approximation so it cannot dominate the loop cost.
	if adj < -0.30 {
		return -0.30
	}
	return adj
}

func unpairedStructureBases(s string, start, n int) []byte {
	if n <= 0 || start < 0 || start >= len(s) {
		return nil
	}
	end := start + n
	if end > len(s) {
		end = len(s)
	}
	return []byte(s[start:end])
}

func danglingBaseAdjustmentKcal(b byte) float64 {
	switch b {
	case 'G', 'C':
		return -0.08
	case 'A', 'T':
		return -0.05
	default:
		return 0
	}
}

func structureDenomCalPerK(cond Conditions) float64 {
	cond = cond.WithDefaults()
	denom := 200.0
	if tm, err := Tm("GCGC", "CGCG", cond.TmInput()); err == nil {
		in := cond.TmInput()
		candidate := tm.DS_Na + Rcal*math.Log(in.CT/float64(in.X))
		if candidate != 0 && !math.IsNaN(candidate) && !math.IsInf(candidate, 0) {
			denom = math.Abs(candidate)
		}
	}
	return denom
}

func normalizeStructureOptions(opts StructureOptions) StructureOptions {
	if opts.MinStem <= 0 {
		opts.MinStem = 4
	}
	if opts.MinLoop <= 0 {
		opts.MinLoop = 3
	}
	if opts.MaxBulge <= 0 {
		opts.MaxBulge = 2
	}
	if opts.MaxInternalLoop <= 0 {
		opts.MaxInternalLoop = 2
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
