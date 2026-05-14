package fasta

import (
	"bufio"
	"context"
	"fmt"
	"io"
)

func scanFASTALines(
	ctx context.Context,
	r io.Reader,
	onHeader func([]byte) error,
	onSeq func([]byte) error,
) error {
	br := bufio.NewReaderSize(r, 1024*1024)
	atLineStart := true

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		frag, err := br.ReadSlice('\n')
		if len(frag) > 0 {
			if atLineStart && frag[0] == '>' {
				header := append([]byte(nil), frag[1:]...)
				for err == bufio.ErrBufferFull {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					frag, err = br.ReadSlice('\n')
					header = append(header, frag...)
				}
				if err != nil && err != io.EOF {
					return fmt.Errorf("fasta scan: %w", err)
				}
				if onHeader != nil {
					if e := onHeader(header); e != nil {
						return e
					}
				}
				atLineStart = true
			} else {
				if onSeq != nil {
					if e := onSeq(frag); e != nil {
						return e
					}
				}
				atLineStart = err != bufio.ErrBufferFull
			}
		}

		switch err {
		case nil:
			continue
		case bufio.ErrBufferFull:
			continue
		case io.EOF:
			return nil
		default:
			return fmt.Errorf("fasta scan: %w", err)
		}
	}
}
