// internal/cli/options.go
package cli

import (
	"flag"
	"fmt"
	"strings"

	"ipcr/internal/clibase"
	"ipcr/internal/cliutil"
	"ipcr/internal/version"
)

const (
	ModeRealistic = "realistic"
	ModeDebug     = "debug"
)

type Options struct {
	PrimerFile string
	Fwd        string
	Rev        string
	SeqFiles   []string

	Mismatches     int
	MinLen         int
	MaxLen         int
	HitCap         int
	TerminalWindow int

	Threads    int
	ChunkSize  int
	SeedLength int
	Circular   bool

	Output          string
	Products        bool
	Pretty          bool
	Mode            string
	Sort            bool
	Header          bool
	NoMatchExitCode int

	Quiet   bool
	Version bool
}

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {
		out := fs.Output()
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
		fmt.Fprintf(out, "  %s [options] --forward AAA --reverse TTT ref*.fa gz/*.fa.gz\n", name)

		fmt.Fprintln(out, "\nInput:")
		fmt.Fprintln(out, "  -f, --forward string        Forward primer sequence (5'→3') [*]")
		fmt.Fprintln(out, "  -r, --reverse string        Reverse primer sequence (5'→3') [*]")
		fmt.Fprintln(out, "  -p, --primers string        TSV file of primer pairs [*]")
		fmt.Fprintln(out, "  -s, --sequences file        FASTA file(s) (repeatable) or '-' for STDIN [*]")

		fmt.Fprintln(out, "\nPCR Parameters:")
		fmt.Fprintf(out, "  -m, --mismatches int        Maximum mismatches per primer [%s]\n", def("mismatches"))
		fmt.Fprintf(out, "      --min-length int        Minimum product length [%s]\n", def("min-length"))
		fmt.Fprintf(out, "      --max-length int        Maximum product length [%s]\n", def("max-length"))
		fmt.Fprintf(out, "      --terminal-window int   3' terminal mismatch window (0=allow, -1=auto) [%s]\n", def("terminal-window"))
		fmt.Fprintf(out, "      --mode string           Matching mode: realistic | debug [%s]\n", def("mode"))
		fmt.Fprintf(out, "  -c, --circular              Treat each FASTA record as circular [%s]\n", def("circular"))

		fmt.Fprintln(out, "\nPerformance:")
		fmt.Fprintf(out, "  -t, --threads int           Number of worker threads (0 = all CPUs) [%s]\n", def("threads"))
		fmt.Fprintf(out, "      --hit-cap int           Max matches stored per primer/window (0 = unlimited) [%s]\n", def("hit-cap"))
		fmt.Fprintf(out, "      --chunk-size int        Chunk size (0 = no chunking) [%s]\n", def("chunk-size"))
		fmt.Fprintf(out, "      --seed-length int       Seed length for multi-pattern scan (0=auto) [%s]\n", def("seed-length"))

		fmt.Fprintln(out, "\nOutput:")
		fmt.Fprintf(out, "  -o, --output string         Output format: text | json | fasta [%s]\n", def("output"))
		fmt.Fprintf(out, "      --products              Emit product sequences [%s]\n", def("products"))
		fmt.Fprintf(out, "      --pretty                Pretty ASCII alignment block (text mode) [%s]\n", def("pretty"))
		fmt.Fprintf(out, "      --sort                  Sort outputs for determinism [%s]\n", def("sort"))
		fmt.Fprintf(out, "      --no-header             Suppress header line [%s]\n", def("no-header"))
		fmt.Fprintf(out, "      --no-match-exit-code int  Exit code when no amplicons found [%s]\n", def("no-match-exit-code"))

		fmt.Fprintln(out, "\nMiscellaneous:")
		fmt.Fprintf(out, "  -q, --quiet                 Suppress non-essential warnings [%s]\n", def("quiet"))
		fmt.Fprintln(out, "  -v, --version               Print version and exit")
		fmt.Fprintln(out, "  -h, --help                  Show this help and exit")
	}
	return fs
}

func Parse() (Options, error) { return ParseArgs(flag.CommandLine, nil) }

type stringSlice []string
func (s *stringSlice) String() string     { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

func ParseArgs(fs *flag.FlagSet, argv []string) (Options, error) {
	var o Options
	var help bool

	// Register shared flags
	var c clibase.Common
	noHeader := clibase.Register(fs, &c)

	// Help flag (so -h returns flag.ErrHelp like before)
	fs.BoolVar(&help, "h", false, "show this help [false]")

	// Split & parse
	flagArgs, posArgs := cliutil.SplitFlagsAndPositionals(fs, argv)
	if err := fs.Parse(flagArgs); err != nil {
		return o, err
	}
	if help {
		return o, flag.ErrHelp
	}
	if c.Version {
		// Copy the version bit through so callers can print and exit.
		o.Version = true
		return o, nil
	}

	// Finalize header, expand positionals, validate
	if err := clibase.AfterParse(fs, &c, noHeader, posArgs); err != nil {
		return o, err
	}

	// Copy Common → Options (field-for-field)
	o.PrimerFile, o.Fwd, o.Rev, o.SeqFiles = c.PrimerFile, c.Fwd, c.Rev, c.SeqFiles
	o.Mismatches, o.MinLen, o.MaxLen, o.HitCap, o.TerminalWindow = c.Mismatches, c.MinLen, c.MaxLen, c.HitCap, c.TerminalWindow
	o.Threads, o.ChunkSize, o.SeedLength, o.Circular = c.Threads, c.ChunkSize, c.SeedLength, c.Circular
	o.Output, o.Products, o.Pretty, o.Mode = c.Output, c.Products, c.Pretty, c.Mode
	o.Sort, o.Header, o.NoMatchExitCode = c.Sort, c.Header, c.NoMatchExitCode
	o.Quiet = c.Quiet
	return o, nil
}
