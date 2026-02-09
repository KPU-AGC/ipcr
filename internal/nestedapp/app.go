// internal/nestedapp/app.go
package nestedapp

import (
	"context"
	"fmt"
	"io"
	"ipcr-core/engine"
	"ipcr-core/primer"
	"ipcr/internal/appcore"
	"ipcr/internal/common"
	"ipcr/internal/nestedcli"
	"ipcr/internal/runutil"
	"ipcr/internal/version"
	"ipcr/internal/visitors"
)

func RunContext(parent context.Context, argv []string, stdout, stderr io.Writer) int {
	fs := nestedcli.NewFlagSet("ipcr-nested")
	fs.SetOutput(io.Discard)

	opts, err := nestedcli.ParseArgs(fs, argv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	if opts.Version {
		_, _ = fmt.Fprintf(stdout, "ipcr-nested version %s\n", version.Version)
		return 0
	}

	// Outer pairs
	var outerPairs []primer.Pair
	if opts.PrimerFile != "" {
		outerPairs, err = primer.LoadTSV(opts.PrimerFile)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 2
		}
	} else {
		outerPairs = []primer.Pair{{ID: "outer", Forward: opts.Fwd, Reverse: opts.Rev, MinProduct: opts.MinLen, MaxProduct: opts.MaxLen}}
	}
	if opts.Self {
		outerPairs = common.AddSelfPairs(outerPairs)
	}

	// Inner pairs
	var innerPairs []primer.Pair
	if opts.InnerPrimerFile != "" {
		innerPairs, err = primer.LoadTSV(opts.InnerPrimerFile)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 2
		}
	} else {
		innerPairs = []primer.Pair{{ID: "inner", Forward: opts.InnerFwd, Reverse: opts.InnerRev}}
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

	// Nested visitor exists as a struct with Visit method.
	v := visitors.Nested{
		InnerPairs:   innerPairs,
		EngineCfg:    engine.Config{MaxMM: opts.Mismatches, TerminalWindow: termWin, SeedLen: opts.SeedLength, AllowSoftmask: opts.AllowSoftmask, NeedSites: opts.Pretty},
		RequireInner: opts.RequireInner,
	}

	wf := appcore.NewNestedWriterFactory(opts.Output, opts.Sort, opts.Header, opts.Pretty)
	return appcore.Run(parent, stdout, stderr, coreOpts, outerPairs, v.Visit, wf)
}

func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}
