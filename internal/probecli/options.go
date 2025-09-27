// internal/probecli/options.go
package probecli

import (
	"flag"
	"fmt"
	"io"

	"ipcr/internal/clibase"
	"ipcr/internal/cliutil"
)

// NOTE: mode constants are defined once in clibase.

type Options struct {
	clibase.Common

	// Probe-specific
	Probe        string
	ProbeName    string
	ProbeMaxMM   int
	RequireProbe bool
}

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	clibase.UsageCommon(fs, name, func(out io.Writer, def func(string) string) {
		fmt.Fprintln(out, "Usage:")
		fmt.Fprintf(out, "  %s [options] --forward AAA --reverse TTT --probe PROBE ref.fa\n", name)

		fmt.Fprintln(out, "\nProbe:")
		fmt.Fprintln(out, "  -P, --probe string          Internal oligo sequence (5'→3') [required]")
		fmt.Fprintf(out, "      --probe-name string     Label for the probe [%s]\n", def("probe-name"))
		fmt.Fprintf(out, "  -M, --probe-max-mm int      Max mismatches allowed in probe match [%s]\n", def("probe-max-mm"))
		fmt.Fprintf(out, "      --require-probe         Only report amplicons that contain the probe [%s]\n", def("require-probe"))
	})
	return fs
}

func Parse() (Options, error) { return ParseArgs(NewFlagSet("ipcr-probe"), nil) }

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

	// Single assignment now that Options embeds Common.
	o.Common = c
	return o, nil
}
