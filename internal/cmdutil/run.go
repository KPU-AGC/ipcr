package cmdutil

import (
	"context"

	"ipcr/internal/engine"
	"ipcr/internal/pipeline"
	"ipcr/internal/primer"
)

// RunStream runs the shared pipeline, applies a visitor, and streams results via send.
// It returns the number of kept outputs and the first error encountered.
func RunStream[T any](
	ctx context.Context,
	cfg pipeline.Config,
	seqFiles []string,
	pairs []primer.Pair,
	eng *engine.Engine,
	visit func(engine.Product) (bool, T, error),
	send func(T) error,
) (int, error) {
	total := 0
	err := pipeline.ForEachProduct(ctx, cfg, seqFiles, pairs, eng, func(p engine.Product) error {
		keep, out, vErr := visit(p)
		if vErr != nil {
			return vErr
		}
		if !keep {
			return nil
		}
		if err := send(out); err != nil {
			return err
		}
		total++
		return nil
	})
	return total, err
}
