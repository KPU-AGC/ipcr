// internal/thermoapp/app.go
package thermoapp

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"ipcr-core/engine"
	"ipcr-core/oligo"
	"ipcr-core/primer"
	"ipcr-core/thermoaddons"
	"ipcr/internal/appcore"
	"ipcr/internal/clibase"
	"ipcr/internal/common"
	"ipcr/internal/thermocli"
	"ipcr/internal/thermovisitors"
	"ipcr/internal/version"
	"ipcr/internal/writers"
	"os"
	"strings"
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

	// Pairwise combinations (i<j) => ID = "Oi+Oj"
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
func parseMolar(s string) (float64, error) {
	return thermoaddons.ParseConc(s)
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

	// Build primer pairs input: either primer TSV, inline primers, or oligo mode.
	var pairs []primer.Pair
	if opts.PrimerFile != "" {
		pairs, err = primer.LoadTSV(opts.PrimerFile)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 2
		}
	} else if len(opts.OligoInline) > 0 || opts.OligosTSV != "" {
		var oligs []primer.Oligo
		for i, s := range opts.OligoInline {
			o, err := parseOligoInline(s, i)
			if err != nil {
				_, _ = fmt.Fprintln(stderr, err)
				return 2
			}
			oligs = append(oligs, o)
		}
		if opts.OligosTSV != "" {
			more, err := loadOligosTSV(opts.OligosTSV)
			if err != nil {
				_, _ = fmt.Fprintln(stderr, err)
				return 2
			}
			oligs = append(oligs, more...)
		}
		if len(oligs) < 1 {
			_, _ = fmt.Fprintln(stderr, "need at least one --oligo/--oligos entry")
			return 2
		}
		pairs = pairsFromOligos(oligs, opts.MinLen, opts.MaxLen, opts.Self)
	} else {
		// Standard inline primer pair
		pairs = []primer.Pair{{
			ID:         "manual",
			Forward:    opts.Fwd,
			Reverse:    opts.Rev,
			MinProduct: opts.MinLen,
			MaxProduct: opts.MaxLen,
		}}
		if opts.Self {
			pairs = common.AddSelfPairs(pairs)
		}
	}

	// Parse thermo solution conditions
	naM, errNa := parseMolar(opts.NaSpec)
	mgM, errMg := parseMolar(opts.MgSpec)
	ctM, errCt := parseMolar(opts.PrimerConcSpec)
	if errNa != nil && !opts.Quiet {
		_, _ = fmt.Fprintf(stderr, "WARN: invalid --na %q (%v); using 50mM\n", opts.NaSpec, errNa)
		naM = 5e-2
	}
	if errMg != nil && !opts.Quiet {
		_, _ = fmt.Fprintf(stderr, "WARN: invalid --mg %q (%v); using 3mM\n", opts.MgSpec, errMg)
		mgM = 3e-3
	}
	if errCt != nil && !opts.Quiet {
		_, _ = fmt.Fprintf(stderr, "WARN: invalid --primer-conc %q (%v); using 250nM\n", opts.PrimerConcSpec, errCt)
		ctM = 2.5e-7
	}

	// Effective monovalent (optional Owczarzy-lite via env)
	naEff := thermoaddons.EffectiveMonovalent(naM, mgM)

	// Build thermo scorer (visitor)
	scorer := thermovisitors.Score{
		AnnealTempC:    opts.AnnealTempC,
		Na_M:           naEff,
		PrimerConc_M:   ctM,
		AllowIndels:    opts.AllowIndel,
		LengthBiasOn:   false, // keep stable unless/until wired as a flag
		SingleStranded: opts.SingleStranded,
		StructScale:    opts.StructScale,
		UseAutoDenom:   strings.ToLower(opts.DenomMode) == "auto",
	}

	// Terminal window normalization
	termWin := opts.TerminalWindow
	if termWin < 1 {
		termWin = 0
	}

	// Core pipeline options (NOTE: allow-softmask is threaded here)
	coreOpts := appcore.Options{
		SeqFiles:        opts.SeqFiles,
		MaxMM:           opts.Mismatches,
		TerminalWindow:  termWin,
		MinLen:          opts.MinLen,
		MaxLen:          opts.MaxLen,
		HitCap:          opts.HitCap,
		SeedLength:      opts.SeedLength,
		Circular:        opts.Circular,
		AllowSoftmask:   opts.AllowSoftmask,
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

	return appcore.Run[engine.Product](parent, stdout, stderr, coreOpts, pairs, scorer.Visit, wf)
}

func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}
