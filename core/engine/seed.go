// core/engine/seed.go
package engine

import (
	"ipcr-core/primer"
)

// ---- Seeds & AC automaton --------------------------------------------------

type Seed struct {
	PairIdx    int
	Which      byte // 'A','B','a','b'
	Pat        []byte
	PrimerLen  int
	SeedOffset int
}

// configuredSeedLen returns the configured exact seed length before mismatch
// model safety constraints are applied. cfg == 0 means auto (min(12, primerLen)).
func configuredSeedLen(primerLen, cfg int) int {
	if cfg <= 0 {
		if primerLen < 12 {
			return primerLen
		}
		return 12
	}
	if cfg > primerLen {
		return primerLen
	}
	return cfg
}

// seedLenFor chooses an exact 3'-anchored seed length. When mismatches are
// allowed, exact seeding is only sound inside the protected terminal window;
// any longer seed could include a mismatch that the verifier would otherwise
// accept, causing a false negative.
func seedLenFor(primerLen, cfg, terminalWindow, maxMM int) int {
	sl := configuredSeedLen(primerLen, cfg)
	if maxMM > 0 && terminalWindow > 0 && sl > terminalWindow {
		sl = terminalWindow
	}
	if sl > primerLen {
		return primerLen
	}
	return sl
}

func isACGT(b byte) bool { return b == 'A' || b == 'C' || b == 'G' || b == 'T' }
func isUnambigBytes(p []byte) bool {
	for _, c := range p {
		if !isACGT(c) {
			return false
		}
	}
	return true
}

// BuildSeeds: 3' suffix for forward primers; 5' prefix for rc-primers.
// seedLen=0 => auto (min(12, primerLen)).
// seedLen<0 => **disable seeding** (thermo mode): return no seeds so
// SimulateBatch falls back to full FindMatches() verification for all orientations.
func BuildSeeds(pairs []primer.Pair, seedLen, terminalWindow int) (seeds []Seed, has map[int]map[byte]bool) {
	return buildSeeds(pairs, seedLen, terminalWindow, 0)
}

func buildSeeds(pairs []primer.Pair, seedLen, terminalWindow, maxMM int) (seeds []Seed, has map[int]map[byte]bool) {
	seeds = make([]Seed, 0, 4*len(pairs))
	has = make(map[int]map[byte]bool, len(pairs))

	if seedLen < 0 || (maxMM > 0 && terminalWindow <= 0) {
		// No seeds: 'has' stays empty, SimulateBatch will fallback for all orientations.
		return seeds, has
	}

	mark := func(i int, w byte) {
		m, ok := has[i]
		if !ok {
			m = make(map[byte]bool, 4)
			has[i] = m
		}
		m[w] = true
	}
	for i, p := range pairs {
		a := []byte(p.Forward)
		b := []byte(p.Reverse)
		ra := primer.RevComp(a)
		rb := primer.RevComp(b)

		// Forward A seed (3' suffix)
		if len(a) > 0 {
			sl := seedLenFor(len(a), seedLen, terminalWindow, maxMM)
			suf := a[len(a)-sl:]
			if isUnambigBytes(suf) {
				seeds = append(seeds, Seed{PairIdx: i, Which: 'A', Pat: suf, PrimerLen: len(a), SeedOffset: len(a) - sl})
				mark(i, 'A')
			}
		}
		// Forward B seed (3' suffix)
		if len(b) > 0 {
			sl := seedLenFor(len(b), seedLen, terminalWindow, maxMM)
			suf := b[len(b)-sl:]
			if isUnambigBytes(suf) {
				seeds = append(seeds, Seed{PairIdx: i, Which: 'B', Pat: suf, PrimerLen: len(b), SeedOffset: len(b) - sl})
				mark(i, 'B')
			}
		}
		// RC(A) seed: 5' prefix of rc(A)
		if len(ra) > 0 {
			sl := seedLenFor(len(ra), seedLen, terminalWindow, maxMM)
			pre := ra[:sl]
			if isUnambigBytes(pre) {
				seeds = append(seeds, Seed{PairIdx: i, Which: 'a', Pat: pre, PrimerLen: len(ra), SeedOffset: 0})
				mark(i, 'a')
			}
		}
		// RC(B) seed: 5' prefix of rc(B)
		if len(rb) > 0 {
			sl := seedLenFor(len(rb), seedLen, terminalWindow, maxMM)
			pre := rb[:sl]
			if isUnambigBytes(pre) {
				seeds = append(seeds, Seed{PairIdx: i, Which: 'b', Pat: pre, PrimerLen: len(rb), SeedOffset: 0})
				mark(i, 'b')
			}
		}
	}
	return seeds, has
}
