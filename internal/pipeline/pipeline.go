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
	results := make(chan engine.Product, cfg.Threads*2)

	compiledSim, useCompiled := sim.(CompiledSimulator)
	scratchCompiledSim, useScratchCompiled := sim.(ScratchCompiledSimulator)
	streamingSim, useStreaming := sim.(StreamingCompiledSimulator)
	var compiledPanel *engine.CompiledPanel
	if useCompiled {
		compiledPanel = compiledSim.CompilePanel(pairs)
	}

	// Workers
	var wg sync.WaitGroup
	wg.Add(cfg.Threads)
	for w := 0; w < cfg.Threads; w++ {
		go func() {
			defer wg.Done()
			var scratch *engine.SimulationScratch
			if useScratchCompiled {
				scratch = scratchCompiledSim.NewSimulationScratch(compiledPanel)
			}

			for {
				select {
				case <-ctx.Done():
					return
				case j, ok := <-jobs:
					if !ok {
						return
					}

					sendProduct := func(p engine.Product) error {
						if cfg.NeedSeq {
							if cfg.Circular && p.Start > p.End {
								seqBytes := j.rec.Seq
								p.Seq = string(seqBytes[p.Start:]) + string(seqBytes[:p.End])
							} else {
								p.Seq = string(j.rec.Seq[p.Start:p.End])
							}
						}
						p.SourceFile = j.sourceFile
						select {
						case results <- p:
							return nil
						case <-ctx.Done():
							return ctx.Err()
						}
					}

					switch {
					case useStreaming:
						if err := streamingSim.ForEachCompiledProduct(j.rec.ID, j.rec.Seq, compiledPanel, scratch, sendProduct); err != nil {
							return
						}
					case useScratchCompiled:
						for _, p := range scratchCompiledSim.SimulateCompiledWithScratch(j.rec.ID, j.rec.Seq, compiledPanel, scratch) {
							if err := sendProduct(p); err != nil {
								return
							}
						}
					case useCompiled:
						for _, p := range compiledSim.SimulateCompiled(j.rec.ID, j.rec.Seq, compiledPanel) {
							if err := sendProduct(p); err != nil {
								return
							}
						}
					default:
						for _, p := range sim.SimulateBatch(j.rec.ID, j.rec.Seq, pairs) {
							if err := sendProduct(p); err != nil {
								return
							}
						}
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
		for p := range results {
			if cerr != nil {
				continue
			}
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
			// Present chunked results in reference-global coordinates so --chunk-size does not
			// change the external coordinate system or sorted order.
			if ok {
				p.SequenceID = base
				p.Start = gs
				p.End = ge
			}
			if err := visit(p); err != nil && cerr == nil {
				cerr = err
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
