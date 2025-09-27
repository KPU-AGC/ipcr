// ./internal/nestedapp/app.go
package nestedapp

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"ipcr/internal/appcore"
	"ipcr/internal/engine"
	"ipcr/internal/nestedcli"
	"ipcr/internal/primer"
	"ipcr/internal/runutil"
	"ipcr/internal/version"
	"ipcr/internal/visitors"
	"ipcr/internal/writers"
)

func RunContext(parent context.Context, argv []string, stdout, stderr io.Writer) int {
	outw := bufio.NewWriter(stdout)
	defer outw.Flush()

	fs := nestedcli.NewFlagSet("ipcr-nested")
	fs.SetOutput(io.Discard)

	if len(argv) == 0 {
		_, _ = nestedcli.ParseArgs(fs, []string{"-h"})
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

	opts, err := nestedcli.ParseArgs(fs, argv)
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
		fmt.Fprintf(outw, "ipcr version %s (ipcr-nested)\n", version.Version)
		if err := outw.Flush(); writers.IsBrokenPipe(err) {
			return 0
		} else if err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		return 0
	}

	// Outer primer pairs
	var outer []primer.Pair
	if opts.PrimerFile != "" {
		outer, err = primer.LoadTSV(opts.PrimerFile)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	} else {
		outer = []primer.Pair{{
			ID:         "outer",
			Forward:    opts.Fwd,
			Reverse:    opts.Rev,
			MinProduct: opts.MinLen,
			MaxProduct: opts.MaxLen,
		}}
	}

	// Inner primer pairs
	var inner []primer.Pair
	if opts.InnerPrimerFile != "" {
		inner, err = primer.LoadTSV(opts.InnerPrimerFile)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	} else {
		inner = []primer.Pair{{
			ID:      "inner",
			Forward: strings.ToUpper(opts.InnerFwd),
			Reverse: strings.ToUpper(opts.InnerRev),
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

	writer := appcore.NewNestedWriterFactory(opts.Output, opts.Sort, opts.Header)

	visitor := visitors.Nested{
		InnerPairs: inner,
		EngineCfg: engine.Config{
			MaxMM:          opts.Mismatches,
			TerminalWindow: termWin,
			SeedLen:        opts.SeedLength,
			Circular:       false, // inner stage scans linear amplicon strings
			NeedSites:      false,
		},
		RequireInner: opts.RequireInner,
	}

	return appcore.Run(parent, stdout, stderr, coreOpts, outer, visitor.Visit, writer)
}

func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}
