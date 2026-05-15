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
	"ipcr-core/thermo"
	"ipcr/internal/appcore"
	"ipcr/internal/clibase"
	"ipcr/internal/cmdutil"
	"ipcr/internal/common"
	"ipcr/internal/thermocli"
	"ipcr/internal/thermomodel"
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

func panelRefsFromOligos(oligs []primer.Oligo) []thermovisitors.PrimerRef {
	out := make([]thermovisitors.PrimerRef, 0, len(oligs))
	for _, o := range oligs {
		out = append(out, thermovisitors.PrimerRef{ID: o.ID, Seq: strings.ToUpper(o.Seq)})
	}
	return out
}

func panelRefsFromPairs(pairs []primer.Pair) []thermovisitors.PrimerRef {
	out := make([]thermovisitors.PrimerRef, 0, len(pairs)*2)
	for _, p := range pairs {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			id = "pair"
		}
		out = append(out, thermovisitors.PrimerRef{ID: id + ":fwd", Seq: strings.ToUpper(p.Forward)})
		out = append(out, thermovisitors.PrimerRef{ID: id + ":rev", Seq: strings.ToUpper(p.Reverse)})
	}
	return out
}

// parseMolar: "250nM" → 2.5e-7; "50mM" → 5e-2
func parseMolar(s string) (float64, error) {
	return thermo.ParseConc(s)
}

func isStrictACGTSeq(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 'A', 'C', 'G', 'T', 'a', 'c', 'g', 't':
		default:
			return false
		}
	}
	return true
}

func validateNNPrimers(mode thermomodel.Mode, pairs []primer.Pair) error {
	for _, pair := range pairs {
		if !isStrictACGTSeq(pair.Forward) || !isStrictACGTSeq(pair.Reverse) {
			return fmt.Errorf("--thermo-model %s uses strict A/C/G/T primer thermodynamics; pair %q contains degenerate/IUPAC bases", mode, pair.ID)
		}
	}
	return nil
}

/* ---------- writer (forces NeedSeq + score column + rank-by-score) ---------- */

type thermoWF struct {
	Format        string
	Sort          bool
	Header        bool
	Pretty        bool
	IncludeScore  bool
	RankByScore   bool
	ThermoDetails bool
}

func (w thermoWF) NeedSites() bool { return false }
func (w thermoWF) NeedSeq() bool   { return true }
func (w thermoWF) Start(out io.Writer, bufSize int) (chan<- engine.Product, <-chan error) {
	return writers.StartProductWriterWithThermoDetails(out, w.Format, w.Sort, w.Header, w.Pretty, w.IncludeScore, w.RankByScore, w.ThermoDetails, bufSize)
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
	var panelRefs []thermovisitors.PrimerRef
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
		panelRefs = panelRefsFromOligos(oligs)
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
		panelRefs = panelRefsFromPairs(pairs)
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

	saltModel, err := thermo.ParseSaltModel(opts.SaltModel)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	conditions := thermo.Conditions{
		AnnealC:      opts.AnnealTempC,
		NaM:          naM,
		MgM:          mgM,
		PrimerTotalM: ctM,
		SaltModel:    saltModel,
	}
	naEff := conditions.EffectiveNaM()

	mode, err := thermomodel.Parse(opts.ThermoModel)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if !mode.Implemented() {
		_, _ = fmt.Fprintf(stderr, "--thermo-model %q is reserved for staged rollout but is not implemented yet; use %q\n", mode, thermomodel.LegacyHeuristic)
		return 2
	}
	if mode == thermomodel.NNDuplexV1 || mode == thermomodel.NNStructureV1 {
		if err := validateNNPrimers(mode, pairs); err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 2
		}
	}

	// Build scorer (visitor)
	scorer := thermovisitors.Score{
		Model:          mode,
		Conditions:     conditions,
		AnnealTempC:    opts.AnnealTempC,
		Na_M:           naEff,
		PrimerConc_M:   ctM,
		AllowIndels:    opts.AllowIndel,
		LengthBiasOn:   false, // reserved; keep behavior stable
		SingleStranded: opts.SingleStranded,
		StructHairpin:  opts.StructHairpin,
		StructDimer:    opts.StructDimer,
		StructScale:    opts.StructScale,
		PanelPrimers:   panelRefs,
		ScoreProfile:   opts.ScoreProfile,
		ExtAlpha:       opts.ExtAlpha,
		ExtWeight:      opts.ExtWeight,
		LenKneeBP:      opts.LenKneeBP,
		LenSteep:       opts.LenSteep,
		LenMaxPenC:     opts.LenMaxPenC,
		BindWeight:     opts.BindWeight,
		BandMassWeight: opts.BandMassWeight,
		// NEW: enable auto-denominator when requested
		UseAutoDenom: strings.ToLower(opts.DenomMode) == "auto",
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
		Format:        opts.Output,
		Sort:          true,
		Header:        opts.Header,
		Pretty:        opts.Pretty,
		IncludeScore:  true,
		RankByScore:   rankByScore,
		ThermoDetails: opts.ThermoDetails,
	}

	return appcore.Run[engine.Product](parent, outw, stderr, coreOpts, pairs, scorer.Visit, wf)
}

func Run(argv []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), argv, stdout, stderr)
}
