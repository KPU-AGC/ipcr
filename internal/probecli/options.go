package probecli

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

// Modes (kept for parity with ipcr)
const (
	ModeRealistic = "realistic"
	ModeDebug     = "debug"
)

type Options struct {
	// Inputs
	PrimerFile string
	Fwd        string
	Rev        string
	SeqFiles   []string

	// Probe
	Probe        string
	ProbeName    string
	ProbeMaxMM   int
	RequireProbe bool

	// PCR params
	Mismatches     int
	MinLen         int
	MaxLen         int
	HitCap         int
	TerminalWindow int

	// Performance
	Threads    int
	ChunkSize  int
	SeedLength int
	Circular   bool

	// Output
	Output          string // text | json | fasta
	Products        bool   // not used for filtering; FASTA needs sequences anyway
	Sort            bool
	Header          bool
	NoMatchExitCode int
	Mode            string

	// Misc
	Quiet   bool
	Version bool
}

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {
		out := fs.Output()
		def := func(n string) string {
			if f := fs.Lookup(n); f != nil {
				return f.DefValue
			}
			return ""
		}
		fmt.Fprintf(out, "%s – in-silico PCR + internal probe scan\n\n", name)
		fmt.Fprintln(out, "Usage:")
		fmt.Fprintf(out, "  %s [options] --forward AAA --reverse TTT --probe PROBE ref.fa\n", name)
		fmt.Fprintln(out, "\nRequirements:")
		fmt.Fprintln(out, "  • Provide either --primers OR (--forward AND --reverse).")
		fmt.Fprintln(out, "  • Provide at least one FASTA input.")
		fmt.Fprintln(out, "  • Provide --probe (sequence 5'→3').")

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

		fmt.Fprintln(out, "\nPCR Parameters:")
		fmt.Fprintf(out, "  -m, --mismatches int        Max mismatches per primer [%s]\n", def("mismatches"))
		fmt.Fprintf(out, "      --min-length int        Minimum product length [%s]\n", def("min-length"))
		fmt.Fprintf(out, "      --max-length int        Maximum product length [%s]\n", def("max-length"))
		fmt.Fprintf(out, "      --terminal-window int   3' terminal window for primers (0=allow, -1=auto) [%s]\n", def("terminal-window"))
		fmt.Fprintf(out, "      --mode string           Matching mode: realistic | debug [%s]\n", def("mode"))
		fmt.Fprintf(out, "  -c, --circular              Treat each record as circular [%s]\n", def("circular"))

		fmt.Fprintln(out, "\nPerformance:")
		fmt.Fprintf(out, "  -t, --threads int           Worker threads (0 = all CPUs) [%s]\n", def("threads"))
		fmt.Fprintf(out, "      --hit-cap int           Max matches stored per primer/window (0=unlimited) [%s]\n", def("hit-cap"))
		fmt.Fprintf(out, "      --chunk-size int        Chunk size (0 = no chunking) [%s]\n", def("chunk-size"))
		fmt.Fprintf(out, "      --seed-length int       Seed length for multi-pattern scan (0=auto) [%s]\n", def("seed-length"))

		fmt.Fprintln(out, "\nOutput:")
		fmt.Fprintf(out, "  -o, --output string         text | json | fasta [%s]\n", def("output"))
		fmt.Fprintf(out, "      --sort                  Sort outputs for determinism [%s]\n", def("sort"))
		fmt.Fprintf(out, "      --no-header             Suppress header line [false]\n")
		fmt.Fprintf(out, "      --no-match-exit-code int  Exit code when no amplicons found [%s]\n", def("no-match-exit-code"))

		fmt.Fprintln(out, "\nMisc:")
		fmt.Fprintf(out, "  -q, --quiet                 Suppress non-essential warnings [%s]\n", def("quiet"))
		fmt.Fprintln(out, "  -v, --version               Print version and exit")
		fmt.Fprintln(out, "  -h, --help                  Show this help and exit")
	}
	return fs
}

type stringSlice []string

func (s *stringSlice) String() string     { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

func hasGlobMeta(s string) bool { return strings.ContainsAny(s, "*?[") }

// split interspersed flags and positional args
func splitFlagsAndPositionals(fs *flag.FlagSet, argv []string) (flagArgs, posArgs []string) {
	boolFlags := map[string]bool{}
	fs.VisitAll(func(f *flag.Flag) {
		if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
			boolFlags[f.Name] = true
		}
	})
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if arg == "--" {
			posArgs = append(posArgs, argv[i+1:]...)
			break
		}
		if arg == "-" {
			posArgs = append(posArgs, arg)
			continue
		}
		if strings.HasPrefix(arg, "-") {
			if strings.Contains(arg, "=") {
				flagArgs = append(flagArgs, arg)
				continue
			}
			name := strings.TrimLeft(arg, "-")
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				name = name[:eq]
			}
			needsVal := !boolFlags[name]
			flagArgs = append(flagArgs, arg)
			if needsVal && i+1 < len(argv) {
				flagArgs = append(flagArgs, argv[i+1])
				i++
			}
			continue
		}
		posArgs = append(posArgs, arg)
	}
	return
}

func ParseArgs(fs *flag.FlagSet, argv []string) (Options, error) {
	var o Options
	var help bool

	// Inputs
	fs.StringVar(&o.PrimerFile, "primers", "", "TSV primer file")
	fs.StringVar(&o.Fwd, "forward", "", "forward primer (5'→3')")
	fs.StringVar(&o.Rev, "reverse", "", "reverse primer (5'→3')")
	var seq stringSlice
	fs.Var(&seq, "sequences", "FASTA file(s) or '-'")

	// Short aliases
	fs.StringVar(&o.PrimerFile, "p", "", "alias of --primers")
	fs.StringVar(&o.Fwd, "f", "", "alias of --forward")
	fs.StringVar(&o.Rev, "r", "", "alias of --reverse")
	fs.Var(&seq, "s", "alias of --sequences")

	// Probe
	fs.StringVar(&o.Probe, "probe", "", "internal oligo (5'→3') [required]")
	fs.StringVar(&o.ProbeName, "probe-name", "probe", "probe label")
	fs.IntVar(&o.ProbeMaxMM, "probe-max-mm", 0, "max mismatches allowed for probe")
	fs.BoolVar(&o.RequireProbe, "require-probe", true, "only report amplicons that contain the probe [true]")
	// Short aliases:
	fs.IntVar(&o.ProbeMaxMM, "M", 0, "alias of --probe-max-mm")
	fs.StringVar(&o.Probe, "P", "", "alias of --probe")

	// PCR params
	fs.IntVar(&o.Mismatches, "mismatches", 0, "max mismatches per primer [0]")
	fs.IntVar(&o.MinLen, "min-length", 0, "minimum product length [0]")
	fs.IntVar(&o.MaxLen, "max-length", 2000, "maximum product length [2000]")
	fs.IntVar(&o.HitCap, "hit-cap", 10000, "max matches stored per primer/window (0=unlimited) [10000]")
	fs.IntVar(&o.TerminalWindow, "terminal-window", -1, "3' terminal window for primers (0=allow, -1=auto) [-1]")
	fs.StringVar(&o.Mode, "mode", ModeRealistic, "matching mode: realistic | debug")
	fs.BoolVar(&o.Circular, "circular", false, "treat each FASTA record as circular [false]")
	fs.IntVar(&o.SeedLength, "seed-length", 12, "seed length for multi-pattern scan (0=auto) [12]")

	// Performance
	fs.IntVar(&o.Threads, "threads", 0, "worker threads (0 = all CPUs) [0]")
	fs.IntVar(&o.ChunkSize, "chunk-size", 0, "split sequences into N-bp windows (0=no chunking) [0]")

	// Output / misc
	fs.StringVar(&o.Output, "output", "text", "output: text | json | fasta [text]")
	fs.StringVar(&o.Output, "o", "text", "alias of --output")
	fs.BoolVar(&o.Sort, "sort", false, "sort outputs for determinism [false]")
	noHeader := false
	fs.BoolVar(&noHeader, "no-header", false, "suppress header line [false]")
	fs.IntVar(&o.NoMatchExitCode, "no-match-exit-code", 1, "exit code when no amplicons are found [1]")

	fs.BoolVar(&o.Quiet, "quiet", false, "suppress non-essential warnings [false]")
	fs.BoolVar(&o.Version, "v", false, "print version and exit (shorthand) [false]")
	fs.BoolVar(&o.Version, "version", false, "print version and exit [false]")
	fs.BoolVar(&help, "h", false, "show this help [false]")
	fs.BoolVar(&o.Circular, "c", false, "alias of --circular")
	fs.IntVar(&o.Mismatches, "m", 0, "alias of --mismatches")
	fs.IntVar(&o.Threads, "t", 0, "alias of --threads")

	flagArgs, posArgs := splitFlagsAndPositionals(fs, argv)
	if err := fs.Parse(flagArgs); err != nil {
		return o, err
	}
	if help {
		return o, flag.ErrHelp
	}
	if o.Version {
		return o, nil
	}

	o.SeqFiles = seq
	o.Header = !noHeader

	// Expand positional globs
	for _, a := range posArgs {
		if a == "-" {
			o.SeqFiles = append(o.SeqFiles, a)
			continue
		}
		if hasGlobMeta(a) {
			m, err := filepath.Glob(a)
			if err != nil {
				return o, fmt.Errorf("bad glob %q: %v", a, err)
			}
			if len(m) == 0 {
				return o, fmt.Errorf("no input matched %q", a)
			}
			o.SeqFiles = append(o.SeqFiles, m...)
		} else {
			o.SeqFiles = append(o.SeqFiles, a)
		}
	}

	// Validation
	usingFile := o.PrimerFile != ""
	usingInline := o.Fwd != "" || o.Rev != ""
	switch {
	case usingFile && usingInline:
		return o, errors.New("--primers conflicts with --forward/--reverse")
	case usingInline && (o.Fwd == "" || o.Rev == ""):
		return o, errors.New("--forward and --reverse must be supplied together")
	case !usingFile && !usingInline:
		return o, errors.New("provide --primers or --forward/--reverse")
	}
	if len(o.SeqFiles) == 0 {
		return o, errors.New("at least one sequence file is required")
	}
	if o.Probe == "" {
		return o, errors.New("--probe is required")
	}
	if o.Threads < 0 || o.ChunkSize < 0 || o.HitCap < 0 {
		return o, errors.New("--threads/--chunk-size/--hit-cap must be ≥ 0")
	}
	if o.Output != "text" && o.Output != "json" && o.Output != "fasta" {
		return o, fmt.Errorf("invalid --output %q", o.Output)
	}
	if o.TerminalWindow < -1 {
		return o, errors.New("--terminal-window must be ≥ -1")
	}
	if o.NoMatchExitCode < 0 || o.NoMatchExitCode > 255 {
		return o, errors.New("--no-match-exit-code must be between 0 and 255")
	}
	return o, nil
}

func Parse() (Options, error) { return ParseArgs(NewFlagSet("ipcr-probe"), nil) }
