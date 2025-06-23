// internal/cli/options.go
package cli

import (
	"runtime"
	"errors"
	"flag"
	"fmt"
	"strings"
)

const (
	ModeRealistic = "realistic"
	ModeDebug     = "debug"
)

type Options struct {
	Mode        string
	PrimerFile  string
	Fwd         string
	Rev         string
	SeqFiles    []string
	Mismatches  int
	MinLen      int
	MaxLen      int
	Products    bool
	Output      string // text | json | fasta
	Pretty      bool
	FastaOut    string
	Threads 	int
}

// Parse reads command‑line flags from os.Args.
func Parse() (Options, error) {
	return ParseArgs(flag.CommandLine, nil)
}

func ParseArgs(fs *flag.FlagSet, argv []string) (Options, error) {
	var opt Options
	fs.StringVar(&opt.PrimerFile, "primers", "", "TSV primer file")
	fs.StringVar(&opt.Fwd, "forward", "", "Forward primer (5'->3')")
	fs.StringVar(&opt.Rev, "reverse", "", "Reverse primer (5'->3')")
	var seq stringSlice
	fs.Var(&seq, "sequences", "FASTA file(s) (repeatable)")
	fs.IntVar(&opt.Mismatches, "mismatches", 0, "max mismatches")
	fs.IntVar(&opt.MinLen, "min-length", 0, "minimum product length")
	fs.IntVar(&opt.MaxLen, "max-length", 0, "maximum product length")
	fs.StringVar(&opt.Mode, "mode", ModeRealistic, "realistic|debug")
	fs.BoolVar(&opt.Products, "products", false, "emit product sequences")
	fs.StringVar(&opt.Output, "output", "text", "text|json|fasta")
	fs.BoolVar(&opt.Pretty, "pretty", false, "human pretty output")
	fs.StringVar(&opt.FastaOut, "fasta-out", "", "write products to FASTA file")
	fs.IntVar(&opt.Threads, "threads", runtime.NumCPU(), "number of worker threads")

	// Suppress default usage in tests; callers can re‑enable.
	fs.Usage = func() {}

	if err := fs.Parse(argv); err != nil {
		return opt, err
	}
	opt.SeqFiles = seq

	// ---------- validation ----------

	usingFile := opt.PrimerFile != ""
	usingInline := opt.Fwd != "" || opt.Rev != ""
	
	if opt.Threads <= 0 {
		return opt, errors.New("--threads must be > 0")
	}

	// Mutually exclusive
	switch {
	case usingFile && usingInline:
		return opt, errors.New("--primers cannot be combined with --forward/--reverse")
	case usingInline && (opt.Fwd == "" || opt.Rev == ""):
		return opt, errors.New("--forward and --reverse must both be supplied")
	case !usingFile && !usingInline:
		return opt, errors.New("you must provide either --primers or --forward/--reverse")
	}

	if len(opt.SeqFiles) == 0 {
		return opt, errors.New("at least one --sequences FASTA is required")
	}

	if opt.Mode != ModeRealistic && opt.Mode != ModeDebug {
		return opt, fmt.Errorf("invalid --mode %q", opt.Mode)
	}
	if opt.Output != "text" && opt.Output != "json" && opt.Output != "fasta" {
		return opt, fmt.Errorf("invalid --output %q", opt.Output)
	}
	return opt, nil
}

// ---------- helper types ----------

type stringSlice []string

func (s *stringSlice) String() string   { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }
