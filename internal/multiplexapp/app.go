// internal/multiplexapp/app.go
package multiplexapp

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"ipcr-core/engine"
	"ipcr-core/primer"
	"ipcr/internal/appcore"
	"ipcr/internal/cli"
	"ipcr/internal/runutil"
	"ipcr/internal/version"
	"ipcr/internal/visitors"
	"ipcr/internal/writers"
)

func RunContext(parent context.Context, argv []string, stdout, stderr io.Writer) int {
	outw := bufio.NewWriter(stdout)
	defer func() { _ = outw.Flush() }()

	// Build a FlagSet so we can render usage like the main app.
	fs := cli.NewFlagSet("ipcr-multiplex")
	fs.SetOutput(io.Discard)

	// No args → help
	if len(argv) == 0 {
		_, _ = cli.ParseArgs(fs, []string{"-h"})
		fs.SetOutput(outw)
		fs.Usage()
		if err := outw.Flush(); writers.IsBrokenPipe(err) {
			return 0
		} else if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 3
		}
		return 0
	}

	// Parse standard options (supports --primers or inline A/B)
	opts, err := cli.ParseArgs(fs, argv)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(outw)
			fs.Usage()
			if e := outw.Flush(); writers.IsBrokenPipe(e) {
				return 0
			} else if e != nil {
				_, _ = fmt.Fprintln(stderr, e)
				return 3
			}
			return 0
		}
		_, _ = fmt.Fprintln(stderr, err)
		fs.SetOutput(outw)
		fs.Usage()
		if e := outw.Flush(); writers.IsBrokenPipe(e) {
			return 0
		} else if e != nil {
			_, _ = fmt.Fprintln(stderr, e)
			return 3
		}
		return 2
	}

	if opts.Version {
		_, _ = fmt.Fprintf(outw, "ipcr version %s (ipcr-multiplex)\n", version.Version)
		if e := outw.Flush(); writers.IsBrokenPipe(e) {
			return 0
		} else if e != nil {
			_, _ = fmt.Fprintln(stderr, e)
			return 3
		}
		return 0
	}

	// Build primer pairs: prefer TSV when provided
	var pairs []primer.Pair
	if opts.PrimerFile != "" {
		pairs, err = primer.LoadTSV(opts.PrimerFile)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 2
		}
	} else {
		pairs = []primer.Pair{{
			ID:         "manual",
			Forward:    opts.Fwd,
			Reverse:    opts.Rev,
			MinProduct: opts.MinLen,
			MaxProduct: opts.MaxLen,
		}}
	}

	termWin := runutil.ComputeTerminalWindow(opts.Mode, opts.TerminalWindow)
	coreOpts := appcore.Options{
		SeqFiles:        opts.SeqFiles,
		MaxMM:           opts.Mismatches,
		TerminalWindow:  termWin,
		MinLen:          opts.MinLen,
		MaxLen:          opts.MaxLen,
		HitCap:          opts.HitCap,
		SeedLength:      opts.SeedLength,
		Circular:        opts.Circular,
		Threads:         opts.Threads,
		ChunkSize:       opts.ChunkSize,
		Quiet:           opts.Quiet,
		NoMatchExitCode: opts.NoMatchExitCode,
	}

	// Multiplex just passes products through (different source of pairs)
	vis := visitors.PassThrough{}

	wf := appcore.NewProductWriterFactory(opts.Output, opts.Sort, opts.Header, opts.Pretty, opts.Products)
	return appcore.Run[engine.Product](
		parent, stdout, stderr,
		coreOpts,
		pairs,
		vis.Visit,
		wf,
	)
}

// Compatibility shim for tests: same signature as other apps.
func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}
