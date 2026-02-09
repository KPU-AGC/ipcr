// internal/multiplexapp/app.go
package multiplexapp

import (
	"context"
	"fmt"
	"io"
	"ipcr-core/engine"
	"ipcr-core/primer"
	"ipcr/internal/appcore"
	"ipcr/internal/cli"
	"ipcr/internal/common"
	"ipcr/internal/runutil"
	"ipcr/internal/version"
	"strings"
)

func collectPools(argv []string) (fwds, revs []string) {
	nextVal := func(i int) (string, int) {
		if i >= len(argv) {
			return "", i
		}
		s := argv[i]
		if s == "" || s[0] == '-' {
			return "", i
		}
		return s, i + 1
	}

	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		switch {
		case strings.HasPrefix(arg, "--forward="):
			fwds = append(fwds, strings.TrimPrefix(arg, "--forward="))
		case arg == "--forward":
			if val, j := nextVal(i + 1); val != "" {
				fwds, i = append(fwds, val), j-1
			}
		case strings.HasPrefix(arg, "-f="):
			fwds = append(fwds, strings.TrimPrefix(arg, "-f="))
		case arg == "-f":
			if val, j := nextVal(i + 1); val != "" {
				fwds, i = append(fwds, val), j-1
			}

		case strings.HasPrefix(arg, "--reverse="):
			revs = append(revs, strings.TrimPrefix(arg, "--reverse="))
		case arg == "--reverse":
			if val, j := nextVal(i + 1); val != "" {
				revs, i = append(revs, val), j-1
			}
		case strings.HasPrefix(arg, "-r="):
			revs = append(revs, strings.TrimPrefix(arg, "-r="))
		case arg == "-r":
			if val, j := nextVal(i + 1); val != "" {
				revs, i = append(revs, val), j-1
			}
		}
	}

	return common.UniqueUpper(fwds), common.UniqueUpper(revs)
}

func expandPairsFromPools(fwds, revs []string, minLen, maxLen int) []primer.Pair {
	out := make([]primer.Pair, 0, len(fwds)*len(revs))
	for i, f := range fwds {
		for j, r := range revs {
			out = append(out, primer.Pair{
				ID:         fmt.Sprintf("pool:%d+%d", i+1, j+1),
				Forward:    f,
				Reverse:    r,
				MinProduct: minLen,
				MaxProduct: maxLen,
			})
		}
	}
	return out
}

func RunContext(parent context.Context, argv []string, stdout, stderr io.Writer) int {
	fs := cli.NewFlagSet("ipcr-multiplex")
	fs.SetOutput(io.Discard)

	// capture repeated -f/-r before ParseArgs collapses them
	fPool, rPool := collectPools(argv)

	opts, err := cli.ParseArgs(fs, argv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	if opts.Version {
		_, _ = fmt.Fprintf(stdout, "ipcr version %s (ipcr-multiplex)\n", version.Version)
		return 0
	}

	var pairs []primer.Pair
	switch {
	case opts.PrimerFile != "":
		pairs, err = primer.LoadTSV(opts.PrimerFile)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 2
		}
		if opts.Self {
			pairs = common.AddSelfPairsUnique(pairs)
		}
	default:
		if len(fPool) > 0 && len(rPool) > 0 {
			pairs = expandPairsFromPools(fPool, rPool, opts.MinLen, opts.MaxLen)
		} else {
			pairs = []primer.Pair{{ID: "manual", Forward: opts.Fwd, Reverse: opts.Rev, MinProduct: opts.MinLen, MaxProduct: opts.MaxLen}}
		}
		if opts.Self {
			pairs = common.AddSelfPairs(pairs)
		}
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

	wf := appcore.NewProductWriterFactory(opts.Output, opts.Sort, opts.Header, opts.Pretty, opts.Products, false, false)
	visit := func(p engine.Product) (bool, engine.Product, error) { return true, p, nil }

	return appcore.Run(parent, stdout, stderr, coreOpts, pairs, visit, wf)
}

func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}
