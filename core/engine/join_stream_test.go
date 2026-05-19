package engine

import (
	"errors"
	"ipcr-core/primer"
	"testing"
)

var errForEachJoinedProductTest = errors.New("stop joined product stream")

func testJoinInputs(seq []byte, pair primer.Pair, cfg Config) ([]primer.Match, []primer.Match, []primer.Match, []primer.Match) {
	tw := cfg.TerminalWindow
	fwdA := primer.FindMatches(seq, []byte(pair.Forward), cfg.MaxMM, cfg.HitCap, tw)
	fwdB := primer.FindMatches(seq, []byte(pair.Reverse), cfg.MaxMM, cfg.HitCap, tw)
	revA := filterLeftTW(primer.FindMatches(seq, primer.RevComp([]byte(pair.Forward)), cfg.MaxMM, cfg.HitCap, 0), tw)
	revB := filterLeftTW(primer.FindMatches(seq, primer.RevComp([]byte(pair.Reverse)), cfg.MaxMM, cfg.HitCap, 0), tw)
	return fwdA, fwdB, revA, revB
}

func TestForEachJoinedProductMatchesJoinProducts(t *testing.T) {
	seq := []byte("TTTACGTACAAAAGGTACCTTTGGGACGTATAAAAGGTACCAAA")
	pair := primer.Pair{
		ID:         "join_stream",
		Forward:    "ACGTAC",
		Reverse:    "GGTACC",
		MinProduct: 6,
		MaxProduct: 80,
	}
	cfg := Config{MaxMM: 1, TerminalWindow: 0, MinLen: 6, MaxLen: 80, NeedSites: true}
	eng := New(cfg)
	fwdA, fwdB, revA, revB := testJoinInputs(seq, pair, cfg)

	want := eng.joinProducts("seq", seq, pair, fwdA, fwdB, revA, revB)
	var got []Product
	err := eng.forEachJoinedProduct("seq", seq, pair, fwdA, fwdB, revA, revB, func(product Product) error {
		got = append(got, product)
		return nil
	})
	if err != nil {
		t.Fatalf("forEachJoinedProduct error: %v", err)
	}
	assertProductMultisetEqual(t, got, want)
}

func TestForEachJoinedProductPropagatesEmitError(t *testing.T) {
	seq := []byte("TTTACGTACAAAAGGTACCTTT")
	pair := primer.Pair{ID: "join_error", Forward: "ACGTAC", Reverse: "GGTACC", MinProduct: 6, MaxProduct: 60}
	cfg := Config{MaxMM: 0, TerminalWindow: 0, MinLen: 6, MaxLen: 60}
	eng := New(cfg)
	fwdA, fwdB, revA, revB := testJoinInputs(seq, pair, cfg)

	err := eng.forEachJoinedProduct("seq", seq, pair, fwdA, fwdB, revA, revB, func(Product) error {
		return errForEachJoinedProductTest
	})
	if !errors.Is(err, errForEachJoinedProductTest) {
		t.Fatalf("forEachJoinedProduct error = %v, want %v", err, errForEachJoinedProductTest)
	}
}
