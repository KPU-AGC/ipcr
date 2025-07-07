// internal/cli/options_test.go
package cli

import (
	"flag"
	"testing"
)

func newFS() *flag.FlagSet {
	return flag.NewFlagSet("test", flag.ContinueOnError)
}

// Helper for tests expecting successful parse.
func mustParse(t *testing.T, args ...string) Options {
	t.Helper()
	opts, err := ParseArgs(newFS(), args)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	return opts
}

// Should parse primer file input
func TestPrimersFileOK(t *testing.T) {
	o := mustParse(t, "--primers", "p.tsv", "--sequences", "ref.fa")
	if o.PrimerFile != "p.tsv" || o.Fwd != "" {
		t.Errorf("expected primers file only, got %+v", o)
	}
}

// Should parse inline primer input with multiple sequence files
func TestInlinePrimersOK(t *testing.T) {
	o := mustParse(t,
		"--forward", "AAA",
		"--reverse", "TTT",
		"--sequences", "ref.fa", "--sequences", "extra.fa",
	)
	if o.Fwd != "AAA" || len(o.SeqFiles) != 2 {
		t.Errorf("bad inline parse: %+v", o)
	}
}

// Should fail when reverse is missing
func TestErrorMissingReverse(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{
		"--forward", "AAA", "--sequences", "ref.fa",
	})
	if err == nil {
		t.Fatal("expected error when reverse not supplied")
	}
}

// Should fail for mutual exclusion of primers and forward/reverse
func TestErrorMutualExclusion(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{
		"--primers", "p.tsv", "--forward", "AAA",
		"--reverse", "TTT", "--sequences", "ref.fa",
	})
	if err == nil {
		t.Fatal("expected mutual-exclusion error")
	}
}

// Should fail when no primer input is given
func TestErrorNoPrimerInput(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{"--sequences", "ref.fa"})
	if err == nil {
		t.Fatal("expected error with no primers")
	}
}

// Should fail when sequences are missing
func TestErrorNoSequences(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{"--forward", "AAA", "--reverse", "TTT"})
	if err == nil {
		t.Fatal("expected error when sequences missing")
	}
}
// ===