package engine

// seqRange is a half-open interval [start, end) over sequence coordinates.
type seqRange struct {
	start int
	end   int
}

func nonACGTRanges(seq []byte) []seqRange {
	var ranges []seqRange
	for i := 0; i < len(seq); {
		if baseCode[seq[i]] >= 0 {
			i++
			continue
		}

		start := i
		for i < len(seq) && baseCode[seq[i]] < 0 {
			i++
		}
		ranges = append(ranges, seqRange{start: start, end: i})
	}
	return ranges
}

func forEachNonACGTHaloStart(seqLen, primerLen int, ranges []seqRange, fn func(start int)) {
	if seqLen <= 0 || primerLen <= 0 || primerLen > seqLen || len(ranges) == 0 || fn == nil {
		return
	}

	limit := seqLen - primerLen
	curLo, curHi := -1, -1
	emit := func() {
		if curLo < 0 {
			return
		}
		for start := curLo; start <= curHi; start++ {
			fn(start)
		}
	}

	for _, r := range ranges {
		if r.end <= r.start {
			continue
		}

		lo := r.start - primerLen + 1
		if lo < 0 {
			lo = 0
		}
		hi := r.end - 1
		if hi > limit {
			hi = limit
		}
		if hi < lo {
			continue
		}

		if curLo < 0 {
			curLo, curHi = lo, hi
			continue
		}
		if lo <= curHi+1 {
			if hi > curHi {
				curHi = hi
			}
			continue
		}

		emit()
		curLo, curHi = lo, hi
	}
	emit()
}

func scanNonACGTHalos(seq []byte, cp *CompiledPanel, tryStart func(pairIdx int, which byte, start int)) {
	if cp == nil || cp.Cfg.MaxMM <= 0 || tryStart == nil {
		return
	}

	ranges := nonACGTRanges(seq)
	if len(ranges) == 0 {
		return
	}

	for i := range cp.Pairs {
		if compiledHas(cp.Have, i, 'A') {
			forEachNonACGTHaloStart(len(seq), len(cp.fwdASeq(i)), ranges, func(start int) {
				tryStart(i, 'A', start)
			})
		}
		if compiledHas(cp.Have, i, 'B') {
			forEachNonACGTHaloStart(len(seq), len(cp.fwdBSeq(i)), ranges, func(start int) {
				tryStart(i, 'B', start)
			})
		}
		if compiledHas(cp.Have, i, 'a') {
			forEachNonACGTHaloStart(len(seq), len(cp.rcASeq(i)), ranges, func(start int) {
				tryStart(i, 'a', start)
			})
		}
		if compiledHas(cp.Have, i, 'b') {
			forEachNonACGTHaloStart(len(seq), len(cp.rcBSeq(i)), ranges, func(start int) {
				tryStart(i, 'b', start)
			})
		}
	}
}
