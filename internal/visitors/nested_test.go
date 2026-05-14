package visitors

import (
	"ipcr-core/engine"
	"ipcr-core/primer"
	"testing"
)

func TestNestedVisitChoosesBestAfterSorting(t *testing.T) {
	outer := engine.Product{
		ExperimentID: "outer",
		SequenceID:   "amp",
		Seq:          "AAAACACACACGGGACACACTTTACCCC",
	}

	v := Nested{
		InnerPairs: []primer.Pair{
			{ID: "short-late", Forward: "TTT", Reverse: "GGG"},
			{ID: "long-early", Forward: "AAA", Reverse: "CCC"},
		},
		EngineCfg: engine.Config{MaxMM: 0, TerminalWindow: 0},
	}

	keep, got, err := v.Visit(outer)
	if err != nil {
		t.Fatalf("Visit returned error: %v", err)
	}
	if !keep || !got.InnerFound {
		t.Fatalf("expected nested hit, keep=%v got=%+v", keep, got)
	}
	if got.InnerPairID != "long-early" {
		t.Fatalf("expected best/longest inner pair long-early, got %+v", got)
	}
	if got.InnerStart != 0 || got.InnerLength != 14 {
		t.Fatalf("unexpected selected inner coordinates: %+v", got)
	}
}
