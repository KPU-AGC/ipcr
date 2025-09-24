// internal/cli/options_test.go
package cli

import (
	"flag"
	"os"
	"path/filepath"
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
	o := mustParse(t, "--primers", "p.tsv", "ref.fa")
	if o.PrimerFile != "p.tsv" || o.Fwd != "" {
		t.Errorf("expected primers file only, got %+v", o)
	}
	if len(o.SeqFiles) != 1 || o.SeqFiles[0] != "ref.fa" {
		t.Errorf("expected positional sequence, got %+v", o.SeqFiles)
	}
}

// Should parse inline primer input with multiple sequence files (legacy flag)
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
		"--forward", "AAA", "ref.fa",
	})
	if err == nil {
		t.Fatal("expected error when reverse not supplied")
	}
}

// Should fail for mutual exclusion of primers and forward/reverse
func TestErrorMutualExclusion(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{
		"--primers", "p.tsv", "--forward", "AAA",
		"--reverse", "TTT", "ref.fa",
	})
	if err == nil {
		t.Fatal("expected mutual-exclusion error")
	}
}

// Should fail when no primer input is given
func TestErrorNoPrimerInput(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{"ref.fa"})
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

func TestNewFlags(t *testing.T) {
	o := mustParse(t, "--forward", "AAA", "--reverse", "TTT", "ref.fa", "--sort", "--no-header", "--terminal-window", "2")
	if !o.Sort {
		t.Errorf("expected --sort = true")
	}
	if o.Header {
		t.Errorf("expected header=false due to --no-header")
	}
	if o.TerminalWindow != 2 {
		t.Errorf("terminal window parse failed, got %d", o.TerminalWindow)
	}
}

// Positional globs should expand to matching files
func TestPositionalGlobOK(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.fa")
	b := filepath.Join(dir, "b.fa")
	if err := os.WriteFile(a, []byte(">a\nA\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(">b\nA\n"), 0644); err != nil {
		t.Fatal(err)
	}
	pattern := filepath.Join(dir, "*.fa")

	o := mustParse(t, "--forward", "AAA", "--reverse", "TTT", pattern)
	if len(o.SeqFiles) != 2 {
		t.Fatalf("expected 2 files from glob, got %d: %+v", len(o.SeqFiles), o.SeqFiles)
	}
	// Order is determined by filepath.Glob; just assert both are present.
	foundA, foundB := false, false
	for _, f := range o.SeqFiles {
		if f == a {
			foundA = true
		} else if f == b {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Fatalf("glob expansion missing files: %+v", o.SeqFiles)
	}
}

// Mix legacy --sequences and positional inputs
func TestMixFlagAndPositional(t *testing.T) {
	dir := t.TempDir()
	x := filepath.Join(dir, "x.fa")
	y := filepath.Join(dir, "y.fa")
	_ = os.WriteFile(x, []byte(">x\nA\n"), 0644)
	_ = os.WriteFile(y, []byte(">y\nA\n"), 0644)

	o := mustParse(t,
		"--forward", "AAA", "--reverse", "TTT",
		"--sequences", x, // legacy
		y,                // positional
	)
	if len(o.SeqFiles) != 2 {
		t.Fatalf("expected 2 seq files, got %d: %+v", len(o.SeqFiles), o.SeqFiles)
	}
}

// Unmatched glob should error
func TestGlobNoMatchErrors(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{
		"--forward", "AAA", "--reverse", "TTT",
		filepath.Join(t.TempDir(), "*.nope"),
	})
	if err == nil {
		t.Fatal("expected error on unmatched glob")
	}
}
