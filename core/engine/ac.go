package engine

import "ipcr-core/primer"

/*
Aho–Corasick seed scanner + local verifier.

- buildAC(seedPatterns) builds a compact DNA trie with failure links.
- scanACEach(seq, automaton, fn) streams every seed-pattern hit as (endPos, patternIdx).
- verifyAt(seq, pos, pat, ...) re-checks the full primer at 'pos' (0-based),
  counting mismatches with IUPAC and enforcing left/right terminal windows.
*/

// baseCode maps canonical DNA bases onto compact automaton transition indexes.
// Non-ACGT bytes map to -1 and reset the scanner to the root state.
var baseCode = func() [256]int8 {
	var table [256]int8
	for i := range table {
		table[i] = -1
	}
	table['A'] = 0
	table['C'] = 1
	table['G'] = 2
	table['T'] = 3
	table['a'] = 0
	table['c'] = 1
	table['g'] = 2
	table['t'] = 3
	return table
}()

// acNode is one state in the compact DNA automaton. Outputs are stored in a
// flat Automaton.out table so each node does not carry its own Go slice header.
type acNode struct {
	next     [4]uint32 // 0 => absent/root; root is state 0.
	fail     uint32
	outStart uint32
	outLen   uint32
}

// Automaton is the compiled seed matcher. Nodes are compact and seed-pattern
// outputs are stored in one flat table.
type Automaton struct {
	nodes []acNode
	out   []uint32
}

func (a Automaton) empty() bool { return len(a.nodes) == 0 }

type acBuildNode struct {
	next [4]uint32
	fail uint32
	out  []uint32
}

func acUint32(n int) uint32 {
	if n < 0 || uint64(n) > uint64(^uint32(0)) {
		panic("engine: AC automaton exceeded uint32 index capacity")
	}
	return uint32(n)
}

// buildAC constructs the compact automaton for all unique seed-pattern byte strings.
func buildAC(patterns []SeedPattern) (Automaton, int) {
	build := make([]acBuildNode, 1) // state 0 = root

	// 1) Build trie edges.
	for i, pattern := range patterns {
		cur := uint32(0)
		for _, b := range pattern.Pat {
			code := int(baseCode[b])
			if code < 0 {
				// buildSeedPatterns only emits A/C/G/T patterns. Failing loudly is
				// safer than silently suppressing a seeded orientation.
				panic("engine: buildAC received non-ACGT seed pattern")
			}
			if build[cur].next[code] == 0 {
				build = append(build, acBuildNode{})
				build[cur].next[code] = acUint32(len(build) - 1)
			}
			cur = build[cur].next[code]
		}
		build[cur].out = append(build[cur].out, acUint32(i))
	}

	// 2) BFS to set fail links and propagate outputs.
	queue := make([]uint32, 0, len(build))
	// Root's immediate children fail to root.
	for c := 0; c < 4; c++ {
		if child := build[0].next[c]; child != 0 {
			build[child].fail = 0
			queue = append(queue, child)
		}
	}
	for len(queue) > 0 {
		r := queue[0]
		queue = queue[1:]
		for c := 0; c < 4; c++ {
			s := build[r].next[c]
			if s == 0 {
				continue
			}
			queue = append(queue, s)

			// Find failure target.
			f := build[r].fail
			for f > 0 && build[f].next[c] == 0 {
				f = build[f].fail
			}
			if build[f].next[c] != 0 {
				f = build[f].next[c]
			}
			build[s].fail = f
			if len(build[f].out) > 0 {
				build[s].out = append(build[s].out, build[f].out...)
			}
		}
	}

	// 3) Flatten per-node output slices into one compact output table.
	a := Automaton{nodes: make([]acNode, len(build))}
	outCount := 0
	for _, n := range build {
		outCount += len(n.out)
	}
	a.out = make([]uint32, 0, outCount)
	for i, n := range build {
		outStart := acUint32(len(a.out))
		a.out = append(a.out, n.out...)
		a.nodes[i] = acNode{
			next:     n.next,
			fail:     n.fail,
			outStart: outStart,
			outLen:   acUint32(len(n.out)),
		}
	}

	return a, len(patterns)
}

func sequenceHasAutomatonReset(seq []byte) bool {
	for _, b := range seq {
		if baseCode[b] < 0 {
			return true
		}
	}
	return false
}

// scanACEach runs the compact automaton over seq and streams seed-pattern hits.
func scanACEach(seq []byte, automaton Automaton, fn func(endPos int, patternIdx int)) {
	if automaton.empty() || fn == nil {
		return
	}

	state := uint32(0)
	for i := 0; i < len(seq); i++ {
		code := int(baseCode[seq[i]])
		if code < 0 {
			state = 0
			continue
		}
		for state > 0 && automaton.nodes[state].next[code] == 0 {
			state = automaton.nodes[state].fail
		}
		if next := automaton.nodes[state].next[code]; next != 0 {
			state = next
		}

		node := automaton.nodes[state]
		if node.outLen == 0 {
			continue
		}
		start := int(node.outStart)
		end := start + int(node.outLen)
		for _, idx := range automaton.out[start:end] {
			fn(i, int(idx))
		}
	}
}

// verifyAt checks a full primer pattern at pos using IUPAC matching, with:
//   - maxMM      : mismatch budget (≤0 means “no mismatches allowed”)
//   - leftTW     : protected window at 5' end (for rc orientations)
//   - rightTW    : protected window at 3' end (for forward orientations)
func verifyAt(seq []byte, pos int, pat []byte, maxMM, leftTW, rightTW int) (primer.Match, bool) {
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
