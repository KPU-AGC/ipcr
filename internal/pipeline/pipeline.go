// internal/pipeline/pipeline.go
package pipeline

import (
	"context"
	"ipcr-core/engine"
	"ipcr-core/fasta"
	"ipcr-core/primer"
	"ipcr/internal/common"
	"ipcr/internal/runutil"
	"sync"
)

// Config controls the scanning pipeline.
type Config struct {
	Threads   int  // number of worker goroutines (>=1)
	ChunkSize int  // FASTA chunking window; 0 disables chunking
	Overlap   int  // overlap between chunks (typically >= MaxLen or primerLen-1)
	Circular  bool // treat sequences as circular
	NeedSeq   bool // fill Product.Seq by slicing record sequence
	DedupCap  int  // NEW: capacity for LRU de-dup window (0=default)
}

// Key uniquely identifies a product in reference-global coordinates to
// deduplicate cross-chunk duplicates.
type Key struct {
	Base, File string
	Start, End int
	Type, Exp  string
}

// ForEachProduct ...
func ForEachProduct(
	ctx context.Context,
	cfg Config,
	seqFiles []string,
	pairs []primer.Pair,
	sim Simulator,
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
					hits := sim.SimulateBatch(j.rec.ID, j.rec.Seq, pairs)

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

	// Collector + bounded deduper (LRU)
	var (
		cerr error
		cwg  sync.WaitGroup
		seen = runutil.NewLRUSet[Key](cfg.DedupCap) // NEW (defaulting inside NewLRUSet if <=0)
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
				if seen.Add(k) { // already seen recently
					continue
				}
				if err := visit(p); err != nil && cerr == nil {
					cerr = err
				}
			}
		}
	}()

	// Feed work
feed:
	for _, fa := range seqFiles {
		err := fasta.StreamChunksPathCtx(ctx, fa, cfg.ChunkSize, cfg.Overlap, func(rec fasta.Record) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case jobs <- job{rec: rec, sourceFile: fa}:
				return nil
			}
		})
		if err != nil {
			if ctx.Err() != nil {
				break feed
			}
			if cerr == nil {
				cerr = err
			}
			continue
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
