package thermo

// Curated mismatch parameter registry.
//
// This v1 table intentionally starts with broad pair-family entries rather than
// pretending to be a complete nearest-neighbor mismatch parameterization. Exact
// triplet overrides can still be added to DeltaTmTriplet/DeltaGTriplet. The
// broad entries move ordinary A/C/G/T mismatches out of the generic fallback
// path and keep source metadata explicit.
const (
	MismatchSourceCuratedPairDeltaG MismatchLookupSource = "curated-pair-ddg-v1"

	MismatchParameterSetPairFamilyV1 = "ipcr-pair-family-v1"
)

type MismatchParameterInfo struct {
	DeltaDeltaGKcal float64
	Source          MismatchLookupSource
	ParameterSet    string
	Note            string
}

// DeltaGTripletSource optionally overrides the source label for user/populated
// DeltaGTriplet entries. If absent, an entry in DeltaGTriplet is treated as an
// exact triplet ΔΔG parameter.
var DeltaGTripletSource = map[MismatchKey]MismatchLookupSource{}

// DeltaGTripletParameterSet optionally labels user/populated DeltaGTriplet
// entries for diagnostics or tests.
var DeltaGTripletParameterSet = map[MismatchKey]string{}

// curatedDeltaGTriplet contains broad pair-family ΔΔG entries keyed with N
// wildcards in the flanking positions. Values are deliberately conservative and
// preserve the historical ipcr ordering while moving common A/C/G/T mismatches
// out of the heuristic-fallback path.
var curatedDeltaGTriplet = map[MismatchKey]MismatchParameterInfo{
	broadMismatchKey('G', 'T'): {0.60, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "G/T wobble pair-family default"},
	broadMismatchKey('T', 'G'): {0.60, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "T/G wobble pair-family default"},

	broadMismatchKey('A', 'G'): {0.85, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "A/G transition pair-family default"},
	broadMismatchKey('G', 'A'): {0.85, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "G/A transition pair-family default"},
	broadMismatchKey('C', 'T'): {0.85, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "C/T transition pair-family default"},
	broadMismatchKey('T', 'C'): {0.85, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "T/C transition pair-family default"},

	broadMismatchKey('A', 'C'): {1.10, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "A/C transversion pair-family default"},
	broadMismatchKey('C', 'A'): {1.10, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "C/A transversion pair-family default"},
	broadMismatchKey('A', 'T'): {1.20, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "A/T transversion pair-family default"},
	broadMismatchKey('T', 'A'): {1.20, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "T/A transversion pair-family default"},
	broadMismatchKey('C', 'G'): {1.20, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "C/G transversion pair-family default"},
	broadMismatchKey('G', 'C'): {1.20, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "G/C transversion pair-family default"},

	broadMismatchKey('A', 'A'): {1.40, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "A/A like-with-like pair-family default"},
	broadMismatchKey('T', 'T'): {1.40, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "T/T like-with-like pair-family default"},
	broadMismatchKey('G', 'G'): {1.40, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "G/G like-with-like pair-family default"},
	broadMismatchKey('C', 'C'): {1.40, MismatchSourceCuratedPairDeltaG, MismatchParameterSetPairFamilyV1, "C/C like-with-like pair-family default"},
}

func broadMismatchKey(p, t byte) MismatchKey {
	return MismatchKey{P5: 'N', P: p, P3: 'N', T5: 'N', T: t, T3: 'N'}
}

func mismatchCandidateKeys(p5, p, p3, t5, t, t3 byte) []MismatchKey {
	keys := []MismatchKey{
		{P5: mismatchFlank(p5), P: p, P3: mismatchFlank(p3), T5: mismatchFlank(t5), T: t, T3: mismatchFlank(t3)},
		{P5: mismatchFlank(p5), P: p, P3: mismatchFlank(p3), T5: 'N', T: t, T3: 'N'},
		{P5: 'N', P: p, P3: 'N', T5: mismatchFlank(t5), T: t, T3: mismatchFlank(t3)},
		broadMismatchKey(p, t),
	}
	out := make([]MismatchKey, 0, len(keys))
	seen := map[MismatchKey]struct{}{}
	for _, k := range keys {
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

func mismatchFlank(b byte) byte {
	if isACGT(b) {
		return b
	}
	return 'N'
}

func lookupDeltaTmTripletOverride(p5, p, p3, t5, t, t3 byte) (float64, bool) {
	for _, key := range mismatchCandidateKeys(p5, p, p3, t5, t, t3) {
		if d, ok := DeltaTmTriplet[key]; ok {
			return d, true
		}
	}
	return 0, false
}

func lookupDeltaGTripletOverride(p5, p, p3, t5, t, t3 byte) (float64, MismatchLookupSource, bool) {
	for _, key := range mismatchCandidateKeys(p5, p, p3, t5, t, t3) {
		if dg, ok := DeltaGTriplet[key]; ok {
			src := DeltaGTripletSource[key]
			if src == "" {
				src = MismatchSourceTripletDeltaG
			}
			return dg, src, true
		}
	}
	return 0, "", false
}

func lookupCuratedDeltaGTriplet(p5, p, p3, t5, t, t3 byte) (MismatchParameterInfo, bool) {
	for _, key := range mismatchCandidateKeys(p5, p, p3, t5, t, t3) {
		if param, ok := curatedDeltaGTriplet[key]; ok {
			return param, true
		}
	}
	return MismatchParameterInfo{}, false
}

// LookupMismatchParameterInfo exposes metadata for an exact or wildcard
// mismatch key. It is mostly used by tests and future report writers.
func LookupMismatchParameterInfo(key MismatchKey) (MismatchParameterInfo, bool) {
	if param, ok := curatedDeltaGTriplet[key]; ok {
		return param, true
	}
	if dg, ok := DeltaGTriplet[key]; ok {
		set := DeltaGTripletParameterSet[key]
		if set == "" {
			set = "user-triplet-ddg"
		}
		src := DeltaGTripletSource[key]
		if src == "" {
			src = MismatchSourceTripletDeltaG
		}
		return MismatchParameterInfo{DeltaDeltaGKcal: dg, Source: src, ParameterSet: set}, true
	}
	return MismatchParameterInfo{}, false
}
