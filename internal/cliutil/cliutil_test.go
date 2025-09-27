package cliutil

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestSplitFlagsAndPositionals(t *testing.T) {
	fs := flag.NewFlagSet("x", flag.ContinueOnError)
	var b bool
	fs.BoolVar(&b, "bool", false, "")
	flagArgs, posArgs := SplitFlagsAndPositionals(fs, []string{"--bool", "pos1", "--", "pos2"})
	if len(flagArgs) != 1 || len(posArgs) != 2 || posArgs[0] != "pos1" || posArgs[1] != "pos2" {
		t.Fatalf("unexpected split: %v / %v", flagArgs, posArgs)
	}
}

func TestExpandPositionals(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.fa")
	b := filepath.Join(dir, "b.fa")
	_ = os.WriteFile(a, []byte(">a\nA\n"), 0o644)
	_ = os.WriteFile(b, []byte(">b\nA\n"), 0o644)
	got, err := ExpandPositionals([]string{filepath.Join(dir, "*.fa")})
	if err != nil || len(got) != 2 {
		t.Fatalf("expand: err=%v got=%v", err, got)
	}
}
