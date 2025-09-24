// internal/cli/options.go
package cli

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
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
	Threads    int
	ChunkSize  int
	SeedLength int // seed length for multi-pattern scan (0=auto)
	Circular   bool

	// Output
	Output          string
	Products        bool
	Pretty          bool
	Mode            string
	Sort            bool
	Header          bool
	NoMatchExitCode int

	// Misc
	Quiet   bool
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

  # With flags (legacy; literal paths, no globbing validated here)
  %s --forward AAA --reverse TTT --sequences ref.fa

  # With positionals (new; globs are expanded; literals are accepted as-is)
  # Flags may appear before or after positionals.
  %s --forward AAA --reverse TTT ref*.fa gz/*.fa.gz

`, name, version.Version, name, name, name)
		fs.PrintDefaults()
	}
	return fs
}

// Parse is the top-level call for CLI parsing.
func Parse() (Options, error) { return ParseArgs(flag.CommandLine, nil) }

// stringSlice allows repeatable string flags.
type stringSlice []string

func (s *stringSlice) String() string     { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

// hasGlobMeta reports whether s contains glob metacharacters.
func hasGlobMeta(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// splitFlagsAndPositionals separates argv into (1) tokens intended for the flag
// parser and (2) positional tokens. It supports interspersed flags. For flags
// that require a value, it also attaches the following token as the value
// (even if it starts with '-' to allow negative numbers).
func splitFlagsAndPositionals(fs *flag.FlagSet, argv []string) (flagArgs []string, posArgs []string) {
	// Discover which flags are boolean (IsBoolFlag returns true for bools).
	boolFlags := make(map[string]bool)
	fs.VisitAll(func(f *flag.Flag) {
		if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
			boolFlags[f.Name] = true
		}
	})

	for i := 0; i < len(argv); i++ {
		arg := argv[i]

		if arg == "--" {
			// Everything after "--" is positional.
			posArgs = append(posArgs, argv[i+1:]...)
			break
		}
		if arg == "-" {
			posArgs = append(posArgs, arg)
			continue
		}
		if strings.HasPrefix(arg, "-") {
			// It's a flag-like token.
			if strings.Contains(arg, "=") {
				flagArgs = append(flagArgs, arg)
				continue
			}
			// Extract the flag name (strip one or two leading dashes).
			name := strings.TrimLeft(arg, "-")
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				name = name[:eq]
			}
			needsValue := true
			if boolFlags[name] {
				needsValue = false // booleans don't require a value token
			}
			flagArgs = append(flagArgs, arg)
			if needsValue && i+1 < len(argv) {
				// Attach the next token as the value (handles negatives like "-1").
				flagArgs = append(flagArgs, argv[i+1])
				i++
			}
			continue
		}

		// Positional token.
		posArgs = append(posArgs, arg)
	}

	return flagArgs, posArgs
}

// ParseArgs registers and parses all flags, returns an Options struct.
func ParseArgs(fs *flag.FlagSet, argv []string) (Options, error) {
	var opt Options
	var help bool

	// File & primer input
	fs.StringVar(&opt.PrimerFile, "primers", "", "TSV primer file [*]")
	fs.StringVar(&opt.Fwd, "forward", "", "forward primer (5'→3') [*]")
	fs.StringVar(&opt.Rev, "reverse", "", "reverse primer (5'→3') [*]")
	var seq stringSlice
	fs.Var(&seq, "sequences", "FASTA file(s) (repeatable or '-') [*] (literal; no glob expansion here)")

	// PCR parameters
	fs.IntVar(&opt.Mismatches, "mismatches", 0, "max mismatches per primer [0]")
	fs.IntVar(&opt.MinLen, "min-length", 0, "minimum product length [0]")
	fs.IntVar(&opt.MaxLen, "max-length", 2000, "maximum product length [2000]")
	fs.IntVar(&opt.HitCap, "hit-cap", 10000, "max matches stored per primer per window (0 = unlimited) [10000]")
	fs.IntVar(&opt.TerminalWindow, "terminal-window", -1, "3' terminal window (nt) disallowed for mismatches (0=allow, -1=auto: realistic=3, debug=0) [-1]")

	// Performance
	fs.IntVar(&opt.Threads, "threads", 0, "number of worker threads (0 = all CPUs) [0]")
	fs.IntVar(&opt.ChunkSize, "chunk-size", 0, "split sequences into N-bp windows (0 = no chunking) [0]")
	fs.IntVar(&opt.SeedLength, "seed-length", 12, "seed length for multi-pattern scan (0=auto: min(12, primer length)) [12]")

	// Output / misc behavior
	fs.StringVar(&opt.Output, "output", "text", "output format: text | json | fasta [text]")
	fs.StringVar(&opt.Mode, "mode", ModeRealistic, "matching mode: realistic | debug ["+ModeRealistic+"]")
	fs.BoolVar(&opt.Circular, "circular", false, "treat each FASTA record as circular (disables chunking) [false]")
	fs.BoolVar(&opt.Products, "products", false, "emit product sequences [false]")
	fs.BoolVar(&opt.Pretty, "pretty", false, "pretty ASCII alignment block (text) [false]")
	fs.BoolVar(&opt.Sort, "sort", false, "sort outputs for determinism (SequenceID,Start,End,Type,ExperimentID) [false]")
	noHeader := false
	fs.BoolVar(&noHeader, "no-header", false, "suppress header line in text/TSV [false]")
	fs.IntVar(&opt.NoMatchExitCode, "no-match-exit-code", 1, "exit code to use when no amplicons are found (set 0 to treat as success) [1]")

	// Misc
	fs.BoolVar(&opt.Quiet, "quiet", false, "suppress non-essential warnings on stderr [false]")
	fs.BoolVar(&opt.Version, "v", false, "print version and exit (shorthand) [false]")
	fs.BoolVar(&opt.Version, "version", false, "print version and exit [false]")
	fs.BoolVar(&help, "h", false, "show this help message (shorthand) [false]")

	// Support interspersed flags: split argv into flag tokens + positionals.
	flagArgs, posArgs := splitFlagsAndPositionals(fs, argv)
	if err := fs.Parse(flagArgs); err != nil {
		return opt, err
	}
	if help {
		// Caller decides where to print Usage; we only signal intent.
		return opt, flag.ErrHelp
	}
	if opt.Version {
		return opt, nil
	}

	// Flags: keep literal order & values (no globbing here to preserve legacy behavior).
	opt.SeqFiles = seq
	opt.Header = !noHeader

	// Positionals: expand globs; accept literals as-is; unmatched globs error.
	if len(posArgs) > 0 {
		for _, a := range posArgs {
			// Pass through "-" (stdin)
			if a == "-" {
				opt.SeqFiles = append(opt.SeqFiles, a)
				continue
			}
			if hasGlobMeta(a) {
				matches, err := filepath.Glob(a)
				if err != nil {
					return opt, fmt.Errorf("bad glob %q: %v", a, err)
				}
				if len(matches) == 0 {
					return opt, fmt.Errorf("no input matched %q", a)
				}
				opt.SeqFiles = append(opt.SeqFiles, matches...)
				continue
			}
			// No glob meta: treat as a literal path without requiring it to exist.
			opt.SeqFiles = append(opt.SeqFiles, a)
		}
	}

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
		return opt, errors.New("at least one sequence file is required")
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
	if opt.NoMatchExitCode < 0 || opt.NoMatchExitCode > 255 {
		return opt, errors.New("--no-match-exit-code must be between 0 and 255")
	}
	return opt, nil
}
