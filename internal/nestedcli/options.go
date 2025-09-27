// ./internal/nestedcli/options.go
package nestedcli

import (
	"flag"
	"fmt"
	"strings"

	"ipcr/internal/clibase"
	"ipcr/internal/cliutil"
)

type Options struct {
	// Outer (via Common)
	PrimerFile string
	Fwd        string
	Rev        string
	SeqFiles   []string

	Mismatches     int
	MinLen         int
	MaxLen         int
	HitCap         int
	TerminalWindow int
	Mode           string

	Threads    int
	ChunkSize  int
	SeedLength int
	Circular   bool

	Output          string
	Sort            bool
	Header          bool
	NoMatchExitCode int
	Pretty          bool // accepted but not used for nested (no pretty renderer yet)
	Products        bool // ignored

	Quiet   bool
	Version bool

	// Inner
	InnerPrimerFile string
	InnerFwd        string
	InnerRev        string
	RequireInner    bool
}

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {
		out := fs.Output()
		def := func(n string) string { if f := fs.Lookup(n); f != nil { return f.DefValue }; return "" }
		fmt.Fprintf(out, "%s – nested in-silico PCR (outer + inner)\n\n", name)
		fmt.Fprintln(out, "Usage:")
		fmt.Fprintf(out, "  %s --forward AAA --reverse TTT --inner-forward CCC --inner-reverse GGG ref.fa\n", name)
		fmt.Fprintf(out, "  %s --primers outer.tsv --inner-primers inner.tsv ref*.fa.gz\n", name)

		fmt.Fprintln(out, "\nOuter (same as ipcr):")
		fmt.Fprintln(out, "  -f, --forward string         Outer forward primer (5'→3')")
		fmt.Fprintln(out, "  -r, --reverse string         Outer reverse primer (5'→3')")
		fmt.Fprintln(out, "  -p, --primers string         Outer primer TSV (id fwd rev [min] [max])")
		fmt.Fprintln(out, "  -s, --sequences file         FASTA file(s) (repeatable) or '-' for STDIN")

		fmt.Fprintln(out, "\nInner:")
		fmt.Fprintln(out, "      --inner-forward string   Inner forward primer (5'→3')")
		fmt.Fprintln(out, "      --inner-reverse string   Inner reverse primer (5'→3')")
		fmt.Fprintln(out, "      --inner-primers string   Inner primer TSV (id fwd rev [min] [max])")
		fmt.Fprintf(out, "      --require-inner          Only keep outer amplicons that contain an inner product [%s]\n", def("require-inner"))

		fmt.Fprintln(out, "\nPCR/Performance/Output/Misc (outer stage; same as ipcr):")
		fmt.Fprintln(out, "      --mismatches, --min-length, --max-length, --terminal-window, --threads, --hit-cap, --chunk-size, --seed-length, --circular")
		fmt.Fprintln(out, "      --output text|json|jsonl|fasta, --sort, --no-header, --no-match-exit-code, --quiet, --version")
	}
	return fs
}

type stringSlice []string
func (s *stringSlice) String() string     { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

func ParseArgs(fs *flag.FlagSet, argv []string) (Options, error) {
	var o Options
	var help bool

	var c clibase.Common
	noHeader := clibase.Register(fs, &c)

	// Inner flags
	fs.StringVar(&o.InnerPrimerFile, "inner-primers", "", "Inner primer TSV")
	fs.StringVar(&o.InnerFwd, "inner-forward", "", "Inner forward primer (5'→3')")
	fs.StringVar(&o.InnerRev, "inner-reverse", "", "Inner reverse primer (5'→3')")
	fs.BoolVar(&o.RequireInner, "require-inner", false, "Only keep amplicons that contain an inner product [false]")

	// Help
	fs.BoolVar(&help, "h", false, "show this help [false]")

	flagArgs, posArgs := cliutil.SplitFlagsAndPositionals(fs, argv)
	if err := fs.Parse(flagArgs); err != nil { return o, err }
	if help { return o, flag.ErrHelp }
	if c.Version { o.Version = true; return o, nil }

	if err := clibase.AfterParse(fs, &c, noHeader, posArgs); err != nil { return o, err }

	// Validate inner
	usingFile := o.InnerPrimerFile != ""
	usingInline := o.InnerFwd != "" || o.InnerRev != ""
	switch {
	case usingFile && usingInline:
		return o, fmt.Errorf("--inner-primers conflicts with --inner-forward/--inner-reverse")
	case usingInline && (o.InnerFwd == "" || o.InnerRev == ""):
		return o, fmt.Errorf("--inner-forward and --inner-reverse must be supplied together")
	case !usingFile && !usingInline:
		return o, fmt.Errorf("provide --inner-primers or --inner-forward/--inner-reverse")
	}

	// Copy Common → Options
	o.PrimerFile, o.Fwd, o.Rev, o.SeqFiles = c.PrimerFile, c.Fwd, c.Rev, c.SeqFiles
	o.Mismatches, o.MinLen, o.MaxLen, o.HitCap, o.TerminalWindow, o.Mode =
		c.Mismatches, c.MinLen, c.MaxLen, c.HitCap, c.TerminalWindow, c.Mode
	o.Threads, o.ChunkSize, o.SeedLength, o.Circular = c.Threads, c.ChunkSize, c.SeedLength, c.Circular
	o.Output, o.Products, o.Pretty, o.Sort, o.Header, o.NoMatchExitCode =
		c.Output, c.Products, c.Pretty, c.Sort, c.Header, c.NoMatchExitCode
	o.Quiet = c.Quiet
	return o, nil
}

func Parse() (Options, error) { return ParseArgs(NewFlagSet("ipcr-nested"), nil) }
