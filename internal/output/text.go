package output

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"ipcr/internal/engine"
)

/*
Pretty text rendering (biologically accurate):

 (+) strand is shown 5'→3' left→right
 (−) strand is shown 3'→5' left→right (antiparallel under (+))

We print:
  1) A primer (5'→3') and its match bars with a rightward arrow.
  2) The (+) genomic line.
  3) The (−) genomic line, complementary and aligned column‑wise with (+).
  4) The reverse primer (3'→5') and its bars with a leftward arrow.

For readability, the interior gap ('.') is capped; for short amplicons we use
the true interior. We render one fewer '.' on the (+) line so the (−) site
starts exactly where expected visually.

Change: “partial-ish” matches (ambiguous primer base that still matches an
unambiguous genomic base) are now drawn with a distinct glyph (default “:”).
*/

// Max dots we’ll show between the primer sites (purely cosmetic).
const maxPrettyGap = 24

// Glyphs used in match bar rendering.
// - exactMatchGlyph: unambiguous A/C/G/T primer base equals genomic base
// - partialMatchGlyph: ambiguous primer code matches the genomic base (e.g., R vs A)
// - mismatch is drawn as space ' ' (kept)
const (
	exactMatchGlyph   = "|"
	partialMatchGlyph = "¦"
)
// intsCSV converts []int to "1,2,3" (empty string if none).
func intsCSV(a []int) string {
	if len(a) == 0 {
		return ""
	}
	var b strings.Builder
	for i, v := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(v))
	}
	return b.String()
}

// reverseString returns s reversed (ASCII-safe).
func reverseString(s string) string {
	rs := []rune(s)
	for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
		rs[i], rs[j] = rs[j], rs[i]
	}
	return string(rs)
}

// complementString returns the DNA complement for common IUPAC codes (uppercase).
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

// isACGT returns true if b ∈ {A,C,G,T}.
func isACGT(b byte) bool {
	return b == 'A' || b == 'C' || b == 'G' || b == 'T'
}

// matchLineAmbig draws a per‑column match bar string given primer/site (both 5'→3').
// - ' ' (space) for mismatches (indices present in mismIdx)
// - exactMatchGlyph for unambiguous exact matches (primer ∈ A/C/G/T and equals site)
// - partialMatchGlyph for ambiguous‑primer matches (primer ∉ A/C/G/T but column is a match)
func matchLineAmbig(primer, site string, mismIdx []int) string {
	n := len(primer)
	if n <= 0 {
		return ""
	}
	if len(site) < n {
		n = len(site) // defensive
	}

	// Build a set for O(1) mismatch lookup.
	mism := make(map[int]struct{}, len(mismIdx))
	for _, i := range mismIdx {
		mism[i] = struct{}{}
	}

	var b strings.Builder
	// Under‑reserve is fine; Builder will grow to accommodate multi‑byte glyphs.
	b.Grow(n)

	for i := 0; i < n; i++ {
		if _, bad := mism[i]; bad {
			b.WriteByte(' ')
			continue
		}
		p := primer[i]
		if isACGT(p) {
			// Unambiguous primer base matched exactly (since not listed as mismatch).
			b.WriteString(exactMatchGlyph)
		} else {
			// Ambiguous primer base matched one of its allowed genomic bases.
			b.WriteString(partialMatchGlyph)
		}
	}
	return b.String()
}

// diagramPretty prints a two‑strand block aligned to sites.
// Keeps the exact alignment you wanted, e.g.:
// 5'-AGAGTTTGATCCTGGCTCAG.......................-3' # (+)
// 3'-........................CCTATGGAACAATGCTGAA-5' # (-)
func diagramPretty(w io.Writer, p engine.Product) {
	if p.FwdPrimer == "" || p.RevPrimer == "" || p.FwdSite == "" || p.RevSite == "" {
		return
	}

	const (
		prefixPlus  = "5'-"
		suffixPlus  = "-3'"
		prefixMinus = "3'-"
		suffixMinus = "-5'"
		arrowRight  = "-->"
		arrowLeft   = "<--"
	)

	aLen, bLen := len(p.FwdPrimer), len(p.RevPrimer)

	// Interior spacing: cap for readability, otherwise use true interior.
	inner := maxPrettyGap
	if p.Length > 0 {
		interior := p.Length - aLen - bLen
		if interior < 0 {
			interior = 0
		}
		if interior < inner {
			inner = interior
		}
	}
	// One fewer dot on the (+) line to align the (−) site visually.
	innerPlus := inner
	if innerPlus > 0 {
		innerPlus--
	}
	innerMinus := inner

	// Forward primer and bars.
	fmt.Fprintf(w, "%s%s%s\n", prefixPlus, p.FwdPrimer, suffixPlus)
	fmt.Fprintf(
		w, "%s%s%s\n",
		strings.Repeat(" ", len(prefixPlus)),
		matchLineAmbig(p.FwdPrimer, p.FwdSite, p.FwdMismatchIdx),
		arrowRight,
	)

	// Genomic lines: (+) 5'→3', (−) 3'→5' under it, complementary and aligned.
	fmt.Fprintf(w, "%s%s%s%s # (+)\n", prefixPlus, p.FwdSite, strings.Repeat(".", innerPlus), suffixPlus)

	// RevSite is RC(plus slice at B) i.e., minus 5'→3'. To render the (−) line 3'→5' under (+),
	// we need the column-wise complement of the plus slice at B: complement(RevSite).
	minusSite := complementString(p.RevSite)
	fmt.Fprintf(w, "%s%s%s%s # (-)\n", prefixMinus, strings.Repeat(".", innerMinus), minusSite, suffixMinus)

	// Column where the first base of the (−) site appears.
	siteStart := len(prefixMinus) + innerMinus

	// Reverse primer bars (3' end under the first base of minusSite).
	revBars := reverseString(matchLineAmbig(p.RevPrimer, p.RevSite, p.RevMismatchIdx))
	padBars := siteStart - len(arrowLeft)
	if padBars < 0 {
		padBars = 0
	}
	fmt.Fprintf(w, "%s%s%s\n", strings.Repeat(" ", padBars), arrowLeft, revBars)

	// Reverse primer shown 3'→5' to match the (−) strand orientation.
	revPrimerDisplayed := reverseString(p.RevPrimer)
	padPrimer := siteStart - len(prefixMinus)
	if padPrimer < 0 {
		padPrimer = 0
	}
	fmt.Fprintf(w, "%s%s%s%s\n\n", strings.Repeat(" ", padPrimer), prefixMinus, revPrimerDisplayed, suffixMinus)
}

// writeOne prints a single product (TSV row + optional pretty block).
func writeOne(w io.Writer, p engine.Product, pretty bool) error {
	if _, err := fmt.Fprintf(
		w, "%s\t%s\t%d\t%d\t%d\t%s\t%d\t%d\t%s\t%s\n",
		p.SequenceID, p.ExperimentID, p.Start, p.End, p.Length, p.Type,
		p.FwdMM, p.RevMM, intsCSV(p.FwdMismatchIdx), intsCSV(p.RevMismatchIdx),
	); err != nil {
		return err
	}
	if pretty {
		diagramPretty(w, p)
	}
	return nil
}

// StreamText writes products from a channel (optional header; pretty mode).
func StreamText(w io.Writer, in <-chan engine.Product, header bool, pretty bool) error {
	if header {
		if _, err := fmt.Fprintln(w, TSVHeader); err != nil {
			return err
		}
		header = false
	}
	for p := range in {
		if err := writeOne(w, p, pretty); err != nil {
			return err
		}
	}
	return nil
}

// WriteText writes a slice of products (optional header; pretty mode).
func WriteText(w io.Writer, list []engine.Product, header bool, pretty bool) error {
	if header {
		if _, err := fmt.Fprintln(w, TSVHeader); err != nil {
			return err
		}
	}
	for _, p := range list {
		if err := writeOne(w, p, pretty); err != nil {
			return err
		}
	}
	return nil
}
