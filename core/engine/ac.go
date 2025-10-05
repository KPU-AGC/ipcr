package engine

import (
	"ipcr-core/primer"
)

/*
Aho–Corasick seed scanner + local verifier.

- buildAC(seeds) builds a trie with failure links.
- scanAC(seq, nodes, seeds) emits (endPos, seedIdx) for every seed occurrence.
- verifyAt(seq, pos, pat, ...) re-checks the full primer at 'pos' (0-based),
  counting mismatches with IUPAC and enforcing left/right terminal windows.
*/

// node is one state in the automaton.
type node struct {
	next [256]int // 0 => absent (root is state 0)
	fail int
	out  []int // seed indexes that end at this state
}

// buildAC constructs the automaton for all seeds' byte patterns.
func buildAC(seeds []Seed) ([]node, int) {
	nodes := make([]node, 1) // state 0 = root

	// 1) Build trie edges
	for i, s := range seeds {
		cur := 0
		for _, b := range s.Pat {
			if nodes[cur].next[b] == 0 {
				nodes = append(nodes, node{})
				nodes[cur].next[b] = len(nodes) - 1
			}
			cur = nodes[cur].next[b]
		}
		nodes[cur].out = append(nodes[cur].out, i)
	}

	// 2) BFS to set fail links and propagate outputs
	queue := make([]int, 0, len(nodes))
	// root’s immediate children fail to root
	for c := 0; c < 256; c++ {
		if child := nodes[0].next[c]; child != 0 {
			nodes[child].fail = 0
			queue = append(queue, child)
		}
	}
	for len(queue) > 0 {
		r := queue[0]
		queue = queue[1:]
		for c := 0; c < 256; c++ {
			s := nodes[r].next[c]
			if s == 0 {
				continue
			}
			queue = append(queue, s)
			// find failure target
			f := nodes[r].fail
			for f > 0 && nodes[f].next[c] == 0 {
				f = nodes[f].fail
			}
			if nodes[f].next[c] != 0 {
				f = nodes[f].next[c]
			}
			nodes[s].fail = f
			if len(nodes[f].out) > 0 {
				nodes[s].out = append(nodes[s].out, nodes[f].out...)
			}
		}
	}
	return nodes, len(seeds)
}

// hit = “pattern ends at Pos” with which seed.
type hit struct {
	Pos     int // position (end index of the matched seed) in seq
	SeedIdx int
}

// scanAC runs the automaton over seq and collects hits.
func scanAC(seq []byte, nodes []node, seeds []Seed) []hit {
	state := 0
	out := make([]hit, 0, 128)
	for i := 0; i < len(seq); i++ {
		b := seq[i]
		for state > 0 && nodes[state].next[b] == 0 {
			state = nodes[state].fail
		}
		if next := nodes[state].next[b]; next != 0 {
			state = next
		}
		if ids := nodes[state].out; len(ids) > 0 {
			for _, idx := range ids {
				out = append(out, hit{Pos: i, SeedIdx: idx})
			}
		}
	}
	return out
}

// verifyAt checks a full primer pattern at pos using IUPAC matching, with:
//   - maxMM      : mismatch budget (≤0 means “no mismatches allowed”)
//   - leftTW     : protected window at 5' end (for rc orientations)
//   - rightTW    : protected window at 3' end (for forward orientations)
func verifyAt(seq []byte, pos int, pat []byte, maxMM int, leftTW, rightTW int) (primer.Match, bool) {
	n := len(pat)
	if pos < 0 || pos+n > len(seq) {
		return primer.Match{}, false
	}

	mm := 0
	mIdx := make([]int, 0, 4)

	// Protected windows: j in [0, leftTW) or [n-rightTW, n)
	leftCut := leftTW
	rightCut := n - rightTW

	for j := 0; j < n; j++ {
		if !primer.BaseMatch(seq[pos+j], pat[j]) {
			// inside a protected window → reject
			if j < leftCut || j >= rightCut {
				return primer.Match{}, false
			}
			mm++
			mIdx = append(mIdx, j)
			if maxMM >= 0 && mm > maxMM {
				return primer.Match{}, false
			}
		}
	}
	return primer.Match{Pos: pos, Mismatches: mm, Length: n, MismatchIdx: mIdx}, true
}
