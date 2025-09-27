// internal/engine/seed.go
package engine

import (
	"ipcr-core/primer"
)

// ---- Seeds & AC automaton --------------------------------------------------

type Seed struct {
	PairIdx    int   // which primer pair
	Which      byte  // 'A','B','a','b' (lower = rc primer scanned on forward strand)
	Pat        []byte
	PrimerLen  int
	SeedOffset int   // start offset of seed within the primer used for verification
}

// choose an exact 3'‑anchored seed length (exact seeds only)
func seedLenFor(primerLen int, cfg int) int {
	// cfg == 0 => auto (min(12, primerLen))
	if cfg <= 0 {
		if primerLen < 12 {
			return primerLen
		}
		return 12
	}
	if cfg > primerLen {
		return primerLen
	}
	if cfg < 1 {
		return 1
	}
	return cfg
}

func isACGT(b byte) bool { return b == 'A' || b == 'C' || b == 'G' || b == 'T' }
func isUnambigBytes(p []byte) bool {
	for _, c := range p {
		if !isACGT(c) { return false }
	}
	return true
}

// BuildSeeds: 3' suffix for forward primers; 5' prefix for rc‑primers.
// seedLen=0 => auto (min(12, primerLen)).
func BuildSeeds(pairs []primer.Pair, seedLen int, _ int) (seeds []Seed, has map[int]map[byte]bool) {
	seeds = make([]Seed, 0, 4*len(pairs))
	has = make(map[int]map[byte]bool, len(pairs))
	mark := func(i int, w byte) {
		m, ok := has[i]; if !ok { m = make(map[byte]bool, 4); has[i] = m }
		m[w] = true
	}
	for i, p := range pairs {
		a := []byte(p.Forward)
		b := []byte(p.Reverse)
		ra := primer.RevComp(a)
		rb := primer.RevComp(b)

		// Forward A seed (3' suffix)
		if len(a) > 0 {
			sl := seedLenFor(len(a), seedLen)
			suf := a[len(a)-sl:]
			if isUnambigBytes(suf) {
				seeds = append(seeds, Seed{PairIdx: i, Which: 'A', Pat: suf, PrimerLen: len(a), SeedOffset: len(a)-sl})
				mark(i, 'A')
			}
		}
		// Forward B seed (3' suffix)
		if len(b) > 0 {
			sl := seedLenFor(len(b), seedLen)
			suf := b[len(b)-sl:]
			if isUnambigBytes(suf) {
				seeds = append(seeds, Seed{PairIdx: i, Which: 'B', Pat: suf, PrimerLen: len(b), SeedOffset: len(b)-sl})
				mark(i, 'B')
			}
		}
		// RC(A) seed: 5' prefix of rc(A)
		if len(ra) > 0 {
			sl := seedLenFor(len(ra), seedLen)
			pre := ra[:sl]
			if isUnambigBytes(pre) {
				seeds = append(seeds, Seed{PairIdx: i, Which: 'a', Pat: pre, PrimerLen: len(ra), SeedOffset: 0})
				mark(i, 'a')
			}
		}
		// RC(B) seed: 5' prefix of rc(B)
		if len(rb) > 0 {
			sl := seedLenFor(len(rb), seedLen)
			pre := rb[:sl]
			if isUnambigBytes(pre) {
				seeds = append(seeds, Seed{PairIdx: i, Which: 'b', Pat: pre, PrimerLen: len(rb), SeedOffset: 0})
				mark(i, 'b')
			}
		}
	}
	return seeds, has
}

// ------------------------ Aho–Corasick (AC) -------------------------------

type acNode struct {
	next [4]int  // -1 means no edge; indices into nodes slice
	fail int
	out  []int   // seed indices ending here
}

func baseIdx(b byte) int {
	switch b {
	case 'A': return 0
	case 'C': return 1
	case 'G': return 2
	case 'T': return 3
	default:  return -1
	}
}

func buildAC(seeds []Seed) ([]acNode, []int) {
	nodes := make([]acNode, 1)
	for i := range nodes[0].next { nodes[0].next[i] = -1 }

	ends := make([]int, len(seeds)) // end node index for each seed
	// goto function
	for si, s := range seeds {
		state := 0
		for _, b := range s.Pat {
			ix := baseIdx(b)
			if ix < 0 {
				state = 0
				continue
			}
			if nodes[state].next[ix] == -1 {
				nodes[state].next[ix] = len(nodes)
				var nn acNode
				for k := range nn.next { nn.next[k] = -1 }
				nn.fail = 0
				nodes = append(nodes, nn)
			}
			state = nodes[state].next[ix]
		}
		nodes[state].out = append(nodes[state].out, si)
		ends[si] = state
	}

	// failure links (BFS)
	queue := make([]int, 0, len(nodes))
	for ch := 0; ch < 4; ch++ {
		nx := nodes[0].next[ch]
		if nx != -1 {
			nodes[nx].fail = 0
			queue = append(queue, nx)
		} else {
			nodes[0].next[ch] = 0
		}
	}
	for qh := 0; qh < len(queue); qh++ {
		r := queue[qh]
		for ch := 0; ch < 4; ch++ {
			s := nodes[r].next[ch]
			if s != -1 {
				queue = append(queue, s)
				f := nodes[r].fail
				nodes[s].fail = nodes[f].next[ch]
				nodes[s].out = append(nodes[s].out, nodes[nodes[s].fail].out...)
			} else {
				nodes[r].next[ch] = nodes[nodes[r].fail].next[ch]
			}
		}
	}
	return nodes, ends
}

type seedHit struct {
	SeedIdx int
	Pos     int // start position of the seed match in seq
}

// scanAC returns all seed matches (start coordinates).
func scanAC(seq []byte, nodes []acNode, seeds []Seed) []seedHit {
	var hits []seedHit
	state := 0
	for i := 0; i < len(seq); i++ {
		ix := baseIdx(seq[i])
		if ix < 0 {
			state = 0
			continue
		}
		state = nodes[state].next[ix]
		if len(nodes[state].out) == 0 { continue }
		for _, si := range nodes[state].out {
			hits = append(hits, seedHit{
				SeedIdx: si,
				Pos:     i - (len(seeds[si].Pat) - 1),
			})
		}
	}
	return hits
}

// ---------------------- Verification helpers -------------------------------

// verifyAt checks primer vs seq[start:] allowing up to maxMM mismatches,
// disallowing mismatches in [0,twLeft) (left end) and [n-twRight, n) (right end).
func verifyAt(seq []byte, start int, pr []byte, maxMM, twLeft, twRight int) (primer.Match, bool) {
	n := len(pr)
	if start < 0 || start+n > len(seq) { return primer.Match{}, false }
	cutLeft := twLeft
	if cutLeft < 0 { cutLeft = 0 }
	cutRightFrom := n - twRight
	if twRight <= 0 { cutRightFrom = n + 1 } // disable

	mm := 0
	var idx []int
	for j := 0; j < n; j++ {
		if !primer.BaseMatch(seq[start+j], pr[j]) {
			if j < cutLeft || j >= cutRightFrom { return primer.Match{}, false }
			mm++
			if mm > maxMM { return primer.Match{}, false }
			idx = append(idx, j)
		}
	}
	return primer.Match{Pos: start, Mismatches: mm, Length: n, MismatchIdx: idx}, true
}
