// internal/cli/options.go
package cli

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"ipcr/internal/version"
)

// Command-line modes
const (
	ModeRealistic = "realistic"
	ModeDebug     = "debug"
)

// Options holds all CLI flags and arguments.
type Options struct {
	// File / primer input
	PrimerFile string
	Fwd        string
	Rev        string
	SeqFiles   []string

	// PCR parameters
	Mismatches     int
	MinLen         int
	MaxLen         int
	HitCap         int
	TerminalWindow int // -1=auto by --mode; 0=allow; N>0=no-mismatch 3' window

	// Performance
	Threads   int
	ChunkSize int

	// Output
	Output   string
	Products bool
	Pretty   bool
	Mode     string
	Sort     bool
	Header   bool // true unless --no-header

	Version bool
}

// NewFlagSet returns a configured FlagSet with custom usage/help.
func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(),
			`%s: in-silico PCR

Author:  Erick Samera (erick.samera@kpu.ca)
License: MIT
Version: %s

Usage of %s:
`, name, version.Version, name)
		fs.PrintDefaults()
	}
	return fs
}

// Parse is the top-level call for CLI parsing.
func Parse() (Options, error) { return ParseArgs(flag.CommandLine, nil) }

// ParseArgs registers and parses all flags, returns an Options struct.
func ParseArgs(fs *flag.FlagSet, argv []string) (Options, error) {
	var opt Options
	var help bool

	// File & primer input
	fs.StringVar(&opt.PrimerFile, "primers", "", "TSV primer file [*]")
	fs.StringVar(&opt.Fwd, "forward", "", "forward primer (5'→3') [*]")
	fs.StringVar(&opt.Rev, "reverse", "", "reverse primer (5'→3') [*]")
	var seq stringSlice
	fs.Var(&seq, "sequences", "FASTA file(s) (repeatable or '-') [*]")

	// PCR parameters
	fs.IntVar(&opt.Mismatches, "mismatches", 0, "max mismatches per primer [0]")
	fs.IntVar(&opt.MinLen, "min-length", 0, "minimum product length [0]")
	fs.IntVar(&opt.MaxLen, "max-length", 0, "maximum product length [0]")
	fs.IntVar(&opt.HitCap, "hit-cap", 10000, "max matches stored per primer per window (0 = unlimited) [10000]")
	fs.IntVar(&opt.TerminalWindow, "terminal-window", -1, "3' terminal window (nt) disallowed for mismatches (0=allow, -1=auto: realistic=3, debug=0) [-1]")

	// Performance
	// Default now 0 to match help text ("0 = all CPUs")
	fs.IntVar(&opt.Threads, "threads", 0, "number of worker threads (0 = all CPUs) [0]")
	fs.IntVar(&opt.ChunkSize, "chunk-size", 0, "split sequences into N-bp windows (0 = no chunking) [0]")

	// Output
	fs.StringVar(&opt.Output, "output", "text", "output format: text | json | fasta [text]")
	fs.StringVar(&opt.Mode, "mode", ModeRealistic, "matching mode: realistic | debug ["+ModeRealistic+"]")
	fs.BoolVar(&opt.Products, "products", false, "emit product sequences [false]")
	fs.BoolVar(&opt.Pretty, "pretty", false, "pretty ASCII alignment block (text) [false]")
	fs.BoolVar(&opt.Sort, "sort", false, "sort outputs for determinism (SequenceID,Start,End,Type,ExperimentID) [false]")
	noHeader := false
	fs.BoolVar(&noHeader, "no-header", false, "suppress header line in text/TSV [false]")

	fs.BoolVar(&opt.Version, "v", false, "print version and exit (shorthand) [false]")
	fs.BoolVar(&opt.Version, "version", false, "print version and exit [false]")
	fs.BoolVar(&help, "h", false, "show this help message (shorthand) [false]")

	if err := fs.Parse(argv); err != nil {
		return opt, err
	}
	if help {
		fs.Usage()
		return opt, flag.ErrHelp
	}
	if opt.Version {
		return opt, nil
	}
	opt.SeqFiles = seq
	opt.Header = !noHeader

	// Validation
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
	if opt.TerminalWindow < -1 {
		return opt, errors.New("--terminal-window must be ≥ -1")
	}
	return opt, nil
}

// stringSlice allows repeatable string flags.
type stringSlice []string

func (s *stringSlice) String() string     { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }
