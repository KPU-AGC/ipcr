// internal/common/primers.go
package common

import (
	"strings"

	"ipcr-core/primer"
)

// AddSelfPairs appends per-row A:self and B:self pairs (Forward == Reverse).
// Sequences are uppercased; Min/Max product bounds fall back to engine/global cfg.
func AddSelfPairs(pairs []primer.Pair) []primer.Pair {
	out := make([]primer.Pair, 0, len(pairs)+2*len(pairs))
	out = append(out, pairs...)
	for _, p := range pairs {
		if p.Forward != "" {
			u := strings.ToUpper(p.Forward)
			out = append(out, primer.Pair{
				ID:         p.ID + "+A:self",
				Forward:    u,
				Reverse:    u,
				MinProduct: 0,
				MaxProduct: 0,
			})
		}
		if p.Reverse != "" {
			u := strings.ToUpper(p.Reverse)
			out = append(out, primer.Pair{
				ID:         p.ID + "+B:self",
				Forward:    u,
				Reverse:    u,
				MinProduct: 0,
				MaxProduct: 0,
			})
		}
	}
	return out
}

// AddSelfPairsUnique appends A:self/B:self ONCE per unique forward/reverse sequence
// across the whole set, preserving first-seen order. Original pairs are kept as-is.
func AddSelfPairsUnique(pairs []primer.Pair) []primer.Pair {
	out := make([]primer.Pair, 0, len(pairs)+2*len(pairs))
	out = append(out, pairs...)

	seenA := make(map[string]struct{}, len(pairs))
	seenB := make(map[string]struct{}, len(pairs))
	for _, p := range pairs {
		if f := strings.ToUpper(strings.TrimSpace(p.Forward)); f != "" {
			if _, ok := seenA[f]; !ok {
				seenA[f] = struct{}{}
				out = append(out, primer.Pair{
					ID:         p.ID + "+A:self",
					Forward:    f,
					Reverse:    f,
					MinProduct: 0,
					MaxProduct: 0,
				})
			}
		}
		if r := strings.ToUpper(strings.TrimSpace(p.Reverse)); r != "" {
			if _, ok := seenB[r]; !ok {
				seenB[r] = struct{}{}
				out = append(out, primer.Pair{
					ID:         p.ID + "+B:self",
					Forward:    r,
					Reverse:    r,
					MinProduct: 0,
					MaxProduct: 0,
				})
			}
		}
	}
	return out
}
