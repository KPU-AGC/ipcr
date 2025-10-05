// internal/probeapp/app.go  (REPLACE)
package probeapp

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"ipcr-core/primer"
	"ipcr/internal/appcore"
	"ipcr/internal/common"
	"ipcr/internal/probecli"
	"ipcr/internal/runutil"
	"ipcr/internal/version"
	"ipcr/internal/visitors"
	"ipcr/internal/writers"
	"strings"
)

// RunContext is the ipcr-probe app entrypoint used by cmd/ipcr-probe.
func RunContext(parent context.Context, argv []string, stdout, stderr io.Writer) int {
	outw := bufio.NewWriter(stdout)
	defer outw.Flush()

	fs := probecli.NewFlagSet("ipcr-probe")
	fs.SetOutput(io.Discard)

	if len(argv) == 0 {
		_, _ = probecli.ParseArgs(fs, []string{"-h"})
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

	opts, err := probecli.ParseArgs(fs, argv)
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
		_, _ = fmt.Fprintf(outw, "ipcr version %s (ipcr-probe)\n", version.Version)
		if e := outw.Flush(); writers.IsBrokenPipe(e) {
			return 0
		} else if e != nil {
			_, _ = fmt.Fprintln(stderr, e)
			return 3
		}
		return 0
	}

	var pairs []primer.Pair
	if opts.PrimerFile != "" {
		var e error
		pairs, e = primer.LoadTSV(opts.PrimerFile)
		if e != nil {
			_, _ = fmt.Fprintln(stderr, e)
			return 2
		}
	} else {
		pairs = []primer.Pair{{ID: "manual", Forward: opts.Fwd, Reverse: opts.Rev, MinProduct: opts.MinLen, MaxProduct: opts.MaxLen}}
	}
	if opts.Self {
		pairs = common.AddSelfPairs(pairs)
	}

	termWin := runutil.ComputeTerminalWindow(opts.Mode, opts.TerminalWindow)
	coreOpts := appcore.Options{
		SeqFiles: opts.SeqFiles, MaxMM: opts.Mismatches, TerminalWindow: termWin,
		MinLen: opts.MinLen, MaxLen: opts.MaxLen, HitCap: opts.HitCap, SeedLength: opts.SeedLength,
		Circular: opts.Circular, Threads: opts.Threads, ChunkSize: opts.ChunkSize,
		Quiet: opts.Quiet, NoMatchExitCode: opts.NoMatchExitCode,
	}
	writer := appcore.NewAnnotatedWriterFactory(opts.Output, opts.Sort, opts.Header, opts.Pretty)
	visitor := visitors.Probe{
		Name: opts.ProbeName, Seq: strings.ToUpper(opts.Probe), MaxMM: opts.ProbeMaxMM, Require: opts.RequireProbe,
	}
	return appcore.Run(parent, stdout, stderr, coreOpts, pairs, visitor.Visit, writer)
}

func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}
