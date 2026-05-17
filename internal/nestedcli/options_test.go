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
