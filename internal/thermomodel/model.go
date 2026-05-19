package thermomodel

import (
	"fmt"
	"strings"
)

// Mode identifies the scoring model used by ipcr-thermo. The explicit mode is
// intentionally separate from lower-level knobs such as --denom so future
// thermodynamic implementations can be introduced without changing legacy
// behavior by accident.
type Mode string

const (
	// LegacyHeuristic is the current shipped behavior: heuristic primer-template
	// mismatch scoring with the existing fixed/auto denominator switch.
	LegacyHeuristic Mode = "legacy-heuristic"

	// NNDuplexV1 is the nearest-neighbor primer-template duplex implementation.
	// It computes condition-aware perfect-duplex thermodynamics and applies the
	// explicit mismatch fallback policy when the target site is imperfect.
	NNDuplexV1 Mode = "nn-duplex-v1"

	// NNStructureV1 adds nearest-neighbor secondary-structure competition terms:
	// primer hairpins, self-dimers, and forward/reverse cross-dimers.
	NNStructureV1 Mode = "nn-structure-v1"
)

// Default returns the release-default thermodynamic model.
func Default() Mode { return NNStructureV1 }

// Parse validates and normalizes a model name.
func Parse(raw string) (Mode, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return Default(), nil
	}
	switch Mode(s) {
	case LegacyHeuristic, NNDuplexV1, NNStructureV1:
		return Mode(s), nil
	default:
		return "", fmt.Errorf("unknown thermo model %q; expected one of: %s", raw, KnownList())
	}
}

// Known returns all reserved mode names in rollout order.
func Known() []Mode {
	return []Mode{LegacyHeuristic, NNDuplexV1, NNStructureV1}
}

// KnownList returns all reserved mode names as CLI help text.
func KnownList() string {
	modes := Known()
	parts := make([]string, 0, len(modes))
	for _, mode := range modes {
		parts = append(parts, mode.String())
	}
	return strings.Join(parts, " | ")
}

func (m Mode) String() string { return string(m) }

// Implemented reports whether the mode is executable in this patch.
func (m Mode) Implemented() bool {
	return m == LegacyHeuristic || m == NNDuplexV1 || m == NNStructureV1
}
