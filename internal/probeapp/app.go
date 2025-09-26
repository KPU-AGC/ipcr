// internal/probeapp/app.go
package probeapp

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

	"ipcr/internal/engine"
	"ipcr/internal/fasta"
	"ipcr/internal/primer"
	"ipcr/internal/probe"
	"ipcr/internal/probecli"
	"ipcr/internal/probeoutput"
	"ipcr/internal/version"
)

func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}

func RunContext(parent context.Context, argv []string, stdout, stderr io.Writer) int {
	outw := bufio.NewWriterSize(stdout, 64<<10)
	isBrokenPipe := func(err error) bool {
		return err != nil && (errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe))
	}

	fs := probecli.NewFlagSet("ipcr-probe")
	fs.SetOutput(io.Discard) // silence default flag pkg

	// No args => register flags then print usage
	if len(argv) == 0 {
		_, _ = probecli.ParseArgs(fs, []string{"-h"})
		fs.SetOutput(outw)
		fs.Usage()
		flushErr := outw.Flush()
		if isBrokenPipe(flushErr) {
			return 0
		} else if flushErr != nil {
			fmt.Fprintln(stderr, flushErr)
			return 3
		}
		return 0
	}

	opts, err := probecli.ParseArgs(fs, argv)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(outw)
			fs.Usage()
			flushErr := outw.Flush()
			if isBrokenPipe(flushErr) {
				return 0
			} else if flushErr != nil {
				fmt.Fprintln(stderr, flushErr)
				return 3
			}
			return 0
		}
		fmt.Fprintln(stderr, err)
		fs.SetOutput(outw)
		fs.Usage()
		flushErr := outw.Flush()
		if isBrokenPipe(flushErr) {
			return 0
		} else if flushErr != nil {
			fmt.Fprintln(stderr, flushErr)
			return 3
		}
		return 2
	}
	if opts.Version {
		fmt.Fprintf(outw, "ipcr-probe version %s\n", version.Version)
		flushErr := outw.Flush()
		if isBrokenPipe(flushErr) {
			return 0
		} else if flushErr != nil {
			fmt.Fprintln(stderr, flushErr)
			return 3
		}
		return 0
	}

	// Load primer pairs (reuse primer loader)
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

	// Max primer length (for chunk overlap)
	maxPLen := 0
	for _, pr := range pairs {
		if l := len(pr.Forward); l > maxPLen {
			maxPLen = l
		}
		if l := len(pr.Reverse); l > maxPLen {
			maxPLen = l
		}
	}

	// Terminal 3' window
	termWin := opts.TerminalWindow
	if termWin < 0 {
		if opts.Mode == probecli.ModeRealistic {
			termWin = 3
		} else {
			termWin = 0
		}
	}

	// Sanity
	if opts.MaxLen > 0 && opts.MaxLen < maxPLen {
		fmt.Fprintf(stderr, "error: --max-length (%d) is smaller than the longest primer (%d)\n", opts.MaxLen, maxPLen)
		return 2
	}
	if opts.MinLen > 0 && opts.MaxLen > 0 && opts.MinLen > opts.MaxLen {
		fmt.Fprintf(stderr, "error: --min-length (%d) exceeds --max-length (%d)\n", opts.MinLen, opts.MaxLen)
		return 2
	}

	// Chunking / overlap (disable if circular)
	overlap := 0
	if opts.Circular {
		if opts.ChunkSize != 0 && !opts.Quiet {
			fmt.Fprintln(stderr, "warning: --circular disables chunking; ignoring --chunk-size")
		}
		opts.ChunkSize = 0
	} else if opts.ChunkSize > 0 {
		if opts.MaxLen <= 0 {
			if !opts.Quiet {
				fmt.Fprintln(stderr, "warning: --chunk-size requires --max-length; disabling")
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

	// Engine
	eng := engine.New(engine.Config{
		MaxMM:          opts.Mismatches,
		TerminalWindow: termWin,
		MinLen:         opts.MinLen,
		MaxLen:         opts.MaxLen,
		HitCap:         opts.HitCap,
		NeedSites:      false, // pretty not used here
		SeedLen:        opts.SeedLength,
		Circular:       opts.Circular,
	})

	// Threads
	thr := opts.Threads
	if thr <= 0 {
		thr = runtime.NumCPU()
	}

	type job struct {
		rec        fasta.Record
		pairs      []primer.Pair
		sourceFile string
	}
	type result struct {
		prods []engine.Product
		err   error
	}

	jobs := make(chan job, thr*2)
	results := make(chan result, thr*2)
	annotCh := make(chan probeoutput.AnnotatedProduct, thr*4)

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	// Writer goroutine
	writeErr := make(chan error, 1)
	go func() {
		var werr error
		switch opts.Output {
		case "json":
			var buf []probeoutput.AnnotatedProduct
			for ap := range annotCh {
				buf = append(buf, ap)
			}
			if opts.Sort {
				sortAnnotated(buf)
			}
			werr = probeoutput.WriteJSON(outw, buf)
		case "fasta":
			if opts.Sort {
				var buf []probeoutput.AnnotatedProduct
				for ap := range annotCh {
					buf = append(buf, ap)
				}
				sortAnnotated(buf)
				werr = probeoutput.WriteFASTA(outw, buf)
				break
			}
			werr = probeoutput.StreamFASTA(outw, annotCh)
		case "text":
			if opts.Sort {
				var buf []probeoutput.AnnotatedProduct
				for ap := range annotCh {
					buf = append(buf, ap)
				}
				sortAnnotated(buf)
				werr = probeoutput.WriteText(outw, buf, opts.Header)
				break
			}
			werr = probeoutput.StreamText(outw, annotCh, opts.Header)
		default:
			werr = fmt.Errorf("unsupported output %q", opts.Output)
		}
		if isBrokenPipe(werr) {
			cancel()
		}
		writeErr <- werr
	}()

	// Workers: one scan per record across ALL pairs
	needSeq := true
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
							if opts.Circular && hits[i].Start > hits[i].End {
								seqBytes := j.rec.Seq
								hits[i].Seq = string(seqBytes[hits[i].Start:]) + string(seqBytes[:hits[i].End])
							} else {
								hits[i].Seq = string(j.rec.Seq[hits[i].Start:hits[i].End])
							}
						}
					}
					// Assign source file
					for i := range hits {
						hits[i].SourceFile = j.sourceFile
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

	// Collector: dedup across chunks then annotate probe
	type dkey struct {
		base, file string
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
				k := dkey{base: base, file: p.SourceFile, start: gs, end: ge, typ: p.Type, exp: p.ExperimentID}
				if _, dup := seen[k]; dup {
					continue
				}
				seen[k] = struct{}{}

				ann := probe.AnnotateAmplicon(p.Seq, opts.Probe, opts.ProbeMaxMM)
				if opts.RequireProbe && !ann.Found {
					continue
				}

				ap := probeoutput.AnnotatedProduct{
					Product:     p,
					ProbeName:   opts.ProbeName,
					ProbeSeq:    strings.ToUpper(opts.Probe),
					ProbeFound:  ann.Found,
					ProbeStrand: ann.Strand,
					ProbePos:    ann.Pos,
					ProbeMM:     ann.MM,
					ProbeSite:   ann.Site,
				}

				select {
				case annotCh <- ap:
					total++
				case <-ctx.Done():
				}
			}
		}
	}()

	// Feed jobs
feed:
	for _, fa := range opts.SeqFiles {
		rch, err := fasta.StreamChunks(fa, opts.ChunkSize, overlap)
		if err != nil {
			fmt.Fprintln(stderr, err)
			continue
		}
		for rec := range rch {
			if ctx.Err() != nil {
				break feed
			}
			select {
			case jobs <- job{rec: rec, pairs: pairs, sourceFile: fa}:
			case <-ctx.Done():
				break feed
			}
		}
	}
	close(jobs)
	wg.Wait()
	close(results)
	cwg.Wait()
	close(annotCh)

	werr := <-writeErr
	if isBrokenPipe(werr) {
		return 0
	}
	if werr != nil {
		fmt.Fprintln(stderr, werr)
		return 3
	}
	flushErr := outw.Flush()
	if isBrokenPipe(flushErr) {
		return 0
	} else if flushErr != nil {
		fmt.Fprintln(stderr, flushErr)
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

func sortAnnotated(a []probeoutput.AnnotatedProduct) {
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

// splitChunkSuffix parses ids that may end with ":<start>-<end>"
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
