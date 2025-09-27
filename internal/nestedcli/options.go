package nestedcli

import (
	"flag"
	"fmt"
	"io"

	"ipcr/internal/clibase"
	"ipcr/internal/cliutil"
)

type Options struct {
	clibase.Common

	// Inner
	InnerPrimerFile string
	InnerFwd        string
	InnerRev        string
	RequireInner    bool
}

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	clibase.UsageCommon(fs, name, func(out io.Writer, def func(string) string) {
		fmt.Fprintln(out, "Usage:")
		fmt.Fprintf(out, "  %s --forward AAA --reverse TTT --inner-forward CCC --inner-reverse GGG ref.fa\n", name)
		fmt.Fprintf(out, "  %s --primers outer.tsv --inner-primers inner.tsv ref*.fa.gz\n", name)

		fmt.Fprintln(out, "\nInner:")
		fmt.Fprintln(out, "      --inner-forward string   Inner forward primer (5'→3')")
		fmt.Fprintln(out, "      --inner-reverse string   Inner reverse primer (5'→3')")
		fmt.Fprintln(out, "      --inner-primers string   Inner primer TSV (id fwd rev [min] [max])")
		fmt.Fprintf(out, "      --require-inner          Only keep outer amplicons that contain an inner product [%s]\n", def("require-inner"))
	})
	return fs
}

func Parse() (Options, error) { return ParseArgs(NewFlagSet("ipcr-nested"), nil) }

func ParseArgs(fs *flag.FlagSet, argv []string) (Options, error) {
	var o Options
	var help bool

	var c clibase.Common
	noHeader := clibase.Register(fs, &c)

	// Inner flags + short aliases (-F/-R/-P for inner)
	fs.StringVar(&o.InnerPrimerFile, "inner-primers", "", "inner primer TSV")
	fs.StringVar(&o.InnerFwd, "inner-forward", "", "inner forward primer (5'→3')")
	fs.StringVar(&o.InnerRev, "inner-reverse", "", "inner reverse primer (5'→3')")
	fs.BoolVar(&o.RequireInner, "require-inner", false, "only keep outer amplicons that contain an inner product [false]")

	fs.StringVar(&o.InnerPrimerFile, "P", "", "alias of --inner-primers")
	fs.StringVar(&o.InnerFwd, "F", "", "alias of --inner-forward")
	fs.StringVar(&o.InnerRev, "R", "", "alias of --inner-reverse")

	// Help
	fs.BoolVar(&help, "h", false, "show this help [false]")

	flagArgs, posArgs := cliutil.SplitFlagsAndPositionals(fs, argv)
	if err := fs.Parse(flagArgs); err != nil { return o, err }
	if help { return o, flag.ErrHelp }
	if c.Version {
		o.Common = c
		return o, nil
	}

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

	o.Common = c
	return o, nil
}
