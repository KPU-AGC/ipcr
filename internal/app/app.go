// internal/app/app.go
package app

import (
	"bytes"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"ipcr/internal/cli"
	"ipcr/internal/engine"
	"ipcr/internal/fasta"
	"ipcr/internal/output"
	"ipcr/internal/primer"
	"ipcr/internal/version"
)

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
	// We only treat the *trailing* ":start-end" as a chunk marker to allow colons in the base id.
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

// Run executes the full pipeline and returns the process exit code.
func Run(argv []string, stdout, stderr *bytes.Buffer) int {
	// Parse CLI options
	fs := cli.NewFlagSet("ipcr")
	opts, err := cli.ParseArgs(fs, argv)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if opts.Version {
		fmt.Fprintf(stdout, "ipcr version %s\n", version.Version)
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
	// 3) warn if mismatches are so high they’re mostly blocked by the 3′ window
	if opts.Mismatches > 0 {
		for _, pr := range pairs {
			type seg struct {
				name string
				L    int
			}
			for _, s := range []seg{{"forward", len(pr.Forward)}, {"reverse", len(pr.Reverse)}} {
				if s.L == 0 {
					continue
				}
				eff := s.L - termWin
				if eff < 0 {
					eff = 0
				}
				if eff == 0 && opts.Mismatches > 0 {
					fmt.Fprintf(stderr, "warning: terminal-window (%d) ≥ %s primer length (%d) for %q; mismatches are effectively disallowed along the entire primer\n",
						termWin, s.name, s.L, pr.ID)
					continue
				}
				if eff > 0 && opts.Mismatches >= eff {
					fmt.Fprintf(stderr, "warning: --mismatches (%d) ≥ %s primer length − terminal-window (%d − %d = %d) for %q; many candidates will be rejected by the 3′ policy\n",
						opts.Mismatches, s.name, s.L, termWin, eff, pr.ID)
				}
			}
		}
	}
	// --------------------------------------------------------------------------

	// Decide safe chunking/overlap policy
	overlap := 0
	if opts.ChunkSize > 0 {
		if opts.MaxLen <= 0 {
			fmt.Fprintln(stderr, "warning: --chunk-size requires --max-length to ensure correctness; disabling chunking")
			opts.ChunkSize = 0
		} else if opts.ChunkSize <= opts.MaxLen {
			fmt.Fprintln(stderr, "warning: --chunk-size must be > --max-length; disabling chunking to avoid boundary misses")
			opts.ChunkSize = 0
		} else {
			// Guarantee that every product with length ≤ MaxLen lies fully in some chunk:
			// need overlap ≥ MaxLen (and also ≥ maxPrimerLen-1 for per-primer scanning).
			overlap = opts.MaxLen
			if mp := maxPLen - 1; mp > overlap {
				overlap = mp
			}
			// (slide = chunkSize - overlap) will be ≥ 1 because chunkSize > MaxLen
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
	// - --products
	// - text+--pretty
	// - FASTA output (always needs Seq)
	needSeq := opts.Products || (opts.Output == "text" && opts.Pretty) || (opts.Output == "fasta")

	// Writer goroutine
	writeErr := make(chan error, 1)
	go func() {
		var err error
		switch opts.Output {
		case "json":
			var buf []engine.Product
			for p := range prodCh {
				buf = append(buf, p)
			}
			if opts.Sort {
				sortProducts(buf)
			}
			err = output.WriteJSON(stdout, buf)

		case "fasta":
			if opts.Sort {
				var buf []engine.Product
				for p := range prodCh {
					buf = append(buf, p)
				}
				sortProducts(buf)
				err = output.WriteFASTA(stdout, buf)
				break
			}
			err = output.StreamFASTA(stdout, prodCh)

		case "text":
			if opts.Sort {
				var buf []engine.Product
				for p := range prodCh {
					buf = append(buf, p)
				}
				sortProducts(buf)
				err = output.WriteText(stdout, buf, opts.Header, opts.Pretty)
				break
			}
			err = output.StreamText(stdout, prodCh, opts.Header, opts.Pretty)

		default:
			err = fmt.Errorf("unsupported output %q", opts.Output)
		}
		writeErr <- err
	}()

	// Worker goroutines
	var wg sync.WaitGroup
	wg.Add(thr)
	for w := 0; w < thr; w++ {
		go func() {
			defer wg.Done()
			for j := range jobs {
				hits := eng.Simulate(j.rec.ID, j.rec.Seq, j.pair)
				if needSeq {
					for i := range hits {
						hits[i].Seq = string(j.rec.Seq[hits[i].Start:hits[i].End])
					}
				}
				results <- result{prods: hits}
			}
		}()
	}

	// Forwarder/collector goroutine with cross-chunk de-duplication
	// Key is based on base SequenceID (record id), *global* start/end, type, and experiment id.
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
					continue // skip duplicate across overlapping chunks
				}
				seen[k] = struct{}{}
				prodCh <- p
				totalHits++
			}
		}
	}()

	// Feed jobs to workers
	for _, fa := range opts.SeqFiles {
		rch, err := fasta.StreamChunks(fa, opts.ChunkSize, overlap)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		for rec := range rch {
			for _, pr := range pairs {
				jobs <- job{rec: rec, pair: pr}
			}
		}
	}

	// Graceful close and shutdown
	close(jobs)
	wg.Wait()
	close(results)
	collectWg.Wait()
	close(prodCh)

	if err := <-writeErr; err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	if colErr != nil {
		fmt.Fprintln(stderr, colErr)
		return 3
	}
	if totalHits == 0 {
		return 1 // No amplicons found
	}
	return 0
}
