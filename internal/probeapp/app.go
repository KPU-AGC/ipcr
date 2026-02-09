// internal/probeapp/app.go
package probeapp

import (
	"context"
	"fmt"
	"io"
	"ipcr-core/primer"
	"ipcr/internal/appcore"
	"ipcr/internal/common"
	"ipcr/internal/probecli"
	"ipcr/internal/runutil"
	"ipcr/internal/version"
	"ipcr/internal/visitors"
	"strings"
)

func RunContext(parent context.Context, argv []string, stdout, stderr io.Writer) int {
	fs := probecli.NewFlagSet("ipcr-probe")
	fs.SetOutput(io.Discard)

	opts, err := probecli.ParseArgs(fs, argv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	if opts.Version {
		_, _ = fmt.Fprintf(stdout, "ipcr version %s (ipcr-probe)\n", version.Version)
		return 0
	}

	var pairs []primer.Pair
	if opts.PrimerFile != "" {
		pairs, err = primer.LoadTSV(opts.PrimerFile)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 2
		}
	} else {
		pairs = []primer.Pair{{ID: "manual", Forward: opts.Fwd, Reverse: opts.Rev, MinProduct: opts.MinLen, MaxProduct: opts.MaxLen}}
	}
	if opts.Self {
		pairs = common.AddSelfPairs(pairs)
	}

	termWin := runutil.EffectiveTerminalWindow(opts.TerminalWindow)
	coreOpts := appcore.Options{
		SeqFiles:        opts.SeqFiles,
		MaxMM:           opts.Mismatches,
		TerminalWindow:  termWin,
		MinLen:          opts.MinLen,
		MaxLen:          opts.MaxLen,
		HitCap:          opts.HitCap,
		SeedLength:      opts.SeedLength,
		Circular:        opts.Circular,
		AllowSoftmask:   opts.AllowSoftmask,
		Threads:         opts.Threads,
		ChunkSize:       opts.ChunkSize,
		DedupeCap:       opts.DedupeCap,
		Quiet:           opts.Quiet,
		NoMatchExitCode: opts.NoMatchExitCode,
	}

	// WriterFactory for annotated products is 4 args: (format, sort, header, pretty)
	wf := appcore.NewAnnotatedWriterFactory(opts.Output, opts.Sort, opts.Header, opts.Pretty)

	v := visitors.Probe{
		Name:    strings.TrimSpace(opts.ProbeName),
		Seq:     strings.TrimSpace(opts.Probe),
		MaxMM:   opts.ProbeMaxMM,
		Require: opts.RequireProbe,
	}

	visit := v.Visit
	return appcore.Run(parent, stdout, stderr, coreOpts, pairs, visit, wf)
}

func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}
