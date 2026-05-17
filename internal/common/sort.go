package common

import (
	"ipcr-core/engine"
	"math"
	"sort"
	"strconv"
	"strings"
)

// productSortCoords returns record-global coordinates. Chunked records are named
// like "record_id:start-end"; their Product.Start/End values are chunk-local.
func productSortCoords(p engine.Product) (base string, start, end int) {
	base, off, ok := SplitChunkSuffix(p.SequenceID)
	if !ok {
		base = p.SequenceID
		off = 0
	}
	return base, p.Start + off, p.End + off
}

func intsKey(a []int) string {
	if len(a) == 0 {
		return ""
	}
	parts := make([]string, len(a))
	for i, v := range a {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

// LessProduct defines a total, reproducible order for products.
func LessProduct(a, b engine.Product) bool {
	if a.SourceFile != b.SourceFile {
		return a.SourceFile < b.SourceFile
	}
	aBase, aStart, aEnd := productSortCoords(a)
	bBase, bStart, bEnd := productSortCoords(b)
	if aBase != bBase {
		return aBase < bBase
	}
	if aStart != bStart {
		return aStart < bStart
	}
	if aEnd != bEnd {
		return aEnd < bEnd
	}
	if a.Length != b.Length {
		return a.Length < b.Length
	}
	if a.Type != b.Type {
		return a.Type < b.Type
	}
	if a.ExperimentID != b.ExperimentID {
		return a.ExperimentID < b.ExperimentID
	}
	if a.FwdMM != b.FwdMM {
		return a.FwdMM < b.FwdMM
	}
	if a.RevMM != b.RevMM {
		return a.RevMM < b.RevMM
	}
	if ai, bi := intsKey(a.FwdMismatchIdx), intsKey(b.FwdMismatchIdx); ai != bi {
		return ai < bi
	}
	if ai, bi := intsKey(a.RevMismatchIdx), intsKey(b.RevMismatchIdx); ai != bi {
		return ai < bi
	}
	if a.SequenceID != b.SequenceID {
		return a.SequenceID < b.SequenceID
	}
	return a.Seq < b.Seq
}

func SortProducts(ps []engine.Product) {
	sort.SliceStable(ps, func(i, j int) bool { return LessProduct(ps[i], ps[j]) })
}

// score-priority sort (descending), then fall back to full coordinate order.
func LessProductByScore(a, b engine.Product) bool {
	aNaN, bNaN := math.IsNaN(a.Score), math.IsNaN(b.Score)
	switch {
	case aNaN && bNaN:
		return LessProduct(a, b)
	case aNaN:
		return false // NaN last
	case bNaN:
		return true
	case a.Score != b.Score:
		return a.Score > b.Score // higher first
	default:
		return LessProduct(a, b)
	}
}

func SortProductsByScore(ps []engine.Product) {
	sort.SliceStable(ps, func(i, j int) bool { return LessProductByScore(ps[i], ps[j]) })
}
