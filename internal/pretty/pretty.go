package pretty

import (
	"fmt"
	"strings"

	"ipcr-core/engine"
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
	ShowCaret bool
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

// DefaultOptions keeps the current look & feel (matching what you already approved).
var DefaultOptions = Options{
	MaxGap:         95,
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

func reverseString(s string) string {
	rs := []rune(s)
	for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
		rs[i], rs[j] = rs[j], rs[i]
	}
	return string(rs)
}

func complementString(s string) string {
	out := make([]byte, len(s))
	for i := range s {
		switch s[i] {
		case 'A':
			out[i] = 'T'
		case 'T':
			out[i] = 'A'
		case 'C':
			out[i] = 'G'
		case 'G':
			out[i] = 'C'
		case 'R':
			out[i] = 'Y'
		case 'Y':
			out[i] = 'R'
		case 'S':
			out[i] = 'S'
		case 'W':
			out[i] = 'W'
		case 'K':
			out[i] = 'M'
		case 'M':
			out[i] = 'K'
		case 'B':
			out[i] = 'V'
		case 'V':
			out[i] = 'B'
		case 'D':
			out[i] = 'H'
		case 'H':
			out[i] = 'D'
		case 'N':
			out[i] = 'N'
		default:
			out[i] = s[i]
		}
	}
	return string(out)
}

func isACGT(b byte) bool { return b == 'A' || b == 'C' || b == 'G' || b == 'T' }

// Primer bars under a site (ipcr semantics)
func matchLineAmbig(primer, site string, mismIdx []int, exactGlyph, partialGlyph string) string {
	n := len(primer)
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
		if isACGT(primer[i]) {
			b.WriteString(exactGlyph)
		} else {
			b.WriteString(partialGlyph)
		}
	}
	return b.String()
}

// scale an interior offset into the capped printed width (endpoint-preserving)
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

// intsCSV for printing mismatch indexes in the summary.
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
		fmt.Fprintf(&b, "%s(pretty not available: sites missing)\n\n", linePrefix)
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
	fmt.Fprintf(&b, "%s%s%s%s\n", linePrefix, prefixPlus, p.FwdPrimer, suffixPlus)

	// 2) Bars under forward primer
	fmt.Fprintf(&b, "%s%s%s%s\n",
		linePrefix,
		strings.Repeat(" ", len(prefixPlus)),
		matchLineAmbig(p.FwdPrimer, p.FwdSite, p.FwdMismatchIdx, opt.ExactGlyphOrDefault(), opt.PartialGlyphOrDefault()),
		arrowRight,
	)

	// 3) (+) genomic line
	fmt.Fprintf(&b, "%s%s%s%s%s # (+)\n",
		linePrefix, prefixPlus, p.FwdSite, strings.Repeat(dot, innerPlus), suffixPlus,
	)

	// 4) (−) genomic line
	minusSite := complementString(p.RevSite)
	fmt.Fprintf(&b, "%s%s%s%s%s # (-)\n",
		linePrefix, prefixMinus, strings.Repeat(dot, innerMinus), minusSite, suffixMinus,
	)

	// 5) bars for reverse primer
	siteStart := len(prefixMinus) + innerMinus
	revBars := reverseString(matchLineAmbig(p.RevPrimer, p.RevSite, p.RevMismatchIdx, opt.ExactGlyphOrDefault(), opt.PartialGlyphOrDefault()))
	padBars := siteStart - len(arrowLeft)
	if padBars < 0 {
		padBars = 0
	}
	fmt.Fprintf(&b, "%s%s%s%s\n", linePrefix, strings.Repeat(" ", padBars), arrowLeft, revBars)

	// 6) reverse primer shown 3'→5'
	revPrimerDisplayed := reverseString(p.RevPrimer)
	padPrimer := siteStart - len(prefixMinus)
	if padPrimer < 0 {
		padPrimer = 0
	}
	fmt.Fprintf(&b, "%s%s%s%s%s\n", linePrefix, strings.Repeat(" ", padPrimer), prefixMinus, revPrimerDisplayed, suffixMinus)

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
		fmt.Fprintf(&b, "%s(pretty not available: sites missing)\n\n", linePrefix)
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
	fmt.Fprintf(&b, "%s%s%s%s\n", linePrefix, prefixPlus, p.FwdPrimer, suffixPlus)

	// 2) Bars under forward primer
	fmt.Fprintf(&b, "%s%s%s%s\n",
		linePrefix,
		strings.Repeat(" ", len(prefixPlus)),
		matchLineAmbig(p.FwdPrimer, p.FwdSite, p.FwdMismatchIdx, opt.ExactGlyphOrDefault(), opt.PartialGlyphOrDefault()),
		arrowRight,
	)

	// Prepare overlayable interiors
	plusInterior := strings.Repeat(dot, innerPlus)
	minusInterior := strings.Repeat(dot, innerMinus)

	type overlay struct {
		onPlus bool
		colAbs int
		seg    string
	}
	var ov *overlay

	// Compute overlay position and visible segment
	if ann.Found && p.Length > 0 && len(ann.Site) > 0 && opt.ShowProbeInline {
		start := ann.Pos
		site := ann.Site

		if ann.Strand == "+" {
			remStart := start
			if remStart < aLen {
				clip := aLen - remStart
				if clip < len(site) {
					site = site[clip:]
					remStart = aLen
				} else {
					site = ""
				}
			}
			if len(site) > 0 && remStart >= aLen && remStart < aLen+interior && innerPlus > 0 && interior > 0 {
				off := remStart - aLen
				s := scalePos(off, interior, innerPlus)
				slot := innerPlus - s
				seg := site
				if len(seg) > slot {
					seg = seg[:slot]
				}
				if len(seg) > 0 {
					ir := []rune(plusInterior)
					copy(ir[s:], []rune(seg))
					plusInterior = string(ir)
					ov = &overlay{onPlus: true, colAbs: len(prefixPlus) + aLen + s, seg: seg}
				}
			}
		} else { // "-"
			remStart := start
			if remStart < aLen {
				clip := aLen - remStart
				if clip < len(site) {
					site = site[clip:]
					remStart = aLen
				} else {
					site = ""
				}
			}
			if len(site) > 0 && remStart >= aLen && remStart < aLen+interior && innerMinus > 0 && interior > 0 {
				off := remStart - aLen
				s := scalePos(off, interior, innerMinus)
				slot := innerMinus - s
				seg := site
				if len(seg) > slot {
					seg = seg[:slot]
				}
				if len(seg) > 0 {
					ir := []rune(minusInterior)
					copy(ir[s:], []rune(seg))
					minusInterior = string(ir)
					ov = &overlay{onPlus: false, colAbs: len(prefixMinus) + s, seg: seg}
				}
			}
		}
	}

	// 3) (+) line (with overlay)
	fmt.Fprintf(&b, "%s%s%s%s%s # (+)\n",
		linePrefix, prefixPlus, p.FwdSite, plusInterior, suffixPlus,
	)

	// Optional caret under (+)
	if opt.ShowCaret && ov != nil && ov.onPlus {
		startCol := ov.colAbs
		if startCol < 0 {
			startCol = 0
		}
		g := opt.CaretGlyph
		if g == "" {
			g = DefaultOptions.CaretGlyph
		}
		fmt.Fprintf(&b, "%s%s%s\n", linePrefix, strings.Repeat(" ", startCol), strings.Repeat(g, len(ov.seg)))
	}

	// 4) (−) line (with overlay) — FIXED to use minusInterior
	minusSite := complementString(p.RevSite)
	fmt.Fprintf(&b, "%s%s%s%s%s # (-)\n",
		linePrefix, prefixMinus, minusInterior, minusSite, suffixMinus,
	)

	// Optional caret under (−)
	if opt.ShowCaret && ov != nil && !ov.onPlus {
		startCol := ov.colAbs
		if startCol < 0 {
			startCol = 0
		}
		g := opt.CaretGlyph
		if g == "" {
			g = DefaultOptions.CaretGlyph
		}
		fmt.Fprintf(&b, "%s%s%s\n", linePrefix, strings.Repeat(" ", startCol), strings.Repeat(g, len(ov.seg)))
	}

	// Composite bars: probe bars + reverse-primer bars
	siteStart := len(prefixMinus) + innerMinus
	revBars := reverseString(matchLineAmbig(p.RevPrimer, p.RevSite, p.RevMismatchIdx, opt.ExactGlyphOrDefault(), opt.PartialGlyphOrDefault()))
	arrowStartCol := siteStart - len(arrowLeft)

	width := arrowStartCol + len(arrowLeft) + len(revBars)
	if ov != nil && ov.colAbs+len(ov.seg) > width {
		width = ov.colAbs + len(ov.seg)
	}
	line := make([]rune, width)
	for i := range line {
		line[i] = ' '
	}
	if opt.ShowProbeBars && ov != nil {
		for i := 0; i < len(ov.seg); i++ {
			pos := ov.colAbs + i
			if pos >= 0 && pos < width {
				line[pos] = []rune(opt.ExactGlyphOrDefault())[0]
			}
		}
	}
	for i, r := range arrowLeft {
		line[arrowStartCol+i] = r
	}
	for i, r := range revBars {
		line[arrowStartCol+len(arrowLeft)+i] = r
	}
	fmt.Fprintf(&b, "%s%s\n", linePrefix, string(line))

	// Sequence row: left probe block (optional) + right reverse-primer block
	if opt.ShowProbeSeqRow {
		leftBlock := ""
		leftStartCol := 0
		if ov != nil {
			if ann.Strand == "+" {
				leftBlock = "5'-" + ov.seg + "-3'"
			} else {
				leftBlock = "3'-" + ov.seg + "-5'"
			}
			leftStartCol = ov.colAbs - len("5'-")
			if leftStartCol < 0 {
				leftStartCol = 0
			}
		}
		rightSeq := reverseString(p.RevPrimer)
		rightBlock := "3'-" + rightSeq + "-5'"
		rightStartCol := arrowStartCol + len(arrowLeft) - len("3'-")
		if rightStartCol < 0 {
			rightStartCol = 0
		}
		w2 := rightStartCol + len(rightBlock)
		if ov != nil && leftStartCol+len(leftBlock) > w2 {
			w2 = leftStartCol + len(leftBlock)
		}
		line2 := make([]rune, w2)
		for i := range line2 {
			line2[i] = ' '
		}
		if ov != nil {
			for i, r := range leftBlock {
				line2[leftStartCol+i] = r
			}
		}
		for i, r := range rightBlock {
			line2[rightStartCol+i] = r
		}
		fmt.Fprintf(&b, "%s%s\n", linePrefix, string(line2))
	} else {
		revPrimerDisplayed := reverseString(p.RevPrimer)
		padPrimer := siteStart - len(prefixMinus)
		if padPrimer < 0 {
			padPrimer = 0
		}
		fmt.Fprintf(&b, "%s%s%s%s%s\n", linePrefix, strings.Repeat(" ", padPrimer), prefixMinus, revPrimerDisplayed, suffixMinus)
	}

	// Summary
	if ann.Found {
		fmt.Fprintf(&b, "%sprobe %q (%s) pos=%d mm=%d site=%s fwd_mm=%d@[%s] rev_mm=%d@[%s]\n",
			linePrefix,
			ann.Name, ann.Strand, ann.Pos, ann.MM, ann.Site,
			p.FwdMM, intsCSV(p.FwdMismatchIdx),
			p.RevMM, intsCSV(p.RevMismatchIdx),
		)
	} else {
		fmt.Fprintf(&b, "%sprobe %q NOT FOUND fwd_mm=%d@[%s] rev_mm=%d@[%s]\n",
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
