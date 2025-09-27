// internal/fasta/path_ctx.go
package fasta

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
)

// StreamChunksPathCtx opens `path`, scans FASTA, and emits overlapped chunks.
// Cancellation via ctx is honored promptly (both between lines and between chunks).
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
	defer rc.Close()

	sc := bufio.NewScanner(rc)
	const maxLine = 64 * 1024 * 1024 // allow very long single-line sequences (64 MiB)
	buf := make([]byte, 64*1024)
	sc.Buffer(buf, maxLine)

	var (
		id  string
		seq = make([]byte, 0, 1<<20)
	)

	flush := func() error {
		if id == "" {
			return nil
		}
		if chunkSize <= 0 || chunkSize >= len(seq) {
			if err := emit(Record{ID: id, Seq: append([]byte(nil), seq...)}); err != nil {
				return err
			}
			return nil
		}
		step := chunkSize - overlap
		if step <= 0 {
			if err := emit(Record{ID: id, Seq: append([]byte(nil), seq...)}); err != nil {
				return err
			}
			return nil
		}
		for off := 0; off < len(seq); off += step {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			end := off + chunkSize
			if end > len(seq) {
				end = len(seq)
			}
			chID := fmt.Sprintf("%s:%d-%d", id, off, end)
			if err := emit(Record{ID: chID, Seq: append([]byte(nil), seq[off:end]...)}); err != nil {
				return err
			}
			if end == len(seq) {
				break
			}
		}
		return nil
	}

	for sc.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		if line[0] == '>' {
			if id != "" {
				if err := flush(); err != nil {
					return err
				}
				seq = seq[:0]
			}
			id = parseHeaderID(line[1:])
			continue
		}
		seq = append(seq, bytes.TrimSpace(line)...)
	}
	if err := sc.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("fasta scan: %w", err)
	}
	if id != "" || len(seq) > 0 {
		if err := flush(); err != nil {
			return err
		}
	}
	return nil
}
