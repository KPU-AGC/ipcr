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

// examplesFor prints a tiny block tailored to the CLI name.
func examplesFor(name string) func(io.Writer) {
	switch name {
	case "ipcr":
		return func(out io.Writer) {
			_, _ = fmt.Fprintln(out, "Standard in-silico PCR: scan references with a forward + reverse primer.")
			_, _ = fmt.Fprintln(out, "Find candidate amplicons under standard mismatch/length/3'-window rules.")
			_, _ = fmt.Fprintln(out, "\nExample:")
			_, _ = fmt.Fprintln(out, "  ipcr \\")
			_, _ = fmt.Fprintln(out, "    --forward AGAGTTTGATCMTGGCTCAG --reverse TACGGYTACCTTGTTAYGACTT \\")
			_, _ = fmt.Fprintln(out, "    --circular \\")
			_, _ = fmt.Fprintln(out, "    --pretty \\")
			_, _ = fmt.Fprintln(out, "    Escherichia-coli.fna.gz")
		}
	case "ipcr-multiplex":
		return func(out io.Writer) {
			_, _ = fmt.Fprintln(out, "Multiplex PCR: amplification with primer pools.")
			_, _ = fmt.Fprintln(out, "Build primer pools from repeated -f/-r flags or a --primers TSV.")
			_, _ = fmt.Fprintln(out, "Use --self to include A×rc(A) and B×rc(B) self-pairs.")
			_, _ = fmt.Fprintln(out, "\nExample:")
			_, _ = fmt.Fprintln(out, "  ipcr-multiplex \\")
			_, _ = fmt.Fprintln(out, "    --primers multiplex-pcr-assay.tsv \\")
			_, _ = fmt.Fprintln(out, "    --output jsonl \\")
			_, _ = fmt.Fprintln(out, "    --products \\")
			_, _ = fmt.Fprintln(out, "    --sort \\")
			_, _ = fmt.Fprintln(out, "    Salmonella-Typhimurium.fna.gz")
		}
	default:
		return nil
	}
}

// PrintExamples is called by apps after ParseArgs returns ErrPrintedAndExitOK.
func PrintExamples(out io.Writer, name string) {
	clibase.PrintExamples(out, name, examplesFor(name))
}

func ParseArgs(fs *flag.FlagSet, argv []string) (Options, error) {
	var o clibase.Common
	var help bool
	var showExamples bool

	noHeader := clibase.Register(fs, &o)
	fs.BoolVar(&help, "h", false, "show this help [false]")
	fs.BoolVar(&showExamples, "examples", false, "show quickstart examples and exit [false]")

	flagArgs, posArgs := cliutil.SplitFlagsAndPositionals(fs, argv)
	if err := fs.Parse(flagArgs); err != nil {
		return o, err
	}
	if showExamples {
		return o, clibase.ErrPrintedAndExitOK
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
