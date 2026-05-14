package appcore

import (
	"ipcr-core/primer"
	"testing"
)

func TestEffectiveMaxProductLenUsesPerPairOverrides(t *testing.T) {
	pairs := []primer.Pair{
		{ID: "global", Forward: "AAA", Reverse: "TTT"},
		{ID: "wide", Forward: "AAA", Reverse: "TTT", MaxProduct: 5000},
	}
	if got := effectiveMaxProductLen(2000, pairs); got != 5000 {
		t.Fatalf("expected per-pair max to widen effective chunk overlap, got %d", got)
	}
}

func TestEffectiveMaxProductLenDetectsUnboundedPair(t *testing.T) {
	pairs := []primer.Pair{
		{ID: "bounded", Forward: "AAA", Reverse: "TTT", MaxProduct: 5000},
		{ID: "unbounded", Forward: "CCC", Reverse: "GGG"},
	}
	if got := effectiveMaxProductLen(0, pairs); got != 0 {
		t.Fatalf("expected unbounded effective max when global max is 0 and a pair has no override, got %d", got)
	}
}

func TestEffectiveMaxProductLenAllowsAllBoundedPairsWithoutGlobalMax(t *testing.T) {
	pairs := []primer.Pair{
		{ID: "a", Forward: "AAA", Reverse: "TTT", MaxProduct: 1000},
		{ID: "b", Forward: "CCC", Reverse: "GGG", MaxProduct: 3000},
	}
	if got := effectiveMaxProductLen(0, pairs); got != 3000 {
		t.Fatalf("expected max bounded pair length, got %d", got)
	}
}
