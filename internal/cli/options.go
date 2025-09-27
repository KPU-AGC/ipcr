// internal/cli/options.go
package cli

import (
	"flag"
	"fmt"
	"io"

	"ipcr/internal/clibase"
	"ipcr/internal/cliutil"
)

// NOTE: mode constants are defined once in clibase.

type Options struct{ clibase.Common }

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	clibase.UsageCommon(fs, name, func(out io.Writer, def func(string) string) {
		fmt.Fprintln(out, "Usage:")
		fmt.Fprintf(out, "  %s [options] --forward AAA --reverse TTT --sequences ref.fa\n", name)
		fmt.Fprintf(out, "  %s [options] --forward AAA --reverse TTT ref*.fa gz/*.fa.gz\n", name)
	})
	return fs
}

func Parse() (Options, error) { return ParseArgs(flag.CommandLine, nil) }

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

	// Single assignment now that Options embeds Common.
	o.Common = c
	return o, nil
}
