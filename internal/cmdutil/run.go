package cmdutil

import (
	"context"

	"ipcr-core/engine"
	"ipcr/internal/pipeline"
	"ipcr-core/primer"
)

// RunStream runs the shared pipeline, applies a visitor, and streams results via send.
// NOTE: now takes a pipeline.Simulator (not *engine.Engine).
func RunStream[T any](
	ctx context.Context,
	cfg pipeline.Config,
	seqFiles []string,
	pairs []primer.Pair,
	sim pipeline.Simulator,
	visit func(engine.Product) (bool, T, error),
	send func(T) error,
) (int, error) {
	total := 0
	err := pipeline.ForEachProduct(ctx, cfg, seqFiles, pairs, sim, func(p engine.Product) error {
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
