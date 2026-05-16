package thermo

const (
	// DanglingEndParameterSetSantaLuciaHicks2004V1 identifies the DNA/DNA
	// terminal dangling-end nearest-neighbor table from SantaLucia & Hicks 2004,
	// Table 3, in 1 M NaCl.
	DanglingEndParameterSetSantaLuciaHicks2004V1 = "santalucia-hicks-2004-dna-dangling-ends-v1"

	// DanglingEndStrand5Prime means the unpaired base is a 5' dangling end on the
	// strand carrying the dangling base, as in 5'-XA-3'/3'-T-5'.
	DanglingEndStrand5Prime byte = '5'

	// DanglingEndStrand3Prime means the unpaired base is a 3' dangling end on the
	// strand carrying the dangling base, as in 5'-AX-3'/3'-T-5'.
	DanglingEndStrand3Prime byte = '3'

	// DanglingEndSideTemplate5Prime means the unpaired target/template base is on
	// the target strand's 5' side. In primer-aligned coordinates this is adjacent
	// to the primer 3' end.
	DanglingEndSideTemplate5Prime byte = DanglingEndStrand5Prime

	// DanglingEndSideTemplate3Prime means the unpaired target/template base is on
	// the target strand's 3' side. In primer-aligned coordinates this is adjacent
	// to the primer 5' end.
	DanglingEndSideTemplate3Prime byte = DanglingEndStrand3Prime
)

const (
	danglingEndSourceSantaLuciaHicks2004Table3 = "santalucia-hicks-2004-table-3"
	danglingEndCitationSantaLuciaHicks2004     = "SantaLucia J Jr, Hicks D. The thermodynamics of DNA structural motifs. Annu Rev Biophys Biomol Struct. 2004;33:415-440. Table 3. doi:10.1146/annurev.biophys.32.110601.141800"
	danglingEndNoteSantaLuciaHicks2004         = "Terminal dangling-end nearest-neighbor increment next to a Watson-Crick DNA pair in 1 M NaCl; Table 3 reports ΔH° and ΔG°37. ΔS° is computed from ΔH° and ΔG°37 at 310.15 K."
)

// DanglingEndKey identifies one terminal dangling-end nearest-neighbor term in
// the orientation of the strand carrying the dangling base. PairedBase is the
// Watson-Crick base adjacent to the dangling base on that same strand;
// OppositeBase is the base on the opposite strand.
type DanglingEndKey struct {
	StrandEnd    byte
	DanglingBase byte
	PairedBase   byte
	OppositeBase byte
}

// DanglingEndParameter stores one SantaLucia-Hicks terminal dangling-end term.
// DeltaG37kcal is an additive endpoint increment, not a penalty; it may be
// favorable (negative) or unfavorable (positive).
type DanglingEndParameter struct {
	Key          DanglingEndKey
	Motif        string
	DeltaHkcal   float64
	DeltaScalK   float64
	DeltaG37kcal float64
	Source       string
	ParameterSet string
	Citation     string
	Note         string
}

var danglingEndParametersByKey = map[DanglingEndKey]DanglingEndParameter{}

// CuratedDanglingEnds contains the complete SantaLucia-Hicks 2004 Table 3 DNA/DNA
// terminal dangling-end ΔH° and ΔG°37 table, in the orientation of the dangling
// strand.
var CuratedDanglingEnds = []DanglingEndParameter{
	// 5' dangling ends: 5'-XA-3'/3'-T-5'.
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'A', 'A', 'T', 0.2, -0.51),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'C', 'A', 'T', 0.6, -0.42),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'G', 'A', 'T', -1.1, -0.62),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'T', 'A', 'T', -6.9, -0.71),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'A', 'C', 'G', -6.3, -0.96),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'C', 'C', 'G', -4.4, -0.52),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'G', 'C', 'G', -5.1, -0.72),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'T', 'C', 'G', -4.0, -0.58),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'A', 'G', 'C', -3.7, -0.58),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'C', 'G', 'C', -4.0, -0.34),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'G', 'G', 'C', -3.9, -0.56),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'T', 'G', 'C', -4.9, -0.61),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'A', 'T', 'A', -2.9, -0.50),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'C', 'T', 'A', -4.1, -0.02),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'G', 'T', 'A', -4.2, 0.48),
	santaLuciaHicksDanglingEnd(DanglingEndStrand5Prime, 'T', 'T', 'A', -0.2, -0.10),

	// 3' dangling ends: 5'-AX-3'/3'-T-5'.
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'A', 'A', 'T', -0.5, -0.12),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'C', 'A', 'T', 4.7, 0.28),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'G', 'A', 'T', -4.1, -0.01),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'T', 'A', 'T', -3.8, 0.13),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'A', 'C', 'G', -5.9, -0.82),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'C', 'C', 'G', -2.6, -0.31),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'G', 'C', 'G', -3.2, -0.01),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'T', 'C', 'G', -5.2, -0.52),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'A', 'G', 'C', -2.1, -0.92),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'C', 'G', 'C', -0.2, -0.23),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'G', 'G', 'C', -3.9, -0.44),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'T', 'G', 'C', -4.4, -0.35),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'A', 'T', 'A', -0.7, -0.48),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'C', 'T', 'A', 4.4, -0.19),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'G', 'T', 'A', -1.6, -0.50),
	santaLuciaHicksDanglingEnd(DanglingEndStrand3Prime, 'T', 'T', 'A', 2.9, -0.29),
}

// CuratedDanglingEndParameters is retained as a descriptive alias for tests and
// callers that want the whole table.
var CuratedDanglingEndParameters = CuratedDanglingEnds

func init() {
	for _, p := range CuratedDanglingEnds {
		danglingEndParametersByKey[p.Key] = p
	}
}

func santaLuciaHicksDanglingEnd(strandEnd, danglingBase, pairedBase, oppositeBase byte, deltaH, deltaG37 float64) DanglingEndParameter {
	key := DanglingEndKey{
		StrandEnd:    normalizeDanglingEndSide(strandEnd),
		DanglingBase: normalizeBase(danglingBase),
		PairedBase:   normalizeBase(pairedBase),
		OppositeBase: normalizeBase(oppositeBase),
	}
	return DanglingEndParameter{
		Key:          key,
		Motif:        danglingEndMotif(key),
		DeltaHkcal:   deltaH,
		DeltaScalK:   (deltaH - deltaG37) * 1000.0 / 310.15,
		DeltaG37kcal: deltaG37,
		Source:       danglingEndSourceSantaLuciaHicks2004Table3,
		ParameterSet: DanglingEndParameterSetSantaLuciaHicks2004V1,
		Citation:     danglingEndCitationSantaLuciaHicks2004,
		Note:         danglingEndNoteSantaLuciaHicks2004,
	}
}

func danglingEndMotif(key DanglingEndKey) string {
	key.StrandEnd = normalizeDanglingEndSide(key.StrandEnd)
	key.DanglingBase = normalizeBase(key.DanglingBase)
	key.PairedBase = normalizeBase(key.PairedBase)
	key.OppositeBase = normalizeBase(key.OppositeBase)
	switch key.StrandEnd {
	case DanglingEndStrand5Prime:
		return string([]byte{key.DanglingBase, key.PairedBase}) + "/" + string([]byte{key.OppositeBase})
	case DanglingEndStrand3Prime:
		return string([]byte{key.PairedBase, key.DanglingBase}) + "/" + string([]byte{key.OppositeBase})
	default:
		return ""
	}
}

// LookupDanglingEndParameter returns a terminal dangling-end parameter keyed in
// the orientation of the dangling strand.
func LookupDanglingEndParameter(key DanglingEndKey) (DanglingEndParameter, bool) {
	key.StrandEnd = normalizeDanglingEndSide(key.StrandEnd)
	key.DanglingBase = normalizeBase(key.DanglingBase)
	key.PairedBase = normalizeBase(key.PairedBase)
	key.OppositeBase = normalizeBase(key.OppositeBase)
	p, ok := danglingEndParametersByKey[key]
	return p, ok
}

// LookupTemplateDanglingEnd returns the table-backed terminal dangling-end term
// for a target/template dangling base next to a Watson-Crick closing pair. The
// side argument is the target/template strand side ('5' or '3').
func LookupTemplateDanglingEnd(side, x, primerBase, targetBase byte) (DanglingEndParameter, bool) {
	key := DanglingEndKey{
		StrandEnd:    normalizeDanglingEndSide(side),
		DanglingBase: normalizeBase(x),
		PairedBase:   normalizeBase(targetBase),
		OppositeBase: normalizeBase(primerBase),
	}
	if key.StrandEnd == 0 || key.DanglingBase == 'N' || key.PairedBase == 'N' || key.OppositeBase == 'N' {
		return DanglingEndParameter{}, false
	}
	if !wc(key.OppositeBase, key.PairedBase) {
		return DanglingEndParameter{}, false
	}
	return LookupDanglingEndParameter(key)
}

// LookupTemplateDanglingEndParameter maps a primer-side label to the target
// dangling-end table. A target base next to the primer 5' end is a target 3'
// dangling end; a target base next to the primer 3' end is a target 5' dangling
// end.
func LookupTemplateDanglingEndParameter(side string, danglingBase, terminalPrimerBase, terminalTargetBase byte) (DanglingEndParameter, bool) {
	switch side {
	case "primer-5p":
		return LookupTemplateDanglingEnd(DanglingEndSideTemplate3Prime, danglingBase, terminalPrimerBase, terminalTargetBase)
	case "primer-3p":
		return LookupTemplateDanglingEnd(DanglingEndSideTemplate5Prime, danglingBase, terminalPrimerBase, terminalTargetBase)
	default:
		return DanglingEndParameter{}, false
	}
}

func normalizeDanglingEndSide(side byte) byte {
	switch side {
	case DanglingEndStrand5Prime:
		return DanglingEndStrand5Prime
	case DanglingEndStrand3Prime:
		return DanglingEndStrand3Prime
	default:
		return 0
	}
}
