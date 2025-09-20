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
//
// Contract:
//   • Results (text/TSV/JSON/FASTA) go to stdout; diagnostics/warnings/errors go to stderr.
//   • Writers are provided by the caller and are never closed (stdout uses an internal bufio.Writer and is flushed).
//   • -h/--help writes usage to stdout and returns 0. --version writes to stdout and returns 0.
//   • Exit codes:
//       0  success (including help, version, and SIGPIPE on output)
//       1  no matches (default; configurable via --no-match-exit-code)
//       2  CLI/usage/config errors
//       3  runtime I/O or processing errors
//     130  canceled by signal (SIGINT/SIGTERM)
//   • SIGPIPE‑friendliness: if a write returns EPIPE/closed pipe, the run is canceled and treated as success (0).
func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}

// RunContext is like Run but observes ctx for cancellation (e.g., Ctrl‑C).
func RunContext(parent context.Context, argv []string, stdout, stderr io.Writer) int {
	// Buffer stdout for throughput but keep streaming behavior.
	outw := bufio.NewWriterSize(stdout, 64<<10) // 64 KiB

	isBrokenPipe := func(err error) bool {
		if err == nil {
			return false
		}
		// syscall.EPIPE is the canonical signal; io.ErrClosedPipe may occur on some platforms.
		return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
	}

	// Build a flag set whose default output is discarded; we print help/errors explicitly.
	fs := cli.NewFlagSet("ipcr")
	fs.SetOutput(io.Discard)

	opts, err := cli.ParseArgs(fs, argv)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// Print usage text to stdout with exit code 0.
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
		// CLI/usage errors → stderr, non‑zero.
		fmt.Fprintln(stderr, err)
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
			ID:         "manual",
			Forward:    opts.Fwd,
			Reverse:    opts.Rev,
			MinProduct: opts.MinLen,
			MaxProduct: opts.MaxLen,
		}}
	}

	// Compute maximal primer length across pairs
	maxPLen := 0
	for _, pr := range pairs {
		if l := len(pr.Forward); l > maxPLen {
			maxPLen = l
		}
		if l := len(pr.Reverse); l > maxPLen {
			maxPLen = l
		}
	}

	// Determine terminal 3' window policy (auto: realistic=3, debug=0)
	termWin := opts.TerminalWindow
	if termWin < 0 {
		if opts.Mode == cli.ModeRealistic {
			termWin = 3
		} else {
			termWin = 0
		}
	}

	// ---- Config sanity checks -------------------------------------------------
	// 1) max-length must be >= longest primer
	if opts.MaxLen > 0 && opts.MaxLen < maxPLen {
		fmt.Fprintf(stderr, "error: --max-length (%d) is smaller than the longest primer length (%d)\n", opts.MaxLen, maxPLen)
		return 2
	}
	// 2) min-length must not exceed max-length
	if opts.MinLen > 0 && opts.MaxLen > 0 && opts.MinLen > opts.MaxLen {
		fmt.Fprintf(stderr, "error: --min-length (%d) exceeds --max-length (%d)\n", opts.MinLen, opts.MaxLen)
		return 2
	}
	// 3) warn if mismatches are mostly blocked by the 3′ window
	if opts.Mismatches > 0 && !opts.Quiet {
		for _, pr := range pairs {
			for _, seg := range []struct {
				name string
				L    int
			}{{"forward", len(pr.Forward)}, {"reverse", len(pr.Reverse)}} {
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
	// --------------------------------------------------------------------------

	// Decide safe chunking/overlap policy
	overlap := 0
	if opts.ChunkSize > 0 {
		if opts.MaxLen <= 0 {
			if !opts.Quiet {
				fmt.Fprintln(stderr, "warning: --chunk-size requires --max-length to ensure correctness; disabling chunking")
			}
			opts.ChunkSize = 0
		} else if opts.ChunkSize <= opts.MaxLen {
			if !opts.Quiet {
				fmt.Fprintln(stderr, "warning: --chunk-size must be > --max-length; disabling chunking to avoid boundary misses")
			}
			opts.ChunkSize = 0
		} else {
			// Guarantee that every product with length ≤ MaxLen lies fully in some chunk
			overlap = opts.MaxLen
			if mp := maxPLen - 1; mp > overlap {
				overlap = mp
			}
		}
	}

	// Set up PCR engine
	eng := engine.New(engine.Config{
		MaxMM:          opts.Mismatches,
		TerminalWindow: termWin,
		MinLen:         opts.MinLen,
		MaxLen:         opts.MaxLen,
	})
	eng.SetHitCap(opts.HitCap)

	// Set up worker pool
	thr := opts.Threads
	if thr <= 0 {
		thr = runtime.NumCPU()
	}

	jobs := make(chan job, thr*2)
	results := make(chan result, thr*2)
	prodCh := make(chan engine.Product, thr*4) // to writer

	// Ensure sequences are captured when needed:
	// - --products, text+--pretty, or FASTA output
	needSeq := opts.Products || (opts.Output == "text" && opts.Pretty) || (opts.Output == "fasta")

	// Derived context we can cancel internally (e.g., on EPIPE)
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	// Writer goroutine (single writer to avoid interleaving)
	writeErr := make(chan error, 1)
	go func() {
		var werr error
		switch opts.Output {
		case "json":
			// JSON requires buffering to emit a valid array.
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

		// On broken pipe, stop the world quickly.
		if isBrokenPipe(werr) {
			cancel()
		}
		writeErr <- werr
	}()

	// Worker goroutines
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
					hits := eng.Simulate(j.rec.ID, j.rec.Seq, j.pair)
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

	// Forwarder/collector with cross‑chunk de‑duplication
	type dkey struct {
		base       string
		start, end int
		typ        string
		exp        string
	}
	var (
		colErr    error
		totalHits int
		collectWg sync.WaitGroup
		seen      = make(map[dkey]struct{}, 1<<12)
	)
	collectWg.Add(1)
	go func() {
		defer collectWg.Done()
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
				gStart := p.Start + off
				gEnd := p.End + off
				k := dkey{base: base, start: gStart, end: gEnd, typ: p.Type, exp: p.ExperimentID}
				if _, dup := seen[k]; dup {
					continue
				}
				seen[k] = struct{}{}
				// If canceled (e.g., EPIPE), keep draining but stop forwarding to writer.
				if ctx.Err() != nil {
					continue
				}
				select {
				case prodCh <- p:
					totalHits++
				case <-ctx.Done():
				}
			}
		}
	}()

	// Feed jobs to workers
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
			for _, pr := range pairs {
				select {
				case jobs <- job{rec: rec, pair: pr}:
				case <-ctx.Done():
					break feed
				}
			}
		}
	}

	// Graceful close and shutdown
	close(jobs)
	wg.Wait()
	close(results)
	collectWg.Wait()
	close(prodCh)

	// Writer result
	werr := <-writeErr
	if isBrokenPipe(werr) {
		// Downstream closed the pipe; treat as successful completion.
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
	// Flush stdout last; treat broken pipe as success.
	if ferr := outw.Flush(); isBrokenPipe(ferr) {
		return 0
	} else if ferr != nil {
		fmt.Fprintln(stderr, ferr)
		return 3
	}
	if ctx.Err() != nil {
		// Canceled by signal (not by SIGPIPE path above)
		return 130
	}
	if totalHits == 0 {
		return opts.NoMatchExitCode // user‑configurable (grep‑style default = 1)
	}
	return 0
}

// job and result are used for parallel PCR simulation.
type job struct {
	rec  fasta.Record
	pair primer.Pair
}
type result struct {
	prods []engine.Product
	err   error
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

// splitChunkSuffix parses an id that may end with ":<start>-<end>" and returns (baseID, startOffset, ok).
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
