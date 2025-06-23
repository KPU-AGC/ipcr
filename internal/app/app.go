// internal/app/app.go
package app

import (
	"bytes"
	"fmt"
	"runtime"

	"ipcress-go/internal/cli"
	"ipcress-go/internal/engine"
	"ipcress-go/internal/fasta"
	"ipcress-go/internal/output"
	"ipcress-go/internal/primer"
)

/* -------------------------------------------------------------------------- */
/*                                 data types                                 */
/* -------------------------------------------------------------------------- */

type job struct {
	rec  fasta.Record
	pair primer.Pair
}

type result struct {
	products []engine.Product
	err      error
}

/* -------------------------------------------------------------------------- */
/*                                     Run                                    */
/* -------------------------------------------------------------------------- */

// Run executes the full in‑silico PCR pipeline.
//
// * argv    – command‑line arguments **without** argv[0]
// * stdout  – where normal output is written (capturable in tests)
// * stderr  – where diagnostics / errors are written
//
// It returns the process exit‑code described in the specification:
//   0 = ≥1 product, 1 = no product, 2 = bad CLI/config, 3 = runtime/I‑O error
func Run(argv []string, stdout, stderr *bytes.Buffer) int {
	/* --------------------------- parse CLI options -------------------------- */

	fs := cli.NewFlagSet("ipcress")
	opts, err := cli.ParseArgs(fs, argv)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	/* ----------------------------- load primers ----------------------------- */

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

	/* ------------------------------ PCR engine ------------------------------ */

	cfg := engine.Config{
		MaxMM:       opts.Mismatches,
		Disallow3MM: opts.Mode == cli.ModeRealistic,
		MinLen:      opts.MinLen,
		MaxLen:      opts.MaxLen,
	}
	eng := engine.New(cfg)

	/* --------------------------- concurrency setup -------------------------- */

	threads := opts.Threads
	if threads <= 0 {
		threads = runtime.NumCPU()
	}

	jobs := make(chan job, threads*2)
	results := make(chan result, threads*2)

	// worker goroutines
	for w := 0; w < threads; w++ {
		go func() {
			for j := range jobs {
				hits := eng.Simulate(j.rec.ID, j.rec.Seq, j.pair)

				if opts.Products {
					for i := range hits {
						hits[i].Seq = string(j.rec.Seq[hits[i].Start:hits[i].End])
					}
				}
				results <- result{products: hits}
			}
		}()
	}

	/* --------------------------- dispatch FASTA ----------------------------- */

	pending := 0
	for _, fa := range opts.SeqFiles {
		recCh, err := fasta.Stream(fa)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		for rec := range recCh {
			for _, pr := range pairs {
				pending++
				jobs <- job{rec: rec, pair: pr}
			}
		}
	}
	close(jobs)

	/* ---------------------------- aggregate hits ---------------------------- */

	var products []engine.Product
	for pending > 0 {
		r := <-results
		if r.err != nil {
			fmt.Fprintln(stderr, r.err)
			return 3
		}
		products = append(products, r.products...)
		pending--
	}

	if len(products) == 0 {
		return 1 // no hits
	}

	/* ------------------------------- output --------------------------------- */

	switch opts.Output {
	case "text":
		err = output.WriteText(stdout, products)
	case "json":
		err = output.WriteJSON(stdout, products)
	case "fasta":
		err = output.WriteFASTA(stdout, products)
	default:
		err = fmt.Errorf("unsupported output %q", opts.Output)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	return 0
}
