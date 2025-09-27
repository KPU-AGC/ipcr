package app

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

// RunContext is the ipcr app entrypoint used by cmd/ipcr.
func RunContext(parent context.Context, argv []string, stdout, stderr io.Writer) int {
	outw := bufio.NewWriter(stdout)
	defer outw.Flush() // harmless if re-flushed later

	// Build a throwaway FlagSet so we can render usage cleanly.
	fs := cli.NewFlagSet("ipcr")
	fs.SetOutput(io.Discard)

	// No args → show help and exit 0.
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
			if e := outw.Flush(); writers.IsBrokenPipe(e) {
				return 0
			} else if e != nil {
				fmt.Fprintln(stderr, e)
				return 3
			}
			return 0
		}
		fmt.Fprintln(stderr, err)
		fs.SetOutput(outw)
		fs.Usage()
		if e := outw.Flush(); writers.IsBrokenPipe(e) {
			return 0
		} else if e != nil {
			fmt.Fprintln(stderr, e)
			return 3
		}
		return 2
	}

	if opts.Version {
		fmt.Fprintf(outw, "ipcr version %s\n", version.Version)
		if e := outw.Flush(); writers.IsBrokenPipe(e) {
			return 0
		} else if e != nil {
			fmt.Fprintln(stderr, e)
			return 3
		}
		return 0
	}

	// Primer pairs
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

	// Normalize options for the core runner.
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

	return appcore.Run[engine.Product](
		parent,
		stdout, stderr,
		coreOpts,
		pairs,
		visitors.PassThrough{}.Visit,
		writer,
	)
}

// Compatibility shim for older tests and callers.
// Signature matches previous app.Run(argv, stdout, stderr) style.
func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}
