package thermoapp

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"ipcr-core/engine"
	"ipcr-core/oligo"
	"ipcr-core/primer"

	"ipcr/internal/appcore"
	"ipcr/internal/cmdutil"
	"ipcr/internal/thermocli"
	"ipcr/internal/thermovisitors"
	"ipcr/internal/version"
	"ipcr/internal/writers"
)

// ---- pairing helpers (oligo mode) ----

func addSelfPairs(pairs []primer.Pair) []primer.Pair {
	out := make([]primer.Pair, 0, len(pairs)+2*len(pairs))
	out = append(out, pairs...)
	seenA := make(map[string]struct{})
	seenB := make(map[string]struct{})
	for _, p := range pairs {
		if p.Forward != "" {
			u := strings.ToUpper(p.Forward)
			if _, ok := seenA[u]; !ok {
				seenA[u] = struct{}{}
				out = append(out, primer.Pair{ID: p.ID + "+A:self", Forward: u, Reverse: u})
			}
		}
		if p.Reverse != "" {
			u := strings.ToUpper(p.Reverse)
			if _, ok := seenB[u]; !ok {
				seenB[u] = struct{}{}
				out = append(out, primer.Pair{ID: p.ID + "+B:self", Forward: u, Reverse: u})
			}
		}
	}
	return out
}

func pairsFromOligos(oligs []primer.Oligo, minLen, maxLen int, includeSelf bool) []primer.Pair {
	out := make([]primer.Pair, 0, len(oligs)*len(oligs))
	for i := 0; i < len(oligs); i++ {
		for j := i + 1; j < len(oligs); j++ {
			out = append(out, primer.Pair{
				ID:         fmt.Sprintf("%s+%s", oligs[i].ID, oligs[j].ID),
				Forward:    strings.ToUpper(oligs[i].Seq),
				Reverse:    strings.ToUpper(oligs[j].Seq),
				MinProduct: minLen,
				MaxProduct: maxLen,
			})
		}
	}
	if includeSelf {
		out = append(out, primer.SelfPairs(oligs)...)
	}
	return out
}

func parseOligoInline(spec string, idx int) (primer.Oligo, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return primer.Oligo{}, fmt.Errorf("empty --oligo at position %d", idx+1)
	}
	id := ""
	seq := spec
	if k := strings.IndexByte(spec, ':'); k >= 0 {
		id = strings.TrimSpace(spec[:k])
		seq = strings.TrimSpace(spec[k+1:])
	}
	if id == "" {
		id = fmt.Sprintf("O%d", idx+1)
	}
	norm, err := oligo.Validate(seq)
	if err != nil {
		return primer.Oligo{}, fmt.Errorf("--oligo %q: %v", spec, err)
	}
	return primer.Oligo{ID: id, Seq: norm}, nil
}

func loadOligosTSV(path string) ([]primer.Oligo, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }()

	var list []primer.Oligo
	sc := bufio.NewScanner(fh)
	ln := 0
	for sc.Scan() {
		ln++
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		fields := strings.Fields(line)
		switch len(fields) {
		case 1:
			norm, err := oligo.Validate(fields[0])
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %v", path, ln, err)
			}
			list = append(list, primer.Oligo{ID: fmt.Sprintf("O%d", len(list)+1), Seq: norm})
		case 2:
			norm, err := oligo.Validate(fields[1])
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %v", path, ln, err)
			}
			list = append(list, primer.Oligo{ID: fields[0], Seq: norm})
		default:
			return nil, fmt.Errorf("%s:%d: expected 1 or 2 columns (id seq), got %d", path, ln, len(fields))
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// ---- tiny helpers ----

func parseMolar(spec string) (float64, error) {
	s := strings.TrimSpace(strings.ToLower(spec))
	unit := ""
	num := s
	for _, u := range []string{"nm", "um", "mm", "m"} {
		if strings.HasSuffix(s, u) {
			unit = u
			num = strings.TrimSpace(strings.TrimSuffix(s, u))
			break
		}
	}
	if num == "" {
		return 0, fmt.Errorf("empty concentration %q", spec)
	}
	f, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, fmt.Errorf("bad concentration %q", spec)
	}
	switch unit {
	case "nm":
		return f * 1e-9, nil
	case "um":
		return f * 1e-6, nil
	case "mm":
		return f * 1e-3, nil
	case "m", "":
		return f, nil
	default:
		return 0, fmt.Errorf("unknown unit in %q", spec)
	}
}

// NeedSeq-forcing writer factory (only ipcr-thermo uses this)
type thermoWF struct {
	Format       string
	Sort         bool
	Header       bool
	Pretty       bool
	IncludeScore bool
	RankByScore  bool
}

func (w thermoWF) NeedSites() bool { return false }
func (w thermoWF) NeedSeq() bool   { return true }
func (w thermoWF) Start(out io.Writer, bufSize int) (chan<- engine.Product, <-chan error) {
	return writers.StartProductWriter(out, w.Format, w.Sort, w.Header, w.Pretty, w.IncludeScore, w.RankByScore, bufSize)
}

// ---- main app ----

func RunContext(parent context.Context, argv []string, stdout, stderr io.Writer) int {
	outw := bufio.NewWriter(stdout)
	defer func() { _ = outw.Flush() }()

	fs := thermocli.NewFlagSet("ipcr-thermo")
	fs.SetOutput(io.Discard)

	if len(argv) == 0 {
		_, _ = thermocli.ParseArgs(fs, []string{"-h"})
		fs.SetOutput(outw)
		fs.Usage()
		if err := outw.Flush(); writers.IsBrokenPipe(err) {
			return 0
		} else if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 3
		}
		return 0
	}

	opts, err := thermocli.ParseArgs(fs, argv)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(outw)
			fs.Usage()
			if e := outw.Flush(); writers.IsBrokenPipe(e) {
				return 0
			} else if e != nil {
				_, _ = fmt.Fprintln(stderr, e)
				return 3
			}
			return 0
		}
		_, _ = fmt.Fprintln(stderr, err)
		fs.SetOutput(outw)
		fs.Usage()
		if e := outw.Flush(); writers.IsBrokenPipe(e) {
			return 0
		} else if e != nil {
			_, _ = fmt.Fprintln(stderr, e)
			return 3
		}
		return 2
	}

	// Version
	if opts.Version {
		_, _ = fmt.Fprintf(outw, "ipcr-thermo version %s\n", version.Version)
		if e := outw.Flush(); writers.IsBrokenPipe(e) {
			return 0
		} else if e != nil {
			_, _ = fmt.Fprintln(stderr, e)
			return 3
		}
		return 0
	}

	// Primer/oligo input
	hasOligoMode := len(opts.OligoInline) > 0 || opts.OligosTSV != ""
	hasPairMode := opts.PrimerFile != "" || (opts.Fwd != "" || opts.Rev != "")

	if hasOligoMode && hasPairMode {
		_, _ = fmt.Fprintln(stderr, "error: --oligo/--oligos cannot be combined with --primers or --forward/--reverse")
		return 2
	}
	if !hasOligoMode && !hasPairMode {
		_, _ = fmt.Fprintln(stderr, "error: provide --oligo/--oligos OR --primers/--forward+--reverse")
		return 2
	}

	var pairs []primer.Pair

	if hasOligoMode {
		var oligs []primer.Oligo
		if opts.OligosTSV != "" {
			lo, err := loadOligosTSV(opts.OligosTSV)
			if err != nil {
				_, _ = fmt.Fprintln(stderr, err)
				return 2
			}
			oligs = append(oligs, lo...)
		}
		for i, spec := range opts.OligoInline {
			o, err := parseOligoInline(spec, i)
			if err != nil {
				_, _ = fmt.Fprintln(stderr, err)
				return 2
			}
			oligs = append(oligs, o)
		}
		if len(oligs) == 0 {
			_, _ = fmt.Fprintln(stderr, "error: no oligos provided")
			return 2
		}
		pairs = pairsFromOligos(oligs, opts.MinLen, opts.MaxLen, opts.Self)
		if len(pairs) == 0 {
			_, _ = fmt.Fprintln(stderr, "error: need ≥2 oligos for pairing (or enable --self)")
			return 2
		}
	} else {
		if opts.PrimerFile != "" {
			pairs, err = primer.LoadTSV(opts.PrimerFile)
			if err != nil {
				_, _ = fmt.Fprintln(stderr, err)
				return 2
			}
		} else {
			if opts.Fwd == "" || opts.Rev == "" {
				_, _ = fmt.Fprintln(stderr, "error: --forward and --reverse must be supplied together")
				return 2
			}
			pairs = []primer.Pair{{
				ID: "manual", Forward: strings.ToUpper(opts.Fwd), Reverse: strings.ToUpper(opts.Rev),
				MinProduct: opts.MinLen, MaxProduct: opts.MaxLen,
			}}
		}
		if opts.Self {
			pairs = addSelfPairs(pairs)
		}
	}

	// Parse solution conditions (best-effort; warn and fall back to defaults on error).
	naM, errNa := parseMolar(opts.NaSpec)
	mgM, errMg := parseMolar(opts.MgSpec) // reserved for future divalent correction
	ctM, errCt := parseMolar(opts.PrimerConcSpec)
	if errNa != nil {
		cmdutil.Warnf(stderr, opts.Quiet, "bad --na %q: %v (using 50mM)", opts.NaSpec, errNa)
		naM = 0.05
	}
	if errMg != nil {
		cmdutil.Warnf(stderr, opts.Quiet, "bad --mg %q: %v (using 3mM)", opts.MgSpec, errMg)
		mgM = 0.003
	}
	_ = mgM // not used in monovalent-only model yet
	if errCt != nil {
		cmdutil.Warnf(stderr, opts.Quiet, "bad --primer-conc %q: %v (using 250nM)", opts.PrimerConcSpec, errCt)
		ctM = 2.5e-7
	}

	// Thermo scoring visitor (thermo-only)
	visitor := &thermovisitors.Score{
		AnnealTempC:  opts.AnnealTempC,
		Na_M:         naM,
		PrimerConc_M: ctM,
		AllowIndels:  opts.AllowIndels > 0,
		LengthBiasOn: true,

		ProbeSeq:    strings.ToUpper(strings.TrimSpace(opts.Probe)),
		ProbeMaxMM:  opts.ProbeMaxMM,
		ProbeWeight: opts.ProbeWeight,

		// thermoaddons knobs
		ExtAlpha:      opts.ExtAlpha,
		LenKneeBP:     opts.LenKneeBP,
		LenSteep:      opts.LenSteep,
		LenMaxPenC:    opts.LenMaxPenC,
		StructHairpin: opts.StructHairpin,
		StructDimer:   opts.StructDimer,
		StructScale:   opts.StructScale,
		BindWeight:    opts.BindWeight,
		ExtWeight:     opts.ExtWeight,
	}

	// Prefilter gating for thermo: keep 3′ mismatches allowed (TW=0), disable seeds,
	// but limit prefilter mismatches so we don't enumerate the whole genome.
	prefMM := opts.Mismatches
	if prefMM <= 0 {
		prefMM = 4 // sensible thermo default
	}
	// If user explicitly set -m, let them know this is a *prefilter* in thermo mode.
	if opts.Mismatches > 0 {
		cmdutil.Warnf(stderr, opts.Quiet,
			"ipcr-thermo treats --mismatches=%d as a scanning prefilter; thermodynamic scoring still ranks hits", prefMM)
	}

	coreOpts := appcore.Options{
		SeqFiles:        opts.SeqFiles,
		MaxMM:           prefMM,
		TerminalWindow:  0,
		MinLen:          opts.MinLen,
		MaxLen:          opts.MaxLen,
		HitCap:          opts.HitCap,
		SeedLength:      -1,
		Circular:        opts.Circular,
		Threads:         opts.Threads,
		ChunkSize:       opts.ChunkSize,
		DedupeCap:       opts.DedupeCap,
		Quiet:           opts.Quiet,
		NoMatchExitCode: opts.NoMatchExitCode,
	}

	// Force NeedSeq=true via local writer factory, so scoring can read amplicons
	rankByScore := !strings.EqualFold(opts.Rank, "coord")

	wf := thermoWF{
		Format:       opts.Output,
		Sort:         true, // default: sort ON for ipcr-thermo
		Header:       opts.Header,
		Pretty:       opts.Pretty,
		IncludeScore: true,        // default: score column ON
		RankByScore:  rankByScore, // default: score; --rank coord flips to coord
	}

	return appcore.Run[engine.Product](parent, stdout, stderr, coreOpts, pairs, visitor.Visit, wf)
}

func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}
