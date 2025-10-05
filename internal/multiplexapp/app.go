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
	"strings"
)

func uniqueUpper(a []string) []string {
	seen := make(map[string]struct{}, len(a))
	out := make([]string, 0, len(a))
	for _, s := range a {
		u := strings.ToUpper(strings.TrimSpace(s))
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}

// collectPools scans argv to capture *all* occurrences of --forward/-f and --reverse/-r.
// It supports both "--flag val" and "--flag=val" forms. Values are *not* split on commas.
func collectPools(argv []string) (fwds, revs []string) {
	nextVal := func(i int) (string, int) {
		// caller ensures i < len(argv)
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
			val, j := nextVal(i + 1)
			if val != "" {
				fwds = append(fwds, val)
				i = j - 1
			}
		case strings.HasPrefix(arg, "-f="):
			fwds = append(fwds, strings.TrimPrefix(arg, "-f="))
		case arg == "-f":
			val, j := nextVal(i + 1)
			if val != "" {
				fwds = append(fwds, val)
				i = j - 1
			}

		case strings.HasPrefix(arg, "--reverse="):
			revs = append(revs, strings.TrimPrefix(arg, "--reverse="))
		case arg == "--reverse":
			val, j := nextVal(i + 1)
			if val != "" {
				revs = append(revs, val)
				i = j - 1
			}
		case strings.HasPrefix(arg, "-r="):
			revs = append(revs, strings.TrimPrefix(arg, "-r="))
		case arg == "-r":
			val, j := nextVal(i + 1)
			if val != "" {
				revs = append(revs, val)
				i = j - 1
			}
		}
	}
	return uniqueUpper(fwds), uniqueUpper(revs)
}

func expandPairsFromPools(fwds, revs []string, minLen, maxLen int) []primer.Pair {
	out := make([]primer.Pair, 0, len(fwds)*len(revs))
	for i, f := range fwds {
		for j, r := range revs {
			out = append(out, primer.Pair{
				ID:         fmt.Sprintf("F%d+R%d", i+1, j+1),
				Forward:    f,
				Reverse:    r,
				MinProduct: minLen,
				MaxProduct: maxLen,
			})
		}
	}
	return out
}

// self across pools: include *all* F×F and R×R (diagonal + cross) when --self=true.
func expandSelfAcrossPools(fwds, revs []string) []primer.Pair {
	out := make([]primer.Pair, 0, len(fwds)*len(fwds)+len(revs)*len(revs))
	for i, f1 := range fwds {
		for j, f2 := range fwds {
			out = append(out, primer.Pair{
				ID:      fmt.Sprintf("F%d+F%d:self", i+1, j+1),
				Forward: f1,
				Reverse: f2,
			})
		}
	}
	for i, r1 := range revs {
		for j, r2 := range revs {
			out = append(out, primer.Pair{
				ID:      fmt.Sprintf("R%d+R%d:self", i+1, j+1),
				Forward: r1,
				Reverse: r2,
			})
		}
	}
	return out
}

func addSelfPairsPerRow(pairs []primer.Pair) []primer.Pair {
	out := make([]primer.Pair, 0, len(pairs)+2*len(pairs))
	out = append(out, pairs...)
	seenA := make(map[string]struct{})
	seenB := make(map[string]struct{})
	for _, p := range pairs {
		if p.Forward != "" {
			u := strings.ToUpper(p.Forward)
			if _, ok := seenA[u]; !ok {
				seenA[u] = struct{}{}
				out = append(out, primer.Pair{ID: p.ID + "+A:self", Forward: u, Reverse: u})
			}
		}
		if p.Reverse != "" {
			u := strings.ToUpper(p.Reverse)
			if _, ok := seenB[u]; !ok {
				seenB[u] = struct{}{}
				out = append(out, primer.Pair{ID: p.ID + "+B:self", Forward: u, Reverse: u})
			}
		}
	}
	return out
}

func RunContext(parent context.Context, argv []string, stdout, stderr io.Writer) int {
	outw := bufio.NewWriter(stdout)
	defer func() { _ = outw.Flush() }()

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

	// Collect all repeated inline primers before parsing (ParseArgs will keep only the last).
	fPool, rPool := collectPools(argv)

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

	var pairs []primer.Pair

	switch {
	case opts.PrimerFile != "":
		// TSV mode: keep row semantics; optionally add A:self/B:self once per unique primer.
		pairs, err = primer.LoadTSV(opts.PrimerFile)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 2
		}
		if opts.Self {
			pairs = addSelfPairsPerRow(pairs)
		}

	default:
		// Inline mode.
		// If user only provided a single --forward/--reverse once, ParseArgs captured it; include it in pools.
		if len(fPool) == 0 && opts.Fwd != "" {
			fPool = []string{opts.Fwd}
		}
		if len(rPool) == 0 && opts.Rev != "" {
			rPool = []string{opts.Rev}
		}
		fPool = uniqueUpper(fPool)
		rPool = uniqueUpper(rPool)

		// Build pairs from pools:
		//   • all F × all R
		//   • if --self: all F × F and all R × R (diag + cross)
		switch {
		case len(fPool) > 0 && len(rPool) > 0:
			pairs = append(pairs, expandPairsFromPools(fPool, rPool, opts.MinLen, opts.MaxLen)...)
			if opts.Self {
				pairs = append(pairs, expandSelfAcrossPools(fPool, rPool)...)
			}
		case len(fPool) > 0 && len(rPool) == 0:
			if !opts.Self {
				_, _ = fmt.Fprintln(stderr, "error: repeatable --forward was supplied without --reverse; enable single-oligo amplification with --self=true")
				return 2
			}
			pairs = append(pairs, expandSelfAcrossPools(fPool, nil)...)
		case len(rPool) > 0 && len(fPool) == 0:
			if !opts.Self {
				_, _ = fmt.Fprintln(stderr, "error: repeatable --reverse was supplied without --forward; enable single-oligo amplification with --self=true")
				return 2
			}
			pairs = append(pairs, expandSelfAcrossPools(nil, rPool)...)
		default:
			// Nothing inline; fall back to validation error path (ParseArgs already ensures at least one of inline/TSV).
			_, _ = fmt.Fprintln(stderr, "error: provide --forward/--reverse (repeatable) or --primers TSV")
			return 2
		}
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
	vis := visitors.PassThrough{}
	wf := appcore.NewProductWriterFactory(opts.Output, opts.Sort, opts.Header, opts.Pretty, opts.Products, false, false)
	return appcore.Run[engine.Product](parent, outw, stderr, coreOpts, pairs, vis.Visit, wf)
}

func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}
