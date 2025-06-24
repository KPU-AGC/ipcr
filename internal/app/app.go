// internal/app/app.go
package app

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"

	"ipcr/internal/cli"
	"ipcr/internal/engine"
	"ipcr/internal/fasta"
	"ipcr/internal/output"
	"ipcr/internal/primer"
)

/* -------------------------------------------------------------------------- */
/*                                 job model                                  */
/* -------------------------------------------------------------------------- */

type job struct {
	rec  fasta.Record
	pair primer.Pair
}
type result struct {
	prods []engine.Product
	err   error
}

/* -------------------------------------------------------------------------- */
/*                                     Run                                    */
/* -------------------------------------------------------------------------- */

// Run executes the full pipeline and returns the intended process exit code.
func Run(argv []string, stdout, stderr *bytes.Buffer) int {
	/* --------------------------- parse CLI options -------------------------- */

	fs := cli.NewFlagSet("ipcress")
	opts, err := cli.ParseArgs(fs, argv)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	/* --------------------------- load primer pairs -------------------------- */

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

	/* ------------------------ calculate chunk overlap ----------------------- */

	maxPLen := 0
	for _, pr := range pairs {
		if l := len(pr.Forward); l > maxPLen {
			maxPLen = l
		}
		if l := len(pr.Reverse); l > maxPLen {
			maxPLen = l
		}
	}
	overlap := maxPLen - 1
	if opts.ChunkSize == 0 {
		overlap = 0
	}

	/* ------------------------------ PCR engine ------------------------------ */

	eng := engine.New(engine.Config{
		MaxMM:       opts.Mismatches,
		Disallow3MM: opts.Mode == cli.ModeRealistic,
		MinLen:      opts.MinLen,
		MaxLen:      opts.MaxLen,
	})
	eng.SetHitCap(opts.HitCap)

	/* ---------------------------- worker set‑up ----------------------------- */

	thr := opts.Threads
	if thr <= 0 {
		thr = runtime.NumCPU()
	}

	jobs    := make(chan job, thr*2)
	results := make(chan result, thr*2)
	prodCh  := make(chan engine.Product, thr*4) // to writer

	/* ------------------------------ writer goroutine ------------------------ */

	writeErr := make(chan error, 1)
	go func() {
		var err error
		switch opts.Output {
		case "text":
			err = output.StreamText(stdout, prodCh)
		case "fasta":
			err = output.StreamFASTA(stdout, prodCh)
		case "json":
			// JSON requires whole slice to close array
			var buf []engine.Product
			for p := range prodCh {
				buf = append(buf, p)
			}
			err = output.WriteJSON(stdout, buf)
		default:
			err = fmt.Errorf("unsupported output %q", opts.Output)
		}
		writeErr <- err
	}()

	/* ------------------------------ worker pool ----------------------------- */

	var wg sync.WaitGroup
	wg.Add(thr)
	for w := 0; w < thr; w++ {
		go func() {
			defer wg.Done()
			for j := range jobs {
				hits := eng.Simulate(j.rec.ID, j.rec.Seq, j.pair)
				if opts.Products {
					for i := range hits {
						hits[i].Seq =
							string(j.rec.Seq[hits[i].Start:hits[i].End])
					}
				}
				results <- result{prods: hits}
			}
		}()
	}

	/* --------------------------- forwarder goroutine ------------------------ */

	var (
		colErr     error
		totalHits  int
		collectWg  sync.WaitGroup
	)
	collectWg.Add(1)
	go func() {
		defer collectWg.Done()
		for r := range results {
			if r.err != nil && colErr == nil {
				colErr = r.err
			}
			for _, p := range r.prods {
				prodCh <- p
				totalHits++
			}
		}
	}()

	/* ------------------------------- feed jobs ------------------------------ */

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

	/* ------------------------------ graceful close -------------------------- */

	close(jobs)      // no more jobs
	wg.Wait()        // wait workers
	close(results)   // stop collector
	collectWg.Wait() // wait collector
	close(prodCh)    // signal writer

	if err := <-writeErr; err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	if colErr != nil {
		fmt.Fprintln(stderr, colErr)
		return 3
	}
	if totalHits == 0 {
		return 1 // no amplicons found
	}
	return 0
}
