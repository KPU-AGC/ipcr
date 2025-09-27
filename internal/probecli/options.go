// internal/probecli/options.go
package probecli

import (
	"flag"
	"fmt"
	"strings"

	"ipcr/internal/clibase"
	"ipcr/internal/cliutil"
)

const (
	ModeRealistic = "realistic"
	ModeDebug     = "debug"
)

type Options struct {
	// Shared (embed the common fields)
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
	Products        bool // parity with ipcr
	Pretty          bool
	Sort            bool
	Header          bool
	NoMatchExitCode int

	Quiet   bool
	Version bool

	// Probe-specific
	Probe        string
	ProbeName    string
	ProbeMaxMM   int
	RequireProbe bool
}

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {
		out := fs.Output()
		def := func(n string) string { if f := fs.Lookup(n); f != nil { return f.DefValue }; return "" }
		fmt.Fprintf(out, "%s – in-silico PCR + internal probe scan\n\n", name)
		fmt.Fprintln(out, "Usage:")
		fmt.Fprintf(out, "  %s [options] --forward AAA --reverse TTT --probe PROBE ref.fa\n", name)

		fmt.Fprintln(out, "\nInput:")
		fmt.Fprintln(out, "  -f, --forward string        Forward primer (5'→3')")
		fmt.Fprintln(out, "  -r, --reverse string        Reverse primer (5'→3')")
		fmt.Fprintln(out, "  -p, --primers string        Primer TSV (id fwd rev [min] [max])")
		fmt.Fprintln(out, "  -s, --sequences file        FASTA file(s) (repeatable) or '-' for STDIN")

		fmt.Fprintln(out, "\nProbe:")
		fmt.Fprintln(out, "  -P, --probe string          Internal oligo sequence (5'→3') [required]")
		fmt.Fprintf(out, "      --probe-name string     Label for the probe [%s]\n", def("probe-name"))
		fmt.Fprintf(out, "  -M, --probe-max-mm int      Max mismatches allowed in probe match [%s]\n", def("probe-max-mm"))
		fmt.Fprintf(out, "      --require-probe         Only report amplicons that contain the probe [%s]\n", def("require-probe"))

		fmt.Fprintln(out, "\nPCR / Performance / Output / Misc: (same as ipcr)")
		fmt.Fprintln(out, "      --output text | json | jsonl | fasta")
	}
	return fs
}

type stringSlice []string
func (s *stringSlice) String() string     { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

func ParseArgs(fs *flag.FlagSet, argv []string) (Options, error) {
	var o Options
	var help bool

	// Shared flags via clibase
	var c clibase.Common
	noHeader := clibase.Register(fs, &c)

	// Probe flags
	fs.StringVar(&o.Probe, "probe", "", "internal oligo (5'→3') [required]")
	fs.StringVar(&o.ProbeName, "probe-name", "probe", "probe label")
	fs.IntVar(&o.ProbeMaxMM, "probe-max-mm", 0, "max mismatches allowed for probe [0]")
	fs.BoolVar(&o.RequireProbe, "require-probe", true, "only report amplicons that contain the probe [true]")
	fs.IntVar(&o.ProbeMaxMM, "M", 0, "alias of --probe-max-mm")
	fs.StringVar(&o.Probe, "P", "", "alias of --probe")

	// Help
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
		o.Version = true
		return o, nil
	}

	// Finalize header, expand pos, shared validation
	if err := clibase.AfterParse(fs, &c, noHeader, posArgs); err != nil {
		return o, err
	}
	// Probe-specific validation
	if o.Probe == "" {
		return o, fmt.Errorf("--probe is required")
	}

	// Copy shared → Options
	o.PrimerFile, o.Fwd, o.Rev, o.SeqFiles = c.PrimerFile, c.Fwd, c.Rev, c.SeqFiles
	o.Mismatches, o.MinLen, o.MaxLen, o.HitCap, o.TerminalWindow, o.Mode =
		c.Mismatches, c.MinLen, c.MaxLen, c.HitCap, c.TerminalWindow, c.Mode
	o.Threads, o.ChunkSize, o.SeedLength, o.Circular = c.Threads, c.ChunkSize, c.SeedLength, c.Circular
	o.Output, o.Products, o.Pretty, o.Sort, o.Header, o.NoMatchExitCode =
		c.Output, c.Products, c.Pretty, c.Sort, c.Header, c.NoMatchExitCode
	o.Quiet = c.Quiet
	return o, nil
}

func Parse() (Options, error) { return ParseArgs(NewFlagSet("ipcr-probe"), nil) }
