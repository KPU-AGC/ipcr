package engine

import "ipcr-core/primer"

const (
	matchCollectorSliceShrinkCap  = 16384
	matchCollectorStartsShrinkCap = 16384
	matchCollectorMapShrinkLen    = 16384
	matchCollectorLinearLimit     = 8
)

// SimulationScratch owns per-sequence buffers reused by one worker. It is not
// safe for concurrent use; the pipeline allocates one scratch value per worker.
type SimulationScratch struct {
	per        []perPair
	collectors []perPairCollectors
}

// NewSimulationScratch creates reusable buffers sized for a compiled panel.
func NewSimulationScratch(cp *CompiledPanel) *SimulationScratch {
	n := 0
	if cp != nil {
		n = len(cp.Pairs)
	}
	s := &SimulationScratch{}
	s.reset(n)
	return s
}

// NewSimulationScratch creates reusable buffers for callers using Engine as the
// simulator implementation.
func (e *Engine) NewSimulationScratch(cp *CompiledPanel) *SimulationScratch {
	return NewSimulationScratch(cp)
}

func (s *SimulationScratch) reset(pairCount int) {
	if pairCount <= 0 {
		for i := range s.per {
			s.per[i] = perPair{}
		}
		for i := range s.collectors {
			s.collectors[i].reset()
		}
		s.per = s.per[:0]
		s.collectors = s.collectors[:0]
		return
	}

	if cap(s.per) < pairCount {
		s.per = make([]perPair, pairCount)
	} else {
		s.per = s.per[:pairCount]
		for i := range s.per {
			s.per[i] = perPair{}
		}
	}

	if cap(s.collectors) < pairCount {
		s.collectors = make([]perPairCollectors, pairCount)
	} else {
		s.collectors = s.collectors[:pairCount]
		for i := range s.collectors {
			s.collectors[i].reset()
		}
	}
}

// matchCollector holds worker-local candidate state for one primer orientation.
// It keeps the first few attempted starts in a small slice and promotes to a
// map only when the candidate set becomes dense. This avoids allocating a map
// for the common sparse-hit case while still deduplicating both accepted and
// rejected candidate starts.
type matchCollector struct {
	matches []primer.Match
	starts  []int
	visited map[int]struct{}
}

func (c *matchCollector) addVerified(seq []byte, start int, pat []byte, maxMM, leftTW, rightTW, hitCap int) {
	if hitCap > 0 && len(c.matches) >= hitCap {
		return
	}
	if c.seenStart(start) {
		return
	}

	// Mark attempted starts, not just accepted starts. Seed hits and non-ACGT
	// halo starts are candidate generators only; verifyAt is deterministic for a
	// given start/pattern/window, so rechecking a rejected candidate cannot change
	// the output but can be expensive in dense approximate-seed neighborhoods.
	c.markStart(start)

	m, ok := verifyAt(seq, start, pat, maxMM, leftTW, rightTW)
	if !ok {
		return
	}
	c.matches = append(c.matches, m)
}

func (c *matchCollector) seenStart(start int) bool {
	if c.visited != nil {
		_, ok := c.visited[start]
		return ok
	}
	for _, seen := range c.starts {
		if seen == start {
			return true
		}
	}
	return false
}

func (c *matchCollector) markStart(start int) {
	if c.visited != nil {
		c.visited[start] = struct{}{}
		return
	}
	if len(c.starts) < matchCollectorLinearLimit {
		c.starts = append(c.starts, start)
		return
	}

	c.visited = make(map[int]struct{}, len(c.starts)+1)
	for _, seen := range c.starts {
		c.visited[seen] = struct{}{}
	}
	c.starts = c.starts[:0]
	c.visited[start] = struct{}{}
}

type perPairCollectors struct {
	fwdA matchCollector
	fwdB matchCollector
	revA matchCollector
	revB matchCollector
}

func (c *matchCollector) reset() {
	if cap(c.matches) > matchCollectorSliceShrinkCap {
		c.matches = nil
	} else {
		c.matches = c.matches[:0]
	}

	if cap(c.starts) > matchCollectorStartsShrinkCap {
		c.starts = nil
	} else {
		c.starts = c.starts[:0]
	}

	if c.visited == nil {
		return
	}
	if len(c.visited) > matchCollectorMapShrinkLen {
		c.visited = nil
		return
	}
	for k := range c.visited {
		delete(c.visited, k)
	}
}

func (c *perPairCollectors) reset() {
	c.fwdA.reset()
	c.fwdB.reset()
	c.revA.reset()
	c.revB.reset()
}
