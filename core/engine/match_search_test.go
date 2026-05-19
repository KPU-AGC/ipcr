package engine

import (
	"testing"

	"ipcr-core/primer"
)

func TestSortMatchesByPosAndBounds(t *testing.T) {
	matches := []primer.Match{{Pos: 8}, {Pos: 2}, {Pos: 5}, {Pos: 5}}
	matches = sortMatchesByPos(matches)

	positions := make([]int, len(matches))
	for i, m := range matches {
		positions[i] = m.Pos
	}
	want := []int{2, 5, 5, 8}
	for i := range want {
		if positions[i] != want[i] {
			t.Fatalf("sorted positions = %v, want %v", positions, want)
		}
	}

	lo := lowerBoundMatchPos(matches, 5)
	hi := upperBoundMatchPos(matches, 5)
	if lo != 1 || hi != 3 {
		t.Fatalf("bounds for 5 = [%d,%d), want [1,3)", lo, hi)
	}
}
