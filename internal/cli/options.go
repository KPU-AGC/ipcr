package cli

import (
	"errors"
	"flag"
	"fmt"
	"runtime"
	"strings"
)

/* ------------------------------ public API ------------------------------ */

const (
	ModeRealistic = "realistic"
	ModeDebug     = "debug"
)

type Options struct {
	/* file / primer input */
	PrimerFile string
	Fwd        string
	Rev        string
	SeqFiles   []string

	/* PCR parameters */
	Mismatches int
	MinLen     int
	MaxLen     int
	HitCap     int // NEW – maximum hits kept per primer per window

	/* performance */
	Threads   int
	ChunkSize int

	/* output */
	Output  string
	Products bool
	Pretty   bool
	Mode     string
}

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {}
	return fs
}

func Parse() (Options, error) { return ParseArgs(flag.CommandLine, nil) }

/* ------------------------------- ParseArgs ------------------------------ */

func ParseArgs(fs *flag.FlagSet, argv []string) (Options, error) {
	var opt Options

	/* files & primers */
	fs.StringVar(&opt.PrimerFile, "primers", "", "TSV primer file")
	fs.StringVar(&opt.Fwd, "forward", "", "forward primer (5'→3')")
	fs.StringVar(&opt.Rev, "reverse", "", "reverse primer (5'→3')")
	var seq stringSlice
	fs.Var(&seq, "sequences", "FASTA file(s) (repeatable or '-')")

	/* PCR parameters */
	fs.IntVar(&opt.Mismatches, "mismatches", 0, "max mismatches per primer")
	fs.IntVar(&opt.MinLen, "min-length", 0, "minimum product length")
	fs.IntVar(&opt.MaxLen, "max-length", 0, "maximum product length")
	fs.IntVar(&opt.HitCap, "hit-cap", 10000,
		"max matches stored per primer per window (0 = unlimited)")

	/* perf */
	fs.IntVar(&opt.Threads, "threads", runtime.NumCPU(),
		"worker threads (0 = all CPUs)")
	fs.IntVar(&opt.ChunkSize, "chunk-size", 0,
		"split sequences into N‑bp windows (0 = no chunking)")

	/* misc */
	fs.StringVar(&opt.Output, "output", "text", "text | json | fasta")
	fs.StringVar(&opt.Mode, "mode", ModeRealistic, "realistic | debug")
	fs.BoolVar(&opt.Products, "products", false, "emit product sequences")
	fs.BoolVar(&opt.Pretty, "pretty", false, "pretty ASCII alignment (text)")

	if err := fs.Parse(argv); err != nil {
		return opt, err
	}
	opt.SeqFiles = seq

	/* validation */
	usingFile := opt.PrimerFile != ""
	usingInline := opt.Fwd != "" || opt.Rev != ""
	switch {
	case usingFile && usingInline:
		return opt, errors.New("--primers conflicts with --forward/--reverse")
	case usingInline && (opt.Fwd == "" || opt.Rev == ""):
		return opt, errors.New("--forward and --reverse must be supplied together")
	case !usingFile && !usingInline:
		return opt, errors.New("provide --primers or --forward/--reverse")
	}
	if len(opt.SeqFiles) == 0 {
		return opt, errors.New("at least one --sequences file is required")
	}
	if opt.Threads < 0 {
		return opt, errors.New("--threads must be ≥ 0")
	}
	if opt.ChunkSize < 0 {
		return opt, errors.New("--chunk-size must be ≥ 0")
	}
	if opt.HitCap < 0 {
		return opt, errors.New("--hit-cap must be ≥ 0")
	}
	if opt.Output != "text" && opt.Output != "json" && opt.Output != "fasta" {
		return opt, fmt.Errorf("invalid --output %q", opt.Output)
	}
	return opt, nil
}

/* ----------------------------- helper types ----------------------------- */

type stringSlice []string

func (s *stringSlice) String() string   { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }
