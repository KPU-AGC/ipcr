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

	// NOTE: Usage relies on flags having been registered on fs.
	// In app.RunContext, we call ParseArgs(fs, []string{"-h"}) to register flags
	// before invoking fs.Usage(), so fs.Lookup() returns correct defaults.
	fs.Usage = func() {
		out := fs.Output()

		// Helpers to fetch defaults from registered flags.
		def := func(flagName string) string {
			if f := fs.Lookup(flagName); f != nil {
				return f.DefValue
			}
			return ""
		}

		fmt.Fprintf(out, "%s – in-silico PCR\n\n", name)
		fmt.Fprintf(out, "Author:  Erick Samera (erick.samera@kpu.ca)\n")
		fmt.Fprintf(out, "License: MIT\n")
		fmt.Fprintf(out, "Version: %s\n\n", version.Version)

		fmt.Fprintln(out, "Usage:")
		fmt.Fprintf(out, "  %s [options] --forward AAA --reverse TTT --sequences ref.fa\n", name)
		fmt.Fprintln(out, "      (flags for inputs; no glob expansion for --sequences)")
		fmt.Fprintf(out, "  %s [options] --forward AAA --reverse TTT ref*.fa gz/*.fa.gz\n", name)
		fmt.Fprintln(out, "      (positional FASTA args; globs are expanded)")

		fmt.Fprintln(out, "\nRequirements:")
		fmt.Fprintln(out, "  • Provide either --primers OR (--forward AND --reverse).")
		fmt.Fprintln(out, "  • Provide at least one FASTA input (via --sequences or positionals).")

		fmt.Fprintln(out, "\nFlags (grouped by category; [*] marks required inputs):")

		// -------------------- Input --------------------
		fmt.Fprintln(out, "Input:")
		fmt.Fprintln(out, "  -f, --forward string        Forward primer sequence (5'→3') [*]")
		fmt.Fprintln(out, "  -r, --reverse string        Reverse primer sequence (5'→3') [*]")
		fmt.Fprintln(out, "  -p, --primers string        TSV file of primer pairs (forward & reverse) [*]")
		fmt.Fprintln(out, "  -s, --sequences file        FASTA file(s) (repeatable) or \"-\" for STDIN [*]")

		// -------------------- PCR Parameters --------------------
		fmt.Fprintln(out, "\nPCR Parameters:")
		fmt.Fprintf(out, "  -m, --mismatches int        Maximum mismatches per primer [%s]\n", def("mismatches"))
		fmt.Fprintf(out, "      --min-length int        Minimum product length [%s]\n", def("min-length"))
		fmt.Fprintf(out, "      --max-length int        Maximum product length [%s]\n", def("max-length"))
		fmt.Fprintf(out, "      --hit-cap int           Max matches stored per primer/window (0 = unlimited) [%s]\n", def("hit-cap"))
		fmt.Fprintf(out, "      --terminal-window int   3' terminal mismatch window (0=allow, -1=auto) [%s]\n", def("terminal-window"))
		fmt.Fprintf(out, "      --mode string           Matching mode: realistic | debug [%s]\n", def("mode"))
		fmt.Fprintf(out, "  -c, --circular              Treat each FASTA record as circular (wrap-around) [%s]\n", def("circular"))

		// -------------------- Performance --------------------
		fmt.Fprintln(out, "\nPerformance:")
		fmt.Fprintf(out, "  -t, --threads int           Number of worker threads (0 = all CPUs) [%s]\n", def("threads"))
		fmt.Fprintf(out, "      --chunk-size int        Chunk size for splitting sequences (0 = no chunking) [%s]\n", def("chunk-size"))
		fmt.Fprintf(out, "      --seed-length int       Seed length for multi-pattern scan (0=auto) [%s]\n", def("seed-length"))

		// -------------------- Output --------------------
		fmt.Fprintln(out, "\nOutput:")
		fmt.Fprintf(out, "  -o, --output string         Output format: text | json | fasta [%s]\n", def("output"))
		fmt.Fprintf(out, "      --products              Emit product sequences [%s]\n", def("products"))
		fmt.Fprintf(out, "      --pretty                Pretty ASCII alignment block (text mode) [%s]\n", def("pretty"))
		fmt.Fprintf(out, "      --sort                  Sort outputs for determinism [%s]\n", def("sort"))
		fmt.Fprintf(out, "      --no-header             Suppress header line in text/TSV [%s]\n", def("no-header"))
		fmt.Fprintf(out, "      --no-match-exit-code int  Exit code when no amplicons found (0=success) [%s]\n", def("no-match-exit-code"))

		// -------------------- Misc --------------------
		fmt.Fprintln(out, "\nMiscellaneous:")
		fmt.Fprintf(out, "  -q, --quiet                 Suppress non-essential warnings [%s]\n", def("quiet"))
		fmt.Fprintln(out, "  -v, --version               Print version and exit")
		fmt.Fprintln(out, "  -h, --help                  Show this help message and exit")
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

	// --------------------- File & primer input ---------------------
	fs.StringVar(&opt.PrimerFile, "primers", "", "TSV primer file [*]")
	fs.StringVar(&opt.Fwd, "forward", "", "forward primer (5'→3') [*]")
	fs.StringVar(&opt.Rev, "reverse", "", "reverse primer (5'→3') [*]")
	var seq stringSlice
	fs.Var(&seq, "sequences", "FASTA file(s) (repeatable or '-') [*] (literal; no glob expansion here)")

	// Short aliases for input
	fs.StringVar(&opt.PrimerFile, "p", "", "alias of --primers")
	fs.StringVar(&opt.Fwd, "f", "", "alias of --forward")
	fs.StringVar(&opt.Rev, "r", "", "alias of --reverse")
	fs.Var(&seq, "s", "alias of --sequences")

	// ------------------------- PCR parameters -------------------------
	fs.IntVar(&opt.Mismatches, "mismatches", 0, "max mismatches per primer [0]")
	fs.IntVar(&opt.MinLen, "min-length", 0, "minimum product length [0]")
	fs.IntVar(&opt.MaxLen, "max-length", 2000, "maximum product length [2000]")
	fs.IntVar(&opt.HitCap, "hit-cap", 10000, "max matches stored per primer per window (0 = unlimited) [10000]")
	fs.IntVar(&opt.TerminalWindow, "terminal-window", -1, "3' terminal window (nt) disallowed for mismatches (0=allow, -1=auto: realistic=3, debug=0) [-1]")

	// Short alias where non-conflicting
	fs.IntVar(&opt.Mismatches, "m", 0, "alias of --mismatches")

	// ---------------------------- Performance ----------------------------
	fs.IntVar(&opt.Threads, "threads", 0, "number of worker threads (0 = all CPUs) [0]")
	fs.IntVar(&opt.ChunkSize, "chunk-size", 0, "split sequences into N-bp windows (0 = no chunking) [0]")
	fs.IntVar(&opt.SeedLength, "seed-length", 12, "seed length for multi-pattern scan (0=auto: min(12, primer length)) [12]")

	// Short aliases
	fs.IntVar(&opt.Threads, "t", 0, "alias of --threads")

	// ------------------------- Output / misc behavior -------------------------
	fs.StringVar(&opt.Output, "output", "text", "output format: text | json | fasta [text]")
	fs.StringVar(&opt.Mode, "mode", ModeRealistic, "matching mode: realistic | debug ["+ModeRealistic+"]")
	fs.BoolVar(&opt.Circular, "circular", false, "treat each FASTA record as circular (disables chunking) [false]")
	fs.BoolVar(&opt.Products, "products", false, "emit product sequences [false]")
	fs.BoolVar(&opt.Pretty, "pretty", false, "pretty ASCII alignment block (text) [false]")
	fs.BoolVar(&opt.Sort, "sort", false, "sort outputs for determinism (SequenceID,Start,End,Type,ExperimentID) [false]")
	noHeader := false
	fs.BoolVar(&noHeader, "no-header", false, "suppress header line in text/TSV [false]")
	fs.IntVar(&opt.NoMatchExitCode, "no-match-exit-code", 1, "exit code to use when no amplicons are found (set 0 to treat as success) [1]")

	// Short aliases
	fs.StringVar(&opt.Output, "o", "text", "alias of --output")
	fs.BoolVar(&opt.Circular, "c", false, "alias of --circular")

	// ------------------------------- Misc --------------------------------
	fs.BoolVar(&opt.Quiet, "quiet", false, "suppress non-essential warnings on stderr [false]")
	fs.BoolVar(&opt.Version, "v", false, "print version and exit (shorthand) [false]")
	fs.BoolVar(&opt.Version, "version", false, "print version and exit [false]")
	fs.BoolVar(&help, "h", false, "show this help message (shorthand) [false]")

	// Short alias
	fs.BoolVar(&opt.Quiet, "q", false, "alias of --quiet")

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
