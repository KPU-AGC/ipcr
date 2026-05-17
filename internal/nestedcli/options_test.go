package nestedcli

import (
	"flag"
	"testing"
)

func newFS() *flag.FlagSet { return NewFlagSet("test") }

func TestOuterPrimersAlias(t *testing.T) {
	o, err := ParseArgs(newFS(), []string{
		"--outer-primers", "outer.tsv",
		"--inner-primers", "inner.tsv",
		"ref.fa",
	})
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if o.PrimerFile != "outer.tsv" || o.InnerPrimerFile != "inner.tsv" {
		t.Fatalf("bad primer files: %+v", o)
	}
}

func TestInnerInlinePrimersNormalizeAndValidate(t *testing.T) {
	o, err := ParseArgs(newFS(), []string{
		"--forward", "aaa",
		"--reverse", "ttt",
		"--inner-forward", " acgtry ",
		"--inner-reverse", "ggg",
		"ref.fa",
	})
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if o.Fwd != "AAA" || o.Rev != "TTT" || o.InnerFwd != "ACGTRY" || o.InnerRev != "GGG" {
		t.Fatalf("expected normalized primers, got %+v", o)
	}
}

func TestInnerInlinePrimersRejectInvalidBase(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{
		"--forward", "AAA",
		"--reverse", "TTT",
		"--inner-forward", "ACGX",
		"--inner-reverse", "GGG",
		"ref.fa",
	})
	if err == nil {
		t.Fatal("expected invalid inner primer error")
	}
}
