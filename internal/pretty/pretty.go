package pretty

import (
	"fmt"
	"ipcr-core/engine"
	"ipcr-core/primer"
	"strings"
)

// ProbeAnnotation carries the probe overlay to render on top of a Product.
type ProbeAnnotation struct {
	Name   string
	Seq    string // 5'→3' as provided
	Found  bool
	Strand string // "+" or "-"
	Pos    int    // 0-based in the amplicon (plus orientation)
	MM     int
	Site   string // matched site in amplicon (plus orientation)
}

// Options control the ASCII rendering.
type Options struct {
	// Interior width cap for readability (dots section). If <=0, use default (95).
	MaxGap int

	// Inline the probe letters into the genomic line (overwriting dots).
	ShowProbeInline bool

	// Draw a caret track (^^^^^) under the genomic line at the probe span.
	ShowCaret  bool
	CaretGlyph string // default "^"

	// Draw probe bars '|||||' on the composite bars row (reverse-primer bars always shown).
	ShowProbeBars bool

	// Draw the left probe sequence block on the “sequence row”
	// ("5'-...-3'" for +, "3'-...-5'" for −). The reverse-primer block is always drawn.
	ShowProbeSeqRow bool

	// Glyphs
	ExactGlyph   string // default "|"
	PartialGlyph string // default "¦"
	DotGlyph     string // default "."
}

// DefaultOptions keeps the current look & feel.
var DefaultOptions = Options{
	MaxGap:          95,
	ShowProbeInline: true,
	ShowCaret:       false,
	CaretGlyph:      "^",
	ShowProbeBars:   true,
	ShowProbeSeqRow: true,
	ExactGlyph:      "|",
	PartialGlyph:    "¦",
	DotGlyph:        ".",
}

const (
	minInterPrimerGap = 5
	linePrefix        = "# "
)

// reverseString reverses a string (rune-aware).
func reverseString(s string) string {
	rs := []rune(s)
	for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
		rs[i], rs[j] = rs[j], rs[i]
	}
	return string(rs)
}

// comp5to3 returns the **complement** (not reverse-complement) of a 5'→3' string.
// Implementation: complement(s) == revcomp(reverse(s)).
func comp5to3(s string) string {
	if s == "" {
		return s
	}
	// reverse bytes (not runes — primer alphabet is ASCII)
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	rc := primer.RevComp(b)
	return string(rc)
}

func isACGT(b byte) bool { return b == 'A' || b == 'C' || b == 'G' || b == 'T' }

// Primer bars under a site (ipcr semantics)
func matchLineAmbig(primerSeq, site string, mismIdx []int, exactGlyph, partialGlyph string) string {
	n := len(primerSeq)
	if n <= 0 {
		return ""
	}
	if len(site) < n {
		n = len(site)
	}
	mism := make(map[int]struct{}, len(mismIdx))
	for _, i := range mismIdx {
		mism[i] = struct{}{}
	}
	var b strings.Builder
	b.Grow(n)
	for i := 0; i < n; i++ {
		if _, bad := mism[i]; bad {
			b.WriteByte(' ')
			continue
		}
		if isACGT(primerSeq[i]) {
			b.WriteString(exactGlyph)
		} else {
			b.WriteString(partialGlyph)
		}
	}
	return b.String()
}

func scalePos(off, interior, inner int) int {
	if interior <= 1 || inner <= 1 {
		return 0
	}
	if off < 0 {
		off = 0
	}
	if off > interior-1 {
		off = interior - 1
	}
	return (off * (inner - 1)) / (interior - 1)
}

func intsCSV(a []int) string {
	if len(a) == 0 {
		return ""
	}
	ss := make([]string, len(a))
	for i, v := range a {
		ss[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(ss, ",")
}

type lineSegment struct {
	col  int
	text string
}

func renderLineSegments(segs ...lineSegment) string {
	var line []rune
	for _, seg := range segs {
		line = putAt(line, seg.col, seg.text)
	}
	return strings.TrimRight(string(line), " ")
}

func putAt(line []rune, col int, text string) []rune {
	if text == "" {
		return line
	}
	runes := []rune(text)
	if col < 0 {
		clip := -col
		if clip >= len(runes) {
			return line
		}
		runes = runes[clip:]
		col = 0
	}
	need := col + len(runes)
	if need > len(line) {
		old := len(line)
		line = append(line, make([]rune, need-old)...)
		for i := old; i < len(line); i++ {
			line[i] = ' '
		}
	}
	for i, r := range runes {
		line[col+i] = r
	}
	return line
}

func firstGlyph(glyph, fallback string) rune {
	if glyph == "" {
		glyph = fallback
	}
	for _, r := range glyph {
		return r
	}
	for _, r := range fallback {
		return r
	}
	return '|'
}

func probeMatchLine(probeSeq, site, exactGlyph, partialGlyph string) string {
	n := len(probeSeq)
	if len(site) < n {
		n = len(site)
	}
	if n <= 0 {
		return ""
	}
	exact := firstGlyph(exactGlyph, DefaultOptions.ExactGlyph)
	partial := firstGlyph(partialGlyph, DefaultOptions.PartialGlyph)

	var b strings.Builder
	b.Grow(n)
	for i := 0; i < n; i++ {
		if !primer.BaseMatch(site[i], probeSeq[i]) {
			b.WriteByte(' ')
			continue
		}
		if isACGT(probeSeq[i]) {
			b.WriteRune(exact)
		} else {
			b.WriteRune(partial)
		}
	}
	return b.String()
}

func probeLabel(name string) string {
	if name == "" {
		return "probe"
	}
	return name
}

func rangesOverlap(aStart, aLen, bStart, bLen int) bool {
	if aLen <= 0 || bLen <= 0 {
		return false
	}
	return aStart < bStart+bLen && bStart < aStart+aLen
}

type probeOverlay struct {
	strand   string
	colAbs   int
	siteSeg  string
	probeSeg string
	bars     string
}

func buildProbeOverlay(ann ProbeAnnotation, aLen, interior, inner, plusOffset, minusOffset int, exactGlyph, partialGlyph string) *probeOverlay {
	if !ann.Found || len(ann.Site) == 0 || interior <= 0 || inner <= 0 {
		return nil
	}

	strand := ann.Strand
	if strand != "-" {
		strand = "+"
	}

	site := ann.Site
	probeSeq := ann.Seq
	if probeSeq == "" {
		probeSeq = site
	}
	if strand == "-" {
		site = comp5to3(site)
		probeSeq = reverseString(probeSeq)
	}

	if len(probeSeq) > len(site) {
		probeSeq = probeSeq[:len(site)]
	} else if len(site) > len(probeSeq) {
		site = site[:len(probeSeq)]
	}

	start := ann.Pos
	if start < aLen {
		clip := aLen - start
		if clip >= len(site) || clip >= len(probeSeq) {
			return nil
		}
		site = site[clip:]
		probeSeq = probeSeq[clip:]
		start = aLen
	}
	if start >= aLen+interior {
		return nil
	}

	off := start - aLen
	scaled := scalePos(off, interior, inner)
	slot := inner - scaled
	if slot <= 0 {
		return nil
	}
	if len(site) > slot {
		site = site[:slot]
	}
	if len(probeSeq) > len(site) {
		probeSeq = probeSeq[:len(site)]
	}
	if len(site) == 0 || len(probeSeq) == 0 {
		return nil
	}

	colAbs := plusOffset + scaled
	if strand == "-" {
		colAbs = minusOffset + scaled
	}

	return &probeOverlay{
		strand:   strand,
		colAbs:   colAbs,
		siteSeg:  site,
		probeSeg: probeSeq,
		bars:     probeMatchLine(probeSeq, site, exactGlyph, partialGlyph),
	}
}

// RenderProductWithOptions prints the ipcr-style block (no probe overlay).
func RenderProductWithOptions(p engine.Product, opt Options) string {
	const (
		prefixPlus  = "5'-"
		suffixPlus  = "-3'"
		prefixMinus = "3'-"
		suffixMinus = "-5'"
		arrowRight  = "-->"
		arrowLeft   = "<--"
	)

	if p.FwdPrimer == "" || p.RevPrimer == "" || p.FwdSite == "" || p.RevSite == "" {
		var b strings.Builder
		_, _ = fmt.Fprintf(&b, "%s(pretty not available: sites missing)\n\n", linePrefix)
		return b.String()
	}

	maxGap := opt.MaxGap
	if maxGap <= 0 {
		maxGap = DefaultOptions.MaxGap
	}
	dot := opt.DotGlyph
	if dot == "" {
		dot = DefaultOptions.DotGlyph
	}

	aLen, bLen := len(p.FwdPrimer), len(p.RevPrimer)
	interior := p.Length - aLen - bLen
	if interior < 0 {
		interior = 0
	}

	inner := maxGap
	if interior < inner {
		inner = interior
	}
	innerMinus := inner
	innerPlus := inner
	if innerMinus < aLen+minInterPrimerGap {
		innerMinus = aLen + minInterPrimerGap
	}
	if innerPlus < bLen+minInterPrimerGap {
		innerPlus = bLen + minInterPrimerGap
	}

	contPlus := aLen + innerPlus
	contMinus := innerMinus + bLen
	if contMinus > contPlus {
		innerPlus += (contMinus - contPlus)
	} else if contPlus > contMinus {
		innerMinus += (contPlus - contMinus)
	}

	var b strings.Builder

	// 1) Forward primer (5'→3')
	_, _ = fmt.Fprintf(&b, "%s%s%s%s\n", linePrefix, prefixPlus, p.FwdPrimer, suffixPlus)

	// 2) Bars under forward primer
	_, _ = fmt.Fprintf(&b, "%s%s%s%s\n",
		linePrefix,
		strings.Repeat(" ", len(prefixPlus)),
		matchLineAmbig(p.FwdPrimer, p.FwdSite, p.FwdMismatchIdx, opt.ExactGlyphOrDefault(), opt.PartialGlyphOrDefault()),
		arrowRight,
	)

	// 3) (+) genomic line
	_, _ = fmt.Fprintf(&b, "%s%s%s%s%s # (+)\n",
		linePrefix, prefixPlus, p.FwdSite, strings.Repeat(dot, innerPlus), suffixPlus,
	)

	// 4) (−) genomic line (complement, not reverse)
	minusSite := comp5to3(p.RevSite)
	_, _ = fmt.Fprintf(&b, "%s%s%s%s%s # (-)\n",
		linePrefix, prefixMinus, strings.Repeat(dot, innerMinus), minusSite, suffixMinus,
	)

	// 5) bars for reverse primer
	siteStart := len(prefixMinus) + innerMinus
	revBars := reverseString(matchLineAmbig(p.RevPrimer, p.RevSite, p.RevMismatchIdx, opt.ExactGlyphOrDefault(), opt.PartialGlyphOrDefault()))
	padBars := siteStart - len(arrowLeft)
	if padBars < 0 {
		padBars = 0
	}
	_, _ = fmt.Fprintf(&b, "%s%s%s%s\n", linePrefix, strings.Repeat(" ", padBars), arrowLeft, revBars)

	// 6) reverse primer shown 3'→5'
	revPrimerDisplayed := reverseString(p.RevPrimer)
	padPrimer := siteStart - len(prefixMinus)
	if padPrimer < 0 {
		padPrimer = 0
	}
	_, _ = fmt.Fprintf(&b, "%s%s%s%s%s\n", linePrefix, strings.Repeat(" ", padPrimer), prefixMinus, revPrimerDisplayed, suffixMinus)

	// spacer
	b.WriteString("#\n")
	return b.String()
}

// RenderProduct keeps backward compat (uses DefaultOptions).
func RenderProduct(p engine.Product) string {
	return RenderProductWithOptions(p, DefaultOptions)
}

// RenderAnnotatedWithOptions prints the ipcr-style block with the probe overlay.
func RenderAnnotatedWithOptions(p engine.Product, ann ProbeAnnotation, opt Options) string {
	const (
		prefixPlus  = "5'-"
		suffixPlus  = "-3'"
		prefixMinus = "3'-"
		suffixMinus = "-5'"
		arrowRight  = "-->"
		arrowLeft   = "<--"
	)

	if p.FwdPrimer == "" || p.RevPrimer == "" || p.FwdSite == "" || p.RevSite == "" {
		var b strings.Builder
		_, _ = fmt.Fprintf(&b, "%s(pretty not available: sites missing)\n\n", linePrefix)
		return b.String()
	}

	maxGap := opt.MaxGap
	if maxGap <= 0 {
		maxGap = DefaultOptions.MaxGap
	}
	dot := opt.DotGlyph
	if dot == "" {
		dot = DefaultOptions.DotGlyph
	}

	exactGlyph := opt.ExactGlyphOrDefault()
	partialGlyph := opt.PartialGlyphOrDefault()

	aLen, bLen := len(p.FwdPrimer), len(p.RevPrimer)
	interior := p.Length - aLen - bLen
	if interior < 0 {
		interior = 0
	}

	inner := maxGap
	if interior < inner {
		inner = interior
	}
	innerMinus := inner
	innerPlus := inner
	if innerMinus < aLen+minInterPrimerGap {
		innerMinus = aLen + minInterPrimerGap
	}
	if innerPlus < bLen+minInterPrimerGap {
		innerPlus = bLen + minInterPrimerGap
	}

	contPlus := aLen + innerPlus
	contMinus := innerMinus + bLen
	if contMinus > contPlus {
		innerPlus += (contMinus - contPlus)
	} else if contPlus > contMinus {
		innerMinus += (contPlus - contMinus)
	}

	minusProbeMode := ann.Found && ann.Strand == "-" && len(ann.Site) > 0
	plusInteriorLen := innerPlus
	minusInteriorLen := innerMinus
	minusProbeOffset := len(prefixMinus)
	if minusProbeMode {
		// A minus-strand probe is still reported in plus-oriented amplicon
		// coordinates. Keep the schematic compact, but include the forward
		// primer footprint on the minus track so probe_pos maps to the same
		// horizontal coordinate users see in the output table. Extend the plus
		// row by the reverse primer footprint so the two genomic rows stay the
		// same visual width.
		plusInteriorLen = innerPlus + bLen
		minusInteriorLen = aLen + innerMinus
		minusProbeOffset = len(prefixMinus) + aLen
	}

	plusInterior := strings.Repeat(dot, plusInteriorLen)
	minusInterior := strings.Repeat(dot, minusInteriorLen)

	plusInteriorStart := len(prefixPlus) + aLen
	minusInteriorStart := len(prefixMinus)
	ovPlus := buildProbeOverlay(ann, aLen, interior, innerPlus, plusInteriorStart, minusProbeOffset, exactGlyph, partialGlyph)
	var ovMinus *probeOverlay
	if ovPlus != nil && ovPlus.strand == "-" {
		ovMinus = buildProbeOverlay(ann, aLen, interior, innerMinus, plusInteriorStart, minusProbeOffset, exactGlyph, partialGlyph)
		ovPlus = nil
	}

	if opt.ShowProbeInline {
		if ovPlus != nil {
			plusInterior = renderLineSegments(
				lineSegment{col: 0, text: plusInterior},
				lineSegment{col: ovPlus.colAbs - plusInteriorStart, text: ovPlus.siteSeg},
			)
		}
		if ovMinus != nil {
			minusInterior = renderLineSegments(
				lineSegment{col: 0, text: minusInterior},
				lineSegment{col: ovMinus.colAbs - minusInteriorStart, text: ovMinus.siteSeg},
			)
		}
	}

	fwdSeqBlock := prefixPlus + p.FwdPrimer + suffixPlus
	fwdBars := matchLineAmbig(p.FwdPrimer, p.FwdSite, p.FwdMismatchIdx, exactGlyph, partialGlyph)
	fwdBarsBlock := fwdBars + arrowRight
	topSeqSegments := []lineSegment{{col: 0, text: fwdSeqBlock}}
	topBarsSegments := []lineSegment{{col: len(prefixPlus), text: fwdBarsBlock}}
	var extraTopSeqSegments []lineSegment
	var extraTopBarsSegments []lineSegment
	if ovPlus != nil {
		probeSeqBlock := prefixPlus + ovPlus.probeSeg + suffixPlus + " " + probeLabel(ann.Name) + " (+)"
		probeSeqCol := ovPlus.colAbs - len(prefixPlus)
		probeBarsCol := ovPlus.colAbs
		probeSeqOverlaps := opt.ShowProbeSeqRow && rangesOverlap(probeSeqCol, len(probeSeqBlock), 0, len(fwdSeqBlock))
		probeBarsOverlaps := opt.ShowProbeBars && rangesOverlap(probeBarsCol, len(ovPlus.bars), len(prefixPlus), len(fwdBarsBlock))
		if probeSeqOverlaps || probeBarsOverlaps {
			if opt.ShowProbeSeqRow {
				extraTopSeqSegments = append(extraTopSeqSegments, lineSegment{col: probeSeqCol, text: probeSeqBlock})
			}
			if opt.ShowProbeBars {
				extraTopBarsSegments = append(extraTopBarsSegments, lineSegment{col: probeBarsCol, text: ovPlus.bars})
			}
		} else {
			if opt.ShowProbeSeqRow {
				topSeqSegments = append(topSeqSegments, lineSegment{col: probeSeqCol, text: probeSeqBlock})
			}
			if opt.ShowProbeBars {
				topBarsSegments = append(topBarsSegments, lineSegment{col: probeBarsCol, text: ovPlus.bars})
			}
		}
	}

	minusSite := comp5to3(p.RevSite)
	siteStart := len(prefixMinus) + minusInteriorLen
	revBars := reverseString(matchLineAmbig(p.RevPrimer, p.RevSite, p.RevMismatchIdx, exactGlyph, partialGlyph))
	arrowStartCol := siteStart - len(arrowLeft)
	if arrowStartCol < 0 {
		arrowStartCol = 0
	}

	revPrimerDisplayed := reverseString(p.RevPrimer)
	rightBlock := prefixMinus + revPrimerDisplayed + suffixMinus
	rightStartCol := arrowStartCol + len(arrowLeft) - len(prefixMinus)
	if rightStartCol < 0 {
		rightStartCol = 0
	}

	bottomBarsSegments := []lineSegment{{col: arrowStartCol, text: arrowLeft + revBars}}
	bottomSeqSegments := []lineSegment{{col: rightStartCol, text: rightBlock}}
	var extraBottomBarsSegments []lineSegment
	var extraBottomSeqSegments []lineSegment
	if ovMinus != nil {
		probeSeqLabel := probeLabel(ann.Name) + " (-) "
		probeSeqBlock := probeSeqLabel + prefixMinus + ovMinus.probeSeg + suffixMinus
		probeSeqCol := ovMinus.colAbs - len(probeSeqLabel) - len(prefixMinus)
		probeBarsCol := ovMinus.colAbs
		probeSeqOverlaps := opt.ShowProbeSeqRow && rangesOverlap(probeSeqCol, len(probeSeqBlock), rightStartCol, len(rightBlock))
		probeBarsOverlaps := opt.ShowProbeBars && rangesOverlap(probeBarsCol, len(ovMinus.bars), arrowStartCol, len(arrowLeft)+len(revBars))
		if probeSeqOverlaps || probeBarsOverlaps {
			if opt.ShowProbeBars {
				extraBottomBarsSegments = append(extraBottomBarsSegments, lineSegment{col: probeBarsCol, text: ovMinus.bars})
			}
			if opt.ShowProbeSeqRow {
				extraBottomSeqSegments = append(extraBottomSeqSegments, lineSegment{col: probeSeqCol, text: probeSeqBlock})
			}
		} else {
			if opt.ShowProbeBars {
				bottomBarsSegments = append(bottomBarsSegments, lineSegment{col: probeBarsCol, text: ovMinus.bars})
			}
			if opt.ShowProbeSeqRow {
				bottomSeqSegments = append(bottomSeqSegments, lineSegment{col: probeSeqCol, text: probeSeqBlock})
			}
		}
	}

	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "%s%s\n", linePrefix, renderLineSegments(topSeqSegments...))
	_, _ = fmt.Fprintf(&b, "%s%s\n", linePrefix, renderLineSegments(topBarsSegments...))
	if len(extraTopSeqSegments) > 0 || len(extraTopBarsSegments) > 0 {
		_, _ = fmt.Fprintf(&b, "%s%s\n", linePrefix, renderLineSegments(extraTopSeqSegments...))
		_, _ = fmt.Fprintf(&b, "%s%s\n", linePrefix, renderLineSegments(extraTopBarsSegments...))
	}

	_, _ = fmt.Fprintf(&b, "%s%s%s%s%s # (+)\n",
		linePrefix, prefixPlus, p.FwdSite, plusInterior, suffixPlus,
	)

	if opt.ShowCaret && ovPlus != nil {
		g := opt.CaretGlyph
		if g == "" {
			g = DefaultOptions.CaretGlyph
		}
		_, _ = fmt.Fprintf(&b, "%s%s%s\n", linePrefix, strings.Repeat(" ", ovPlus.colAbs), strings.Repeat(g, len(ovPlus.siteSeg)))
	}

	_, _ = fmt.Fprintf(&b, "%s%s%s%s%s # (-)\n",
		linePrefix, prefixMinus, minusInterior, minusSite, suffixMinus,
	)

	if opt.ShowCaret && ovMinus != nil {
		g := opt.CaretGlyph
		if g == "" {
			g = DefaultOptions.CaretGlyph
		}
		_, _ = fmt.Fprintf(&b, "%s%s%s\n", linePrefix, strings.Repeat(" ", ovMinus.colAbs), strings.Repeat(g, len(ovMinus.siteSeg)))
	}

	if len(extraBottomBarsSegments) > 0 || len(extraBottomSeqSegments) > 0 {
		_, _ = fmt.Fprintf(&b, "%s%s\n", linePrefix, renderLineSegments(extraBottomBarsSegments...))
		_, _ = fmt.Fprintf(&b, "%s%s\n", linePrefix, renderLineSegments(extraBottomSeqSegments...))
	}
	_, _ = fmt.Fprintf(&b, "%s%s\n", linePrefix, renderLineSegments(bottomBarsSegments...))
	_, _ = fmt.Fprintf(&b, "%s%s\n", linePrefix, renderLineSegments(bottomSeqSegments...))

	if ann.Found {
		_, _ = fmt.Fprintf(&b, "%sprobe %q (%s) pos=%d mm=%d site=%s fwd_mm=%d@[%s] rev_mm=%d@[%s]\n",
			linePrefix,
			ann.Name, ann.Strand, ann.Pos, ann.MM, ann.Site,
			p.FwdMM, intsCSV(p.FwdMismatchIdx),
			p.RevMM, intsCSV(p.RevMismatchIdx),
		)
	} else {
		_, _ = fmt.Fprintf(&b, "%sprobe %q NOT FOUND fwd_mm=%d@[%s] rev_mm=%d@[%s]\n",
			linePrefix, ann.Name,
			p.FwdMM, intsCSV(p.FwdMismatchIdx),
			p.RevMM, intsCSV(p.RevMismatchIdx),
		)
	}
	b.WriteString("#\n")
	return b.String()
}

// RenderAnnotated keeps backward compat (uses DefaultOptions).
func RenderAnnotated(p engine.Product, ann ProbeAnnotation) string {
	return RenderAnnotatedWithOptions(p, ann, DefaultOptions)
}

// helpers for default glyphs
func (o Options) ExactGlyphOrDefault() string {
	if o.ExactGlyph != "" {
		return o.ExactGlyph
	}
	return DefaultOptions.ExactGlyph
}

func (o Options) PartialGlyphOrDefault() string {
	if o.PartialGlyph != "" {
		return o.PartialGlyph
	}
	return DefaultOptions.PartialGlyph
}
