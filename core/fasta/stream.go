// core/fasta/stream.go
package fasta

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
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
// It is **cancelable**: returning promptly when ctx is Done, even mid-record.
func StreamChunksCtx(ctx context.Context, r io.Reader, chunkSize int, circular bool, emit func(RecordChunk) error) error {
	sc := bufio.NewScanner(r)
	const maxLine = 64 * 1024 * 1024 // allow very long single-line sequences (64 MiB)
	buf := make([]byte, 64*1024)
	sc.Buffer(buf, maxLine)

	var (
		id   string
		seq  = make([]byte, 0, 1<<20)
		line []byte
	)

	flush := func(last bool) error {
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

	for sc.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line = sc.Bytes()
		if len(line) == 0 {
			continue
		}
		if line[0] == '>' {
			if id != "" {
				if err := flush(true); err != nil {
					return err
				}
				seq = seq[:0]
			}
			id = parseHeaderID(line[1:])
			continue
		}
		line = bytes.TrimSpace(line)
		seq = append(seq, line...)
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("fasta scan: %w", err)
	}
	if id != "" || len(seq) > 0 {
		if err := flush(true); err != nil {
			return err
		}
	}
	return nil
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
