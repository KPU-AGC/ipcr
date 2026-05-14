// core/fasta/stream.go
package fasta

import (
	"bytes"
	"context"
	"io"
)

// RecordChunk is a window of a single FASTA record's sequence.
// Offset is 0-based within the *linear* sequence (pre-wrap).
type RecordChunk struct {
	RecordID string
	Offset   int
	Seq      []byte
	IsWrap   bool
	IsLast   bool
}

// StreamChunksCtx parses FASTA from r and emits per-record sequence chunks.
// If chunkSize <= 0, each record is emitted as a single chunk.
// If circular = true and chunkSize > 0, a final wrap chunk is emitted.
//
// It is cancelable and does not use bufio.Scanner, so long single-line FASTA
// records are not limited by Scanner's token size.
func StreamChunksCtx(ctx context.Context, r io.Reader, chunkSize int, circular bool, emit func(RecordChunk) error) error {
	var (
		id  string
		seq = make([]byte, 0, 1<<20)
	)

	flush := func() error {
		if len(seq) == 0 && id == "" {
			return nil
		}
		if chunkSize <= 0 || chunkSize >= len(seq) {
			if err := emit(RecordChunk{RecordID: id, Offset: 0, Seq: append([]byte(nil), seq...), IsWrap: false, IsLast: true}); err != nil {
				return err
			}
			return nil
		}
		for off := 0; off < len(seq); off += chunkSize {
			end := off + chunkSize
			if end > len(seq) {
				end = len(seq)
			}
			ch := RecordChunk{
				RecordID: id,
				Offset:   off,
				Seq:      append([]byte(nil), seq[off:end]...),
				IsWrap:   false,
				IsLast:   false,
			}
			if end == len(seq) && !(circular && len(seq) > 0) {
				ch.IsLast = true
			}
			if err := emit(ch); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		if circular && len(seq) > 0 {
			if chunkSize > len(seq) {
				repeat := chunkSize - len(seq)
				wrap := append(append([]byte(nil), seq...), seq[:min(repeat, len(seq))]...)
				if err := emit(RecordChunk{
					RecordID: id,
					Offset:   0,
					Seq:      wrap,
					IsWrap:   true,
					IsLast:   true,
				}); err != nil {
					return err
				}
			} else {
				start := len(seq) - chunkSize
				wrap := append(append([]byte(nil), seq[start:]...), seq[:chunkSize-(len(seq)-start)]...)
				if err := emit(RecordChunk{
					RecordID: id,
					Offset:   start,
					Seq:      wrap,
					IsWrap:   true,
					IsLast:   true,
				}); err != nil {
					return err
				}
			}
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

// Convenience wrapper for reader-based streaming with background context.
func StreamChunksFromReader(r io.Reader, chunkSize int, circular bool, emit func(RecordChunk) error) error {
	return StreamChunksCtx(context.Background(), r, chunkSize, circular, emit)
}

func parseHeaderID(hdr []byte) string {
	hdr = bytes.TrimSpace(hdr)
	if i := bytes.IndexAny(hdr, " \t"); i >= 0 {
		return string(hdr[:i])
	}
	return string(hdr)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
