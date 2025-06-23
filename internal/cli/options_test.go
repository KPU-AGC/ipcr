// internal/cli/options_test.go
package cli

import (
	"flag"
	"testing"
)

func newFS() *flag.FlagSet { return flag.NewFlagSet("test", flag.ContinueOnError) }

func mustParse(t *testing.T, args ...string) Options {
	t.Helper()
	opts, err := ParseArgs(newFS(), args)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	return opts
}

func TestPrimersFileOK(t *testing.T) {
	o := mustParse(t,
		"--primers", "p.tsv",
		"--sequences", "ref.fa",
	)
	if o.PrimerFile != "p.tsv" || o.Fwd != "" {
		t.Errorf("want primers file only, got %+v", o)
	}
}

func TestInlinePrimersOK(t *testing.T) {
	o := mustParse(t,
		"--forward", "AAA",
		"--reverse", "TTT",
		"--sequences", "ref.fa", "--sequences", "extra.fa",
	)
	if o.Fwd != "AAA" || len(o.SeqFiles) != 2 {
		t.Errorf("bad inline parse %+v", o)
	}
}

func TestErrorMissingReverse(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{
		"--forward", "AAA", "--sequences", "ref.fa",
	})
	if err == nil {
		t.Fatalf("expected error when reverse not supplied")
	}
}

func TestErrorMutualExclusion(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{
		"--primers", "p.tsv", "--forward", "AAA",
		"--reverse", "TTT", "--sequences", "ref.fa",
	})
	if err == nil {
		t.Fatalf("expected mutual‑exclusion error")
	}
}

func TestErrorNoPrimerInput(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{
		"--sequences", "ref.fa",
	})
	if err == nil {
		t.Fatalf("expected error with no primers")
	}
}

func TestErrorNoSequences(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{
		"--forward", "AAA", "--reverse", "TTT",
	})
	if err == nil {
		t.Fatalf("expected error when sequences missing")
	}
}
