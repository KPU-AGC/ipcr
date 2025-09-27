// ./internal/multiplexapp/app.go
package multiplexapp

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"ipcr/internal/appcore"
	"ipcr/internal/cli"
	"ipcr-core/engine"
	"ipcr-core/primer"
	"ipcr/internal/runutil"
	"ipcr/internal/version"
	"ipcr/internal/visitors"
	"ipcr/internal/writers"
)

func RunContext(parent context.Context, argv []string, stdout, stderr io.Writer) int {
	outw := bufio.NewWriter(stdout)
	defer outw.Flush()

	fs := cli.NewFlagSet("ipcr-multiplex")
	fs.SetOutput(io.Discard)

	if len(argv) == 0 {
		_, _ = cli.ParseArgs(fs, []string{"-h"})
		fs.SetOutput(outw)
		fs.Usage()
		if err := outw.Flush(); writers.IsBrokenPipe(err) {
			return 0
		} else if err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		return 0
	}

	opts, err := cli.ParseArgs(fs, argv)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(outw)
			fs.Usage()
			if err := outw.Flush(); writers.IsBrokenPipe(err) {
				return 0
			} else if err != nil {
				fmt.Fprintln(stderr, err)
				return 3
			}
			return 0
		}
		fmt.Fprintln(stderr, err)
		fs.SetOutput(outw)
		fs.Usage()
		if err := outw.Flush(); writers.IsBrokenPipe(err) {
			return 0
		} else if err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		return 2
	}

	if opts.Version {
		fmt.Fprintf(outw, "ipcr version %s (ipcr-multiplex)\n", version.Version)
		if err := outw.Flush(); writers.IsBrokenPipe(err) {
			return 0
		} else if err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		return 0
	}

	// Primer pairs (supports TSV with many pairs = multiplex)
	var pairs []primer.Pair
	if opts.PrimerFile != "" {
		pairs, err = primer.LoadTSV(opts.PrimerFile)
		if err != nil {
			fmt.Fprintln(stderr, err)
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
	writer := appcore.NewProductWriterFactory(opts.Output, opts.Sort, opts.Header, opts.Pretty, opts.Products)

	// For now this is pass-through; later you can add per-panel constraints.
	return appcore.Run[engine.Product](parent, stdout, stderr, coreOpts, pairs, visitors.PassThrough{}.Visit, writer)
}

func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}
