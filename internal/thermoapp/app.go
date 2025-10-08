// internal/thermoapp/app.go
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
	"ipcr-core/thermoaddons"

	"ipcr/internal/appcore"
	"ipcr/internal/clibase"
	"ipcr/internal/cmdutil"
	"ipcr/internal/common"
	"ipcr/internal/thermocli"
	"ipcr/internal/thermovisitors"
	"ipcr/internal/version"
	"ipcr/internal/writers"
)

/* ---------- small helpers (local, no external deps) ---------- */

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

// parseMolar: "250nM" → 2.5e-7; "50mM" → 5e-2
func parseMolar(spec string) (float64, error) {
	s := strings.TrimSpace(strings.ToLower(spec))
	unit := ""
	num := s
	for _, u := range []string{"nm", "um", "mm", "m"} {
		if strings.HasSuffix(s, u) {
			unit = u
			num = strings.TrimSuffix(s, u)
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

/* ---------- writer (forces NeedSeq + score column + rank-by-score) ---------- */

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

/* ----------------------------- main app ----------------------------- */

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
		if errors.Is(err, clibase.ErrPrintedAndExitOK) {
			thermocli.PrintExamples(outw)
			if e := outw.Flush(); writers.IsBrokenPipe(e) {
				return 0
			} else if e != nil {
				_, _ = fmt.Fprintln(stderr, e)
				return 3
			}
			return 0
		}
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

	// Input: either oligo mode or classic primer mode
	hasOligoMode := len(opts.OligoInline) > 0 || opts.OligosTSV != ""
	hasPairMode := opts.PrimerFile != "" || (opts.Fwd != "" && opts.Rev != "")

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
			var e error
			pairs, e = primer.LoadTSV(opts.PrimerFile)
			if e != nil {
				_, _ = fmt.Fprintln(stderr, e)
				return 2
			}
		} else {
			if opts.Fwd == "" || opts.Rev == "" {
				_, _ = fmt.Fprintln(stderr, "error: --forward and --reverse must be supplied together")
				return 2
			}
			pairs = []primer.Pair{
				{ID: "manual", Forward: strings.ToUpper(opts.Fwd), Reverse: strings.ToUpper(opts.Rev), MinProduct: opts.MinLen, MaxProduct: opts.MaxLen},
			}
		}
		if opts.Self {
			pairs = common.AddSelfPairsUnique(pairs)
		}
	}

	// Parse solution conditions (warn and default on errors)
	naM, errNa := parseMolar(opts.NaSpec)
	mgM, errMg := parseMolar(opts.MgSpec)
	ctM, errCt := parseMolar(opts.PrimerConcSpec)
	if errNa != nil {
		cmdutil.Warnf(stderr, opts.Quiet, "bad --na %q: %v (using 50mM)", opts.NaSpec, errNa)
		naM = 0.05
	}
	if errMg != nil {
		cmdutil.Warnf(stderr, opts.Quiet, "bad --mg %q: %v (using 3mM)", opts.MgSpec, errMg)
		mgM = 0.003
	}
	if errCt != nil {
		cmdutil.Warnf(stderr, opts.Quiet, "bad --primer-conc %q: %v (using 250nM)", opts.PrimerConcSpec, errCt)
		ctM = 2.5e-7
	}

	// Effective monovalent (optional Owczarzy-lite via env)
	naEff := thermoaddons.EffectiveMonovalent(naM, mgM)

	// Expose ssDNA mode (BS-PCR) to the scorer via env (keeps thermovisitors simple)
	if opts.SingleStranded {
		_ = os.Setenv("IPCR_SINGLE_STRANDED", "1")
	} else {
		_ = os.Unsetenv("IPCR_SINGLE_STRANDED")
	}

	// Build scorer (visitor)
	scorer := thermovisitors.Score{
		AnnealTempC:  opts.AnnealTempC,
		Na_M:         naEff,
		PrimerConc_M: ctM,
		AllowIndels:  opts.AllowIndel,
		LengthBiasOn: false, // reserved; keep behavior stable
	}

	// Core pipeline
	termWin := opts.TerminalWindow
	if termWin < 1 {
		termWin = 0
	}
	coreOpts := appcore.Options{
		SeqFiles:        opts.SeqFiles,
		MaxMM:           opts.Mismatches,
		TerminalWindow:  termWin,
		MinLen:          opts.MinLen,
		MaxLen:          opts.MaxLen,
		HitCap:          opts.HitCap,
		SeedLength:      opts.SeedLength,
		Circular:        opts.Circular,
		Threads:         opts.Threads,
		ChunkSize:       opts.ChunkSize,
		DedupeCap:       opts.DedupeCap,
		Quiet:           opts.Quiet,
		NoMatchExitCode: opts.NoMatchExitCode,
	}

	// Writer: always include score; rank-by-score if requested
	rankByScore := strings.ToLower(opts.Rank) != "coord"
	wf := thermoWF{
		Format:       opts.Output,
		Sort:         true,
		Header:       opts.Header,
		Pretty:       opts.Pretty,
		IncludeScore: true,
		RankByScore:  rankByScore,
	}

	return appcore.Run[engine.Product](parent, outw, stderr, coreOpts, pairs, scorer.Visit, wf)
}

func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}
