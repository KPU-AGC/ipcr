package cli

import (
	"flag"
	"fmt"
	"io"
	"ipcr/internal/clibase"
	"ipcr/internal/cliutil"
)

type Options = clibase.Common

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	clibase.UsageCommon(fs, name, func(out io.Writer, def func(string) string) {
		_, _ = fmt.Fprintln(out, "Usage:")
		_, _ = fmt.Fprintf(out, "  %s [options] --forward AAA --reverse TTT --sequences ref.fa\n", name)
		_, _ = fmt.Fprintf(out, "  %s [options] --forward AAA --reverse TTT ref*.fa gz/*.fa.gz\n", name)
	})
	return fs
}

func Parse() (Options, error) { return ParseArgs(flag.CommandLine, nil) }

func ParseArgs(fs *flag.FlagSet, argv []string) (Options, error) {
	var o clibase.Common
	var help bool

	noHeader := clibase.Register(fs, &o)
	fs.BoolVar(&help, "h", false, "show this help [false]")

	flagArgs, posArgs := cliutil.SplitFlagsAndPositionals(fs, argv)
	if err := fs.Parse(flagArgs); err != nil {
		return o, err
	}
	if help {
		return o, flag.ErrHelp
	}
	if o.Version {
		return o, nil
	}
	if err := clibase.AfterParse(fs, &o, noHeader, posArgs); err != nil {
		return o, err
	}
	return o, nil
}
