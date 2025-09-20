// internal/app/app.go
package app

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"ipcr/internal/cli"
	"ipcr/internal/engine"
	"ipcr/internal/fasta"
	"ipcr/internal/output"
	"ipcr/internal/primer"
	"ipcr/internal/version"
)

// Run streams the program’s results to the provided writers and returns an exit code.
func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}

func RunContext(parent context.Context, argv []string, stdout, stderr io.Writer) int {
	outw := bufio.NewWriterSize(stdout, 64<<10)

	isBrokenPipe := func(err error) bool {
		return err != nil && (errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe))
	}

	fs := cli.NewFlagSet("ipcr")
	// Silence default flag package error text; we control printing.
	fs.SetOutput(io.Discard)

	// If no args, we still need flags registered so PrintDefaults shows up.
	// Parse a synthetic "-h" to register flags (returns flag.ErrHelp), then print usage.
	if len(argv) == 0 {
		_, _ = cli.ParseArgs(fs, []string{"-h"}) // registers flags on fs
		fs.SetOutput(outw)
		fs.Usage()
		if ferr := outw.Flush(); isBrokenPipe(ferr) {
			return 0
		} else if ferr != nil {
			fmt.Fprintln(stderr, ferr)
			return 3
		}
		return 0
	}

	// Normal parse path
	opts, err := cli.ParseArgs(fs, argv)
	if err != nil {
		// -h/--help: show usage and exit 0
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(outw)
			fs.Usage()
			if ferr := outw.Flush(); isBrokenPipe(ferr) {
				return 0
			} else if ferr != nil {
				fmt.Fprintln(stderr, ferr)
				return 3
			}
			return 0
		}
		// Any other parse error: show the error and usage; exit 2.
		fmt.Fprintln(stderr, err)
		// Ensure flags are registered before calling Usage (ParseArgs already did).
		fs.SetOutput(outw)
		fs.Usage()
		if ferr := outw.Flush(); isBrokenPipe(ferr) {
			return 0
		} else if ferr != nil {
			fmt.Fprintln(stderr, ferr)
			return 3
		}
		return 2
	}
	if opts.Version {
		fmt.Fprintf(outw, "ipcr version %s\n", version.Version)
		if ferr := outw.Flush(); isBrokenPipe(ferr) {
			return 0
		} else if ferr != nil {
			fmt.Fprintln(stderr, ferr)
			return 3
		}
		return 0
	}

	// Load primer pairs
	var pairs []primer.Pair
	if opts.PrimerFile != "" {
		pairs, err = primer.LoadTSV(opts.PrimerFile)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	} else {
		pairs = []primer.Pair{{
			ID: "manual", Forward: opts.Fwd, Reverse: opts.Rev,
			MinProduct: opts.MinLen, MaxProduct: opts.MaxLen,
		}}
	}

	// Max primer length
	maxPLen := 0
	for _, pr := range pairs {
		if l := len(pr.Forward); l > maxPLen {
			maxPLen = l
		}
		if l := len(pr.Reverse); l > maxPLen {
			maxPLen = l
		}
	}

	// Terminal 3' window (auto by mode)
	termWin := opts.TerminalWindow
	if termWin < 0 {
		if opts.Mode == cli.ModeRealistic {
			termWin = 3
		} else {
			termWin = 0
		}
	}

	// Sanity checks
	if opts.MaxLen > 0 && opts.MaxLen < maxPLen {
		fmt.Fprintf(stderr, "error: --max-length (%d) is smaller than the longest primer length (%d)\n", opts.MaxLen, maxPLen)
		return 2
	}
	if opts.MinLen > 0 && opts.MaxLen > 0 && opts.MinLen > opts.MaxLen {
		fmt.Fprintf(stderr, "error: --min-length (%d) exceeds --max-length (%d)\n", opts.MinLen, opts.MaxLen)
		return 2
	}
	if opts.Mismatches > 0 && !opts.Quiet {
		for _, pr := range pairs {
			for _, seg := range []struct{ name string; L int }{
				{"forward", len(pr.Forward)}, {"reverse", len(pr.Reverse)},
			} {
				if seg.L == 0 {
					continue
				}
				eff := seg.L - termWin
				if eff < 0 {
					eff = 0
				}
				if eff == 0 && opts.Mismatches > 0 {
					fmt.Fprintf(stderr, "warning: terminal-window (%d) ≥ %s primer length (%d) for %q; mismatches are effectively disallowed along the entire primer\n",
						termWin, seg.name, seg.L, pr.ID)
					continue
				}
				if eff > 0 && opts.Mismatches >= eff {
					fmt.Fprintf(stderr, "warning: --mismatches (%d) ≥ %s primer length − terminal-window (%d − %d = %d) for %q; many candidates will be rejected by the 3′ policy\n",
						opts.Mismatches, seg.name, seg.L, termWin, eff, pr.ID)
				}
			}
		}
	}

	// Chunking / overlap
	overlap := 0
	if opts.ChunkSize > 0 {
		if opts.MaxLen <= 0 {
			if !opts.Quiet {
				fmt.Fprintln(stderr, "warning: --chunk-size requires --max-length; disabling chunking")
			}
			opts.ChunkSize = 0
		} else if opts.ChunkSize <= opts.MaxLen {
			if !opts.Quiet {
				fmt.Fprintln(stderr, "warning: --chunk-size must be > --max-length; disabling")
			}
			opts.ChunkSize = 0
		} else {
			overlap = opts.MaxLen
			if mp := maxPLen - 1; mp > overlap {
				overlap = mp
			}
		}
	}

	// Engine (NeedSites only for pretty text) — includes seed-length
	eng := engine.New(engine.Config{
		MaxMM:          opts.Mismatches,
		TerminalWindow: termWin,
		MinLen:         opts.MinLen,
		MaxLen:         opts.MaxLen,
		HitCap:         opts.HitCap,
		NeedSites:      (opts.Output == "text" && opts.Pretty),
		SeedLen:        opts.SeedLength,
	})

	// Workers
	thr := opts.Threads
	if thr <= 0 {
		thr = runtime.NumCPU()
	}

	type job struct {
		rec   fasta.Record
		pairs []primer.Pair
	}
	type result struct {
		prods []engine.Product
		err   error
	}

	jobs := make(chan job, thr*2)
	results := make(chan result, thr*2)
	prodCh := make(chan engine.Product, thr*4)

	needSeq := opts.Products || (opts.Output == "text" && opts.Pretty) || (opts.Output == "fasta")

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	// Writer
	writeErr := make(chan error, 1)
	go func() {
		var werr error
		switch opts.Output {
		case "json":
			var buf []engine.Product
			for p := range prodCh {
				buf = append(buf, p)
			}
			if opts.Sort {
				sortProducts(buf)
			}
			werr = output.WriteJSON(outw, buf)
		case "fasta":
			if opts.Sort {
				var buf []engine.Product
				for p := range prodCh {
					buf = append(buf, p)
				}
				sortProducts(buf)
				werr = output.WriteFASTA(outw, buf)
				break
			}
			werr = output.StreamFASTA(outw, prodCh)
		case "text":
			if opts.Sort {
				var buf []engine.Product
				for p := range prodCh {
					buf = append(buf, p)
				}
				sortProducts(buf)
				werr = output.WriteText(outw, buf, opts.Header, opts.Pretty)
				break
			}
			werr = output.StreamText(outw, prodCh, opts.Header, opts.Pretty)
		default:
			werr = fmt.Errorf("unsupported output %q", opts.Output)
		}
		if isBrokenPipe(werr) {
			cancel()
		}
		writeErr <- werr
	}()

	// Workers (one scan per record across ALL pairs)
	var wg sync.WaitGroup
	wg.Add(thr)
	for w := 0; w < thr; w++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case j, ok := <-jobs:
					if !ok {
						return
					}
					hits := eng.SimulateBatch(j.rec.ID, j.rec.Seq, j.pairs)
					if needSeq {
						for i := range hits {
							hits[i].Seq = string(j.rec.Seq[hits[i].Start:hits[i].End])
						}
					}
					select {
					case results <- result{prods: hits}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// Collector (de‑dup across chunks)
	type dkey struct {
		base       string
		start, end int
		typ, exp   string
	}
	var (
		colErr error
		total  int
		cwg    sync.WaitGroup
		seen   = make(map[dkey]struct{}, 1<<12)
	)
	cwg.Add(1)
	go func() {
		defer cwg.Done()
		for r := range results {
			if r.err != nil && colErr == nil {
				colErr = r.err
			}
			for _, p := range r.prods {
				base, off, ok := splitChunkSuffix(p.SequenceID)
				if !ok {
					base = p.SequenceID
					off = 0
				}
				gs, ge := p.Start+off, p.End+off
				k := dkey{base: base, start: gs, end: ge, typ: p.Type, exp: p.ExperimentID}
				if _, dup := seen[k]; dup {
					continue
				}
				seen[k] = struct{}{}
				if ctx.Err() != nil {
					continue
				}
				select {
				case prodCh <- p:
					total++
				case <-ctx.Done():
				}
			}
		}
	}()

	// Feed: ONE job per record (all pairs)
feed:
	for _, fa := range opts.SeqFiles {
		rch, err := fasta.StreamChunks(fa, opts.ChunkSize, overlap)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		for rec := range rch {
			if ctx.Err() != nil {
				break feed
			}
			select {
			case jobs <- job{rec: rec, pairs: pairs}:
			case <-ctx.Done():
				break feed
			}
		}
	}

	close(jobs)
	wg.Wait()
	close(results)
	cwg.Wait()
	close(prodCh)

	werr := <-writeErr
	if isBrokenPipe(werr) {
		return 0
	}
	if werr != nil {
		fmt.Fprintln(stderr, werr)
		return 3
	}
	if colErr != nil {
		fmt.Fprintln(stderr, colErr)
		return 3
	}
	if ferr := outw.Flush(); isBrokenPipe(ferr) {
		return 0
	} else if ferr != nil {
		fmt.Fprintln(stderr, ferr)
		return 3
	}
	if ctx.Err() != nil {
		return 130
	}
	if total == 0 {
		return opts.NoMatchExitCode
	}
	return 0
}

// sort key: (SequenceID, Start, End, Type, ExperimentID)
func sortProducts(a []engine.Product) {
	sort.Slice(a, func(i, j int) bool {
		ai, aj := a[i], a[j]
		if ai.SequenceID != aj.SequenceID {
			return ai.SequenceID < aj.SequenceID
		}
		if ai.Start != aj.Start {
			return ai.Start < aj.Start
		}
		if ai.End != aj.End {
			return ai.End < aj.End
		}
		if ai.Type != aj.Type {
			return ai.Type < aj.Type
		}
		return ai.ExperimentID < aj.ExperimentID
	})
}

// splitChunkSuffix parses an id that may end with ":<start>-<end>"
func splitChunkSuffix(id string) (string, int, bool) {
	colon := strings.LastIndex(id, ":")
	if colon == -1 || colon == len(id)-1 {
		return id, 0, false
	}
	suffix := id[colon+1:]
	dash := strings.IndexByte(suffix, '-')
	if dash == -1 {
		return id, 0, false
	}
	startStr := suffix[:dash]
	if start, err := strconv.Atoi(startStr); err == nil {
		return id[:colon], start, true
	}
	return id, 0, false
}
