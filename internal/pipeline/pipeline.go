// internal/pipeline/pipeline.go
package pipeline

import (
	"context"
	"sync"

	"ipcr/internal/common"
	"ipcr/internal/engine"
	"ipcr/internal/fasta"
	"ipcr/internal/primer"
)

// Config controls the scanning pipeline.
type Config struct {
	Threads   int  // number of worker goroutines (>=1)
	ChunkSize int  // FASTA chunking window; 0 disables chunking
	Overlap   int  // overlap between chunks (typically >= MaxLen or primerLen-1)
	Circular  bool // treat sequences as circular
	NeedSeq   bool // fill Product.Seq by slicing record sequence
}

// Key uniquely identifies a product in reference-global coordinates to
// deduplicate cross-chunk duplicates.
type Key struct {
	Base, File string
	Start, End int
	Type, Exp  string
}

// ForEachProduct streams deduplicated engine.Products to the caller via visit.
// It reads chunks from seqFiles, runs SimulateBatch over all primer pairs, fills
// Product.Seq if requested, normalizes IDs/coords, deduplicates, and calls visit.
// It returns the first error encountered (including context cancellation).
func ForEachProduct(
	ctx context.Context,
	cfg Config,
	seqFiles []string,
	pairs []primer.Pair,
	eng *engine.Engine,
	visit func(engine.Product) error,
) error {
	if cfg.Threads < 1 {
		cfg.Threads = 1
	}

	type job struct {
		rec        fasta.Record
		sourceFile string
	}
	jobs := make(chan job, cfg.Threads*2)
	results := make(chan []engine.Product, cfg.Threads*2)

	// Workers
	var wg sync.WaitGroup
	wg.Add(cfg.Threads)
	for w := 0; w < cfg.Threads; w++ {
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
					hits := eng.SimulateBatch(j.rec.ID, j.rec.Seq, pairs)

					// Fill sequence and source file
					if cfg.NeedSeq {
						for i := range hits {
							if cfg.Circular && hits[i].Start > hits[i].End {
								seqBytes := j.rec.Seq
								hits[i].Seq = string(seqBytes[hits[i].Start:]) + string(seqBytes[:hits[i].End])
							} else {
								hits[i].Seq = string(j.rec.Seq[hits[i].Start:hits[i].End])
							}
						}
					}
					for i := range hits {
						hits[i].SourceFile = j.sourceFile
					}

					select {
					case results <- hits:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// Collector + deduper
	var (
		cerr error
		cwg  sync.WaitGroup
		seen = make(map[Key]struct{}, 1<<12)
	)
	cwg.Add(1)
	go func() {
		defer cwg.Done()
		for hs := range results {
			if cerr != nil {
				continue
			}
			for _, p := range hs {
				base, off, ok := common.SplitChunkSuffix(p.SequenceID)
				if !ok {
					base = p.SequenceID
					off = 0
				}
				gs, ge := p.Start+off, p.End+off
				k := Key{Base: base, File: p.SourceFile, Start: gs, End: ge, Type: p.Type, Exp: p.ExperimentID}
				if _, dup := seen[k]; dup {
					continue
				}
				seen[k] = struct{}{}
				if err := visit(p); err != nil && cerr == nil {
					cerr = err
				}
			}
		}
	}()

	// Feed work
feed:
	for _, fa := range seqFiles {
		rch, err := fasta.StreamChunks(fa, cfg.ChunkSize, cfg.Overlap)
		if err != nil {
			// Keep scanning other files; first error will be returned.
			if cerr == nil {
				cerr = err
			}
			continue
		}
		for rec := range rch {
			select {
			case <-ctx.Done():
				break feed
			case jobs <- job{rec: rec, sourceFile: fa}:
			}
		}
	}

	close(jobs)
	wg.Wait()
	close(results)
	cwg.Wait()

	if ctx.Err() != nil {
		return ctx.Err()
	}
	return cerr
}
