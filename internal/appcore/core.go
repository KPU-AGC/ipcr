// internal/appcore/core.go
package appcore

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"

	"ipcr/internal/cmdutil"
	"ipcr/internal/engine"
	"ipcr/internal/pipeline"
	"ipcr/internal/primer"
	"ipcr/internal/runutil"
	"ipcr/internal/writers"
)

type Options struct {
	SeqFiles []string

	MaxMM          int
	TerminalWindow int
	MinLen         int
	MaxLen         int
	HitCap         int
	SeedLength     int
	Circular       bool

	Threads   int
	ChunkSize int

	Quiet           bool
	NoMatchExitCode int
}

type VisitorFunc[T any] func(engine.Product) (keep bool, out T, err error)

type WriterFactory[T any] interface {
	NeedSites() bool
	NeedSeq() bool
	Start(out io.Writer, bufSize int) (chan<- T, <-chan error)
}

func Run[T any](
	parent context.Context,
	stdout, stderr io.Writer,
	o Options,
	pairs []primer.Pair,
	visit VisitorFunc[T],
	wf WriterFactory[T],
) int {
	outw := bufio.NewWriter(stdout)

	// longest primer
	maxPLen := 0
	for _, pr := range pairs {
		if l := len(pr.Forward); l > maxPLen {
			maxPLen = l
		}
		if l := len(pr.Reverse); l > maxPLen {
			maxPLen = l
		}
	}
	if o.MaxLen > 0 && o.MaxLen < maxPLen {
		fmt.Fprintf(stderr, "error: --max-length (%d) is smaller than the longest primer length (%d)\n", o.MaxLen, maxPLen)
		return 2
	}
	if o.MinLen > 0 && o.MaxLen > 0 && o.MinLen > o.MaxLen {
		fmt.Fprintf(stderr, "error: --min-length (%d) exceeds --max-length (%d)\n", o.MinLen, o.MaxLen)
		return 2
	}

	chunkSize, overlap, warns := runutil.ValidateChunking(o.Circular, o.ChunkSize, o.MaxLen, maxPLen)
	if !o.Quiet {
		for _, w := range warns {
			fmt.Fprintln(stderr, w)
		}
	}

	thr := o.Threads
	if thr <= 0 {
		thr = runtime.NumCPU()
	}

	sim := engine.New(engine.Config{
		MaxMM:          o.MaxMM,
		TerminalWindow: o.TerminalWindow,
		MinLen:         o.MinLen,
		MaxLen:         o.MaxLen,
		HitCap:         o.HitCap,
		NeedSites:      wf.NeedSites(),
		SeedLen:        o.SeedLength,
		Circular:       o.Circular,
	})

	inCh, writeErr := wf.Start(outw, thr*4)

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	total, perr := cmdutil.RunStream[T](
		ctx,
		pipeline.Config{
			Threads:   thr,
			ChunkSize: chunkSize,
			Overlap:   overlap,
			Circular:  o.Circular,
			NeedSeq:   wf.NeedSeq(),
		},
		o.SeqFiles,
		pairs,
		sim, // <— interface now
		visit,
		func(x T) error {
			select {
			case inCh <- x:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	)

	close(inCh)

	if werr := <-writeErr; writers.IsBrokenPipe(werr) {
		return 0
	} else if werr != nil {
		fmt.Fprintln(stderr, werr)
		return 3
	}
	if e := outw.Flush(); writers.IsBrokenPipe(e) {
		return 0
	} else if e != nil {
		fmt.Fprintln(stderr, e)
		return 3
	}

	if perr != nil {
		if errors.Is(perr, context.Canceled) {
			return 130
		}
		fmt.Fprintln(stderr, perr)
		return 3
	}
	if total == 0 {
		return o.NoMatchExitCode
	}
	return 0
}
