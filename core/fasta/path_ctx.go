// core/fasta/path_ctx.go
package fasta

import (
	"context"
	"fmt"
)

// StreamChunksPathCtx opens path, scans FASTA, and emits overlapped chunks.
// With chunking enabled, sequence is emitted with a rolling window and the full
// FASTA record is not buffered. With chunking disabled, the full record is
// emitted as one Record and is therefore buffered until its next header/EOF.
// Cancellation via ctx is honored between input fragments and between chunks.
//
// chunkSize <= 0  → whole record as one chunk (no overlap considered)
// overlap < 0     → treated as 0
//
// emit is called for each chunk. Return a non-nil error (e.g., ctx.Err()) to stop early.
func StreamChunksPathCtx(
	ctx context.Context,
	path string,
	chunkSize, overlap int,
	emit func(Record) error,
) error {
	if overlap < 0 {
		overlap = 0
	}
	rc, err := openReader(path)
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()

	if chunkSize <= 0 || overlap >= chunkSize {
		return streamWholeRecords(ctx, rc, emit)
	}
	return streamRollingChunks(ctx, rc, chunkSize, overlap, emit)
}

func streamWholeRecords(ctx context.Context, r interface {
	Read([]byte) (int, error)
}, emit func(Record) error) error {
	var (
		id  string
		seq = make([]byte, 0, 1<<20)
	)

	flush := func() error {
		if id == "" {
			return nil
		}
		if err := emit(Record{ID: id, Seq: append([]byte(nil), seq...)}); err != nil {
			return err
		}
		return nil
	}

	err := scanFASTALines(ctx, r,
		func(header []byte) error {
			if err := flush(); err != nil {
				return err
			}
			id = parseHeaderID(header)
			seq = seq[:0]
			return nil
		},
		func(line []byte) error {
			if id == "" {
				return nil
			}
			seq = appendNormalizedSeqLine(seq, line)
			return nil
		},
	)
	if err != nil {
		return err
	}
	return flush()
}

func streamRollingChunks(ctx context.Context, r interface {
	Read([]byte) (int, error)
}, chunkSize, overlap int, emit func(Record) error) error {
	step := chunkSize - overlap
	if step <= 0 {
		return streamWholeRecords(ctx, r, emit)
	}

	var (
		id             string
		window         = make([]byte, 0, chunkSize+1)
		windowStart    int
		totalLen       int
		lastEmittedEnd int
		emittedChunk   bool
	)

	reset := func(newID string) {
		id = newID
		window = window[:0]
		windowStart = 0
		totalLen = 0
		lastEmittedEnd = 0
		emittedChunk = false
	}

	emitChunk := func(start, end int, seq []byte) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		chID := fmt.Sprintf("%s:%d-%d", id, start, end)
		if err := emit(Record{ID: chID, Seq: append([]byte(nil), seq...)}); err != nil {
			return err
		}
		lastEmittedEnd = end
		emittedChunk = true
		return nil
	}

	flush := func() error {
		if id == "" {
			return nil
		}
		if !emittedChunk {
			if err := emit(Record{ID: id, Seq: append([]byte(nil), window...)}); err != nil {
				return err
			}
			return nil
		}
		if lastEmittedEnd < totalLen {
			return emitChunk(windowStart, totalLen, window)
		}
		return nil
	}

	appendSeq := func(line []byte) error {
		if id == "" {
			return nil
		}
		before := len(window)
		window = appendNormalizedSeqLine(window, line)
		totalLen += len(window) - before

		for len(window) > chunkSize {
			if err := emitChunk(windowStart, windowStart+chunkSize, window[:chunkSize]); err != nil {
				return err
			}
			if step >= len(window) {
				window = window[:0]
			} else {
				copy(window, window[step:])
				window = window[:len(window)-step]
			}
			windowStart += step
		}
		return nil
	}

	err := scanFASTALines(ctx, r,
		func(header []byte) error {
			if err := flush(); err != nil {
				return err
			}
			reset(parseHeaderID(header))
			return nil
		},
		appendSeq,
	)
	if err != nil {
		return err
	}
	return flush()
}
