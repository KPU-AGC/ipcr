package probecli

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func newFS() *flag.FlagSet { return flag.NewFlagSet("test", flag.ContinueOnError) }

func mustParse(t *testing.T, args ...string) Options {
	t.Helper()
	o, err := ParseArgs(newFS(), args)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return o
}

func TestProbeFlagsOK(t *testing.T) {
	o := mustParse(t,
		"--forward", "AAA", "--reverse", "TTT",
		"--probe", "ACG", "ref.fa",
	)
	if o.Probe != "ACG" || len(o.SeqFiles) != 1 || o.SeqFiles[0] != "ref.fa" {
		t.Fatalf("bad parse: %+v", o)
	}
}

func TestShortProbeAlias_P(t *testing.T) {
	o := mustParse(t,
		"--forward", "AAA", "--reverse", "TTT",
		"-P", "NNNTTT",
		"ref.fa",
	)
	if o.Probe != "NNNTTT" {
		t.Fatalf("want probe=NNNTTT via -P, got %q", o.Probe)
	}
}

func TestShortProbeMaxMMAlias_M(t *testing.T) {
	o := mustParse(t,
		"--forward", "AAA", "--reverse", "TTT",
		"--probe", "ACG",
		"-M", "2",
		"ref.fa",
	)
	if o.ProbeMaxMM != 2 {
		t.Fatalf("want ProbeMaxMM=2, got %d", o.ProbeMaxMM)
	}
}

func TestRequireProbeMissingErrors(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{
		"--forward", "AAA", "--reverse", "TTT", "ref.fa",
	})
	if err == nil {
		t.Fatal("expected error when --probe missing")
	}
}

func TestPositionalGlobOK(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.fa")
	b := filepath.Join(dir, "b.fa")
	_ = os.WriteFile(a, []byte(">a\nA\n"), 0o644)
	_ = os.WriteFile(b, []byte(">b\nA\n"), 0o644)
	pat := filepath.Join(dir, "*.fa")

	o := mustParse(t, "--forward", "AAA", "--reverse", "TTT", "--probe", "ACG", pat)
	if len(o.SeqFiles) != 2 {
		t.Fatalf("want 2 seqs, got %d", len(o.SeqFiles))
	}
}

func TestMutualExclusion(t *testing.T) {
	_, err := ParseArgs(newFS(), []string{
		"--primers", "p.tsv", "--forward", "AAA", "--reverse", "TTT", "--probe", "ACG", "ref.fa",
	})
	if err == nil {
		t.Fatal("expected mutual exclusion error")
	}
}
