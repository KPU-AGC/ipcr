// core/fasta/reader.go
package fasta

import (
	"context"
)

// Record represents a parsed FASTA sequence (or a chunk of one).
type Record struct {
	ID  string
	Seq []byte
}

// StreamChunksCtxPath is the ctx-aware channel wrapper around StreamChunksPathCtx.
// Semantics preserved:
//   - gzip and "-" for stdin are handled the same way (early open error for non-stdin)
//   - channel-based API
//   - scan-time errors are not propagated (same as legacy behavior)
func StreamChunksCtxPath(ctx context.Context, path string, chunkSize, overlap int) (<-chan Record, error) {
	// Preserve immediate error reporting for non-stdin paths.
	if path != "-" {
		rc, err := openReader(path)
		if err != nil {
			return nil, err
		}
		_ = rc.Close()
	}

	out := make(chan Record, 8)
	go func() {
		defer close(out)
		_ = StreamChunksPathCtx(
			ctx,
			path,
			chunkSize,
			overlap,
			func(r Record) error {
				out <- r
				return nil
			},
		)
	}()
	return out, nil
}

// StreamChunks remains as the legacy helper that uses a background context.
func StreamChunks(path string, chunkSize, overlap int) (<-chan Record, error) {
	return StreamChunksCtxPath(context.Background(), path, chunkSize, overlap)
}
