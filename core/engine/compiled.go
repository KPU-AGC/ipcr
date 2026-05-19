// core/engine/compiled.go
package engine

import "ipcr-core/primer"

type orientationMask uint8

const (
	orientationA orientationMask = 1 << iota
	orientationB
	orientationRCA
	orientationRCB
)

func orientationBit(which byte) orientationMask {
	switch which {
	case 'A':
		return orientationA
	case 'B':
		return orientationB
	case 'a':
		return orientationRCA
	case 'b':
		return orientationRCB
	default:
		return 0
	}
}

func (m orientationMask) has(which byte) bool {
	bit := orientationBit(which)
	return bit != 0 && m&bit != 0
}

func compiledHas(have []orientationMask, idx int, which byte) bool {
	return idx >= 0 && idx < len(have) && have[idx].has(which)
}

type primerSpan struct {
	start uint32
	len   uint16
}

func checkedPrimerSpan(start, length int) primerSpan {
	if start < 0 || length < 0 || uint64(start) > uint64(^uint32(0)) || uint64(length) > uint64(^uint16(0)) {
		panic("engine: primer orientation storage exceeded span capacity")
	}
	return primerSpan{start: uint32(start), len: uint16(length)}
}

func (cp *CompiledPanel) appendPrimerBytes(b []byte) primerSpan {
	start := len(cp.primerBytes)
	cp.primerBytes = append(cp.primerBytes, b...)
	return checkedPrimerSpan(start, len(b))
}

func (cp *CompiledPanel) appendPrimerString(s string) primerSpan {
	start := len(cp.primerBytes)
	cp.primerBytes = append(cp.primerBytes, s...)
	return checkedPrimerSpan(start, len(s))
}

func (cp *CompiledPanel) primer(span primerSpan) []byte {
	start := int(span.start)
	end := start + int(span.len)
	return cp.primerBytes[start:end]
}

func (cp *CompiledPanel) fwdASeq(i int) []byte { return cp.primer(cp.fwdA[i]) }
func (cp *CompiledPanel) fwdBSeq(i int) []byte { return cp.primer(cp.fwdB[i]) }
func (cp *CompiledPanel) rcASeq(i int) []byte  { return cp.primer(cp.rcA[i]) }
func (cp *CompiledPanel) rcBSeq(i int) []byte  { return cp.primer(cp.rcB[i]) }

// CompiledPanel contains panel-level state that can be shared read-only across
// many sequence records or chunks. Worker-local hit buffers are still allocated
// per SimulateCompiled call.
type CompiledPanel struct {
	Pairs []primer.Pair
	Cfg   Config

	primerBytes []byte
	fwdA        []primerSpan
	fwdB        []primerSpan
	rcA         []primerSpan
	rcB         []primerSpan

	SeedPatterns []SeedPattern
	Automaton    Automaton

	Have []orientationMask
}

// CompilePanel builds seed/automaton and primer-orientation state once for a
// primer panel. The returned value is immutable after construction and can be
// shared by concurrent workers.
func (e *Engine) CompilePanel(pairs []primer.Pair) *CompiledPanel {
	cp := &CompiledPanel{
		Pairs: append([]primer.Pair(nil), pairs...),
		Cfg:   e.cfg,
	}
	if len(cp.Pairs) == 0 {
		return cp
	}

	// Precompute primers once per panel (hoisted RC). All orientation sequences
	// are stored in one byte slab; per-orientation tables keep compact spans into
	// that slab rather than one Go slice allocation per primer orientation.
	cp.fwdA = make([]primerSpan, len(cp.Pairs))
	cp.fwdB = make([]primerSpan, len(cp.Pairs))
	cp.rcA = make([]primerSpan, len(cp.Pairs))
	cp.rcB = make([]primerSpan, len(cp.Pairs))
	for i := range cp.Pairs {
		cp.fwdA[i] = cp.appendPrimerString(cp.Pairs[i].Forward)
		cp.fwdB[i] = cp.appendPrimerString(cp.Pairs[i].Reverse)
		cp.rcA[i] = cp.appendPrimerBytes(primer.RevComp(cp.fwdASeq(i)))
		cp.rcB[i] = cp.appendPrimerBytes(primer.RevComp(cp.fwdBSeq(i)))
	}

	// Build deduplicated concrete A/C/G/T seed patterns and AC automaton.
	patterns, seedHave := buildSeedPatterns(cp.Pairs, e.cfg.SeedLen, e.cfg.TerminalWindow, e.cfg.MaxMM)
	cp.SeedPatterns = patterns
	cp.Automaton, _ = buildAC(patterns)
	cp.Have = make([]orientationMask, len(cp.Pairs))
	for i, byOrientation := range seedHave {
		if i < 0 || i >= len(cp.Have) {
			continue
		}
		for which, ok := range byOrientation {
			if ok {
				cp.Have[i] |= orientationBit(which)
			}
		}
	}

	return cp
}

// SimulateCompiled scans one sequence using a precompiled primer panel. The
// compiled panel captures the Config used when it was built, keeping seed
// soundness and verification parameters aligned.
func (e *Engine) SimulateCompiled(seqID string, seq []byte, cp *CompiledPanel) []Product {
	return e.SimulateCompiledWithScratch(seqID, seq, cp, nil)
}

// SimulateCompiledWithScratch scans one sequence using a precompiled primer
// panel and caller-provided worker-local scratch buffers. The scratch value is
// reset before use and is not safe to share across goroutines.
func (e *Engine) SimulateCompiledWithScratch(seqID string, seq []byte, cp *CompiledPanel, scratch *SimulationScratch) []Product {
	var out []Product
	_ = e.ForEachCompiledProduct(seqID, seq, cp, scratch, func(p Product) error {
		out = append(out, p)
		return nil
	})
	return out
}

// ForEachCompiledProduct scans one sequence with a precompiled primer panel and
// calls emit for each product as it is joined. Unlike SimulateCompiled, it does
// not materialize a sequence-level []Product, which reduces peak memory for
// dense-hit chunks. The scratch value is worker-local and must not be shared
// concurrently.
func (e *Engine) ForEachCompiledProduct(seqID string, seq []byte, cp *CompiledPanel, scratch *SimulationScratch, emit func(Product) error) error {
	if cp == nil || len(cp.Pairs) == 0 || emit == nil {
		return nil
	}

	cfg := cp.Cfg
	run := e
	if e == nil || e.cfg != cfg {
		run = &Engine{cfg: cfg}
	}

	if scratch == nil {
		scratch = NewSimulationScratch(cp)
	} else {
		scratch.reset(len(cp.Pairs))
	}
	per := scratch.per
	collectors := scratch.collectors

	maxMM := cfg.MaxMM
	tw := cfg.TerminalWindow
	hitCap := cfg.HitCap

	// HitCap truncation is order-sensitive in primer.FindMatches. With non-ACGT
	// reset bytes, the AC pass plus halo pass can observe valid starts in a
	// different order, so preserve capped semantics by using the full scanner in
	// that uncommon mode. Unlimited-hit mismatch scans use local halos instead.
	hasResetByte := maxMM > 0 && sequenceHasAutomatonReset(seq)
	forceFallback := hasResetByte && hitCap > 0

	addHit := func(pairIdx int, which byte, start int) {
		if pairIdx < 0 || pairIdx >= len(cp.Pairs) {
			return
		}

		switch which {
		case 'A':
			collectors[pairIdx].fwdA.addVerified(seq, start, cp.fwdASeq(pairIdx), maxMM, 0, tw, hitCap)
		case 'B':
			collectors[pairIdx].fwdB.addVerified(seq, start, cp.fwdBSeq(pairIdx), maxMM, 0, tw, hitCap)
		case 'a':
			collectors[pairIdx].revA.addVerified(seq, start, cp.rcASeq(pairIdx), maxMM, tw, 0, hitCap)
		case 'b':
			collectors[pairIdx].revB.addVerified(seq, start, cp.rcBSeq(pairIdx), maxMM, tw, 0, hitCap)
		}
	}

	// Verify around seed hits. If the compiled panel has no seeds, skip the
	// otherwise pointless automaton pass and go straight to the fallback scanners.
	if len(cp.SeedPatterns) > 0 && !cp.Automaton.empty() && !forceFallback {
		scanACEach(seq, cp.Automaton, func(endPos int, patternIdx int) {
			pattern := cp.SeedPatterns[patternIdx]
			// AC reports endPos as the index of the last byte of the seed pattern.
			// SeedOffset is the seed start inside the full orientation-specific
			// primer, so this formula works for forward, reverse-complement, and
			// internal seeds.
			for _, payload := range pattern.Payloads {
				start := endPos - payload.SeedOffset - (len(pattern.Pat) - 1)
				addHit(payload.PairIdx, payload.Which, start)
			}
		})
	}

	// Non-ACGT reference bytes reset the compact A/C/G/T automaton. With
	// mismatches enabled, such bytes can still be valid primer mismatches outside
	// protected terminal windows. Scan only local halos around reset-byte runs by
	// direct full-primer verification so the seed layer remains a complete
	// candidate generator without falling back for the whole sequence/chunk.
	if hasResetByte && !forceFallback {
		scanNonACGTHalos(seq, cp, addHit)
	}

	// Fallback for orientations lacking seeds (disabled seeding, unsupported
	// mismatch depths, or highly degenerate approximate neighborhoods), or for
	// hit-capped mismatch scans containing reset bytes. This stays per orientation
	// so one hard primer does not slow every primer.
	for i := range cp.Pairs {
		per[i].fwdA = collectors[i].fwdA.matches
		per[i].fwdB = collectors[i].fwdB.matches
		per[i].revA = collectors[i].revA.matches
		per[i].revB = collectors[i].revB.matches

		if forceFallback || !compiledHas(cp.Have, i, 'A') {
			per[i].fwdA = primer.FindMatches(seq, cp.fwdASeq(i), cfg.MaxMM, hitCap, tw)
		}
		if forceFallback || !compiledHas(cp.Have, i, 'B') {
			per[i].fwdB = primer.FindMatches(seq, cp.fwdBSeq(i), cfg.MaxMM, hitCap, tw)
		}
		if forceFallback || !compiledHas(cp.Have, i, 'a') {
			raw := primer.FindMatches(seq, cp.rcASeq(i), cfg.MaxMM, hitCap, 0)
			per[i].revA = filterLeftTW(raw, tw)
		}
		if forceFallback || !compiledHas(cp.Have, i, 'b') {
			raw := primer.FindMatches(seq, cp.rcBSeq(i), cfg.MaxMM, hitCap, 0)
			per[i].revB = filterLeftTW(raw, tw)
		}
	}

	for i := range cp.Pairs {
		if err := run.forEachJoinedProduct(seqID, seq, cp.Pairs[i],
			per[i].fwdA, per[i].fwdB, per[i].revA, per[i].revB, emit); err != nil {
			return err
		}
	}
	return nil
}
