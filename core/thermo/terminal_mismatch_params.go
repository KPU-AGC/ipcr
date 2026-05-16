package thermo

const (
	// TerminalMismatchParameterSetHeuristicV1 identifies the current ipcr terminal
	// mismatch penalty model. It is an empirical fixed-ΔTm fallback, not a
	// literature-backed nearest-neighbor thermodynamic table.
	TerminalMismatchParameterSetHeuristicV1 = "ipcr-terminal-mismatch-heuristic-v1"

	// TerminalMismatchSourceHeuristicPenalty labels terminal mismatch terms that
	// come from the built-in side-specific fallback penalties.
	TerminalMismatchSourceHeuristicPenalty = "ipcr-terminal-mismatch-heuristic"

	// TerminalMismatchPrimer5Prime identifies a mismatch at the primer 5' base.
	TerminalMismatchPrimer5Prime byte = '5'

	// TerminalMismatchPrimer3Prime identifies a mismatch at the primer 3' base.
	TerminalMismatchPrimer3Prime byte = '3'
)

const (
	terminalMismatchCitationHeuristicV1 = "ipcr internal terminal-mismatch heuristic; no peer-reviewed sequence-context terminal-mismatch thermodynamic table is applied"
	terminalMismatchNoteHeuristicV1     = "Empirical fixed terminal mismatch ΔTm penalty preserved from the ipcr imperfect-duplex model; this is not a sequence-context nearest-neighbor thermodynamic parameter."
)

// TerminalMismatchKey identifies a single primer-terminal mismatch plus the
// adjacent inward duplex column. Primer and target are in the same orientation
// used by ImperfectDuplex: primer is 5'→3' and target is 3'→5'.
//
// PrimerEnd is '5' or '3' relative to the primer. P and T are the terminal
// primer and target bases at the mismatch column. PNeighbor and TNeighbor are
// the inward adjacent bases, or 'N' when the context is unavailable.
type TerminalMismatchKey struct {
	PrimerEnd byte
	P         byte
	T         byte
	PNeighbor byte
	TNeighbor byte
}

// TerminalMismatchParameter stores one terminal mismatch scoring term. A
// parameter may be expressed as a ΔTm penalty, a ΔΔG37 penalty, or both. The
// current built-in fallback uses only DeltaTmC.
type TerminalMismatchParameter struct {
	Key               TerminalMismatchKey
	DeltaTmC          float64
	DeltaDeltaG37kcal float64
	HasDeltaTm        bool
	HasDeltaDeltaG37  bool
	Source            string
	ParameterSet      string
	Citation          string
	Note              string
}

var terminalMismatchParametersByKey = map[TerminalMismatchKey]TerminalMismatchParameter{}

// CuratedTerminalMismatchParameters is intentionally empty until a verified
// sequence-context terminal-mismatch thermodynamic table is added. The lookup
// layer is still exposed so scoring/reporting code can distinguish future
// table-backed terms from the current heuristic fallback.
var CuratedTerminalMismatchParameters = []TerminalMismatchParameter{}

func init() {
	for _, p := range CuratedTerminalMismatchParameters {
		p.Key = normalizeTerminalMismatchKey(p.Key)
		terminalMismatchParametersByKey[p.Key] = p
	}
}

// LookupTerminalMismatchParameter returns a curated terminal mismatch parameter
// for an exact key or for a key with wildcarded inward-neighbor context. 'N' is
// treated as a wildcard only for PNeighbor/TNeighbor, not for the central
// mismatch bases.
func LookupTerminalMismatchParameter(key TerminalMismatchKey) (TerminalMismatchParameter, bool) {
	key = normalizeTerminalMismatchKey(key)
	if !isTerminalMismatchKeyUsable(key, false) {
		return TerminalMismatchParameter{}, false
	}
	for _, candidate := range terminalMismatchLookupCandidates(key) {
		if p, ok := terminalMismatchParametersByKey[candidate]; ok {
			return p, true
		}
	}
	return TerminalMismatchParameter{}, false
}

// LookupTerminalMismatchParameterWithFallback returns a curated parameter when
// available; otherwise it returns the named ipcr heuristic parameter that matches
// the side-specific terminal mismatch penalty in ImperfectDuplexOptions.
func LookupTerminalMismatchParameterWithFallback(key TerminalMismatchKey, opts ImperfectDuplexOptions) (TerminalMismatchParameter, bool) {
	if p, ok := LookupTerminalMismatchParameter(key); ok {
		return p, true
	}
	return LookupTerminalMismatchHeuristicParameter(key, opts)
}

// LookupTerminalMismatchHeuristicParameter returns the current side-specific
// fixed-ΔTm terminal mismatch fallback as a named parameter.
func LookupTerminalMismatchHeuristicParameter(key TerminalMismatchKey, opts ImperfectDuplexOptions) (TerminalMismatchParameter, bool) {
	key = normalizeTerminalMismatchKey(key)
	if !isTerminalMismatchKeyUsable(key, true) {
		return TerminalMismatchParameter{}, false
	}

	opts = opts.withDefaults()
	penalty := 0.0
	switch key.PrimerEnd {
	case TerminalMismatchPrimer5Prime:
		penalty = opts.FivePrimeTerminalPenaltyC
	case TerminalMismatchPrimer3Prime:
		penalty = opts.ThreePrimeTerminalPenaltyC
	default:
		return TerminalMismatchParameter{}, false
	}
	if penalty <= 0 {
		return TerminalMismatchParameter{}, false
	}

	return TerminalMismatchParameter{
		Key:          key,
		DeltaTmC:     penalty,
		HasDeltaTm:   true,
		Source:       TerminalMismatchSourceHeuristicPenalty,
		ParameterSet: TerminalMismatchParameterSetHeuristicV1,
		Citation:     terminalMismatchCitationHeuristicV1,
		Note:         terminalMismatchNoteHeuristicV1,
	}, true
}

// TerminalMismatchKeyForPosition builds a terminal mismatch key from primer-
// aligned sequences. It returns false unless pos is a terminal column containing
// a non-Watson-Crick primer/target pair.
func TerminalMismatchKeyForPosition(primer5to3, target3to5 string, pos int) (TerminalMismatchKey, bool) {
	if len(primer5to3) == 0 || len(primer5to3) != len(target3to5) || pos < 0 || pos >= len(primer5to3) {
		return TerminalMismatchKey{}, false
	}
	p := normalizeBase(primer5to3[pos])
	t := normalizeBase(target3to5[pos])
	if !isACGT(p) || !isNT(t) || wc(p, t) {
		return TerminalMismatchKey{}, false
	}

	switch pos {
	case len(primer5to3) - 1:
		return normalizeTerminalMismatchKey(TerminalMismatchKey{
			PrimerEnd: TerminalMismatchPrimer3Prime,
			P:         p,
			T:         t,
			PNeighbor: mismatchAt(primer5to3, pos-1),
			TNeighbor: mismatchAt(target3to5, pos-1),
		}), true
	case 0:
		return normalizeTerminalMismatchKey(TerminalMismatchKey{
			PrimerEnd: TerminalMismatchPrimer5Prime,
			P:         p,
			T:         t,
			PNeighbor: mismatchAt(primer5to3, 1),
			TNeighbor: mismatchAt(target3to5, 1),
		}), true
	default:
		return TerminalMismatchKey{}, false
	}
}

func terminalMismatchLookupCandidates(key TerminalMismatchKey) []TerminalMismatchKey {
	key = normalizeTerminalMismatchKey(key)
	return []TerminalMismatchKey{
		key,
		{PrimerEnd: key.PrimerEnd, P: key.P, T: key.T, PNeighbor: key.PNeighbor, TNeighbor: 'N'},
		{PrimerEnd: key.PrimerEnd, P: key.P, T: key.T, PNeighbor: 'N', TNeighbor: key.TNeighbor},
		{PrimerEnd: key.PrimerEnd, P: key.P, T: key.T, PNeighbor: 'N', TNeighbor: 'N'},
	}
}

func normalizeTerminalMismatchKey(key TerminalMismatchKey) TerminalMismatchKey {
	key.PrimerEnd = normalizeTerminalMismatchEnd(key.PrimerEnd)
	key.P = normalizeBase(key.P)
	key.T = normalizeBase(key.T)
	key.PNeighbor = normalizeBase(key.PNeighbor)
	key.TNeighbor = normalizeBase(key.TNeighbor)
	return key
}

func normalizeTerminalMismatchEnd(end byte) byte {
	switch end {
	case TerminalMismatchPrimer5Prime:
		return TerminalMismatchPrimer5Prime
	case TerminalMismatchPrimer3Prime:
		return TerminalMismatchPrimer3Prime
	default:
		return 0
	}
}

func isTerminalMismatchKeyUsable(key TerminalMismatchKey, allowUnknownTarget bool) bool {
	if key.PrimerEnd != TerminalMismatchPrimer5Prime && key.PrimerEnd != TerminalMismatchPrimer3Prime {
		return false
	}
	if !isACGT(key.P) {
		return false
	}
	if allowUnknownTarget {
		if !isNT(key.T) {
			return false
		}
	} else if !isACGT(key.T) {
		return false
	}
	return !wc(key.P, key.T)
}
