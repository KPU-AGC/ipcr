package thermocli

import (
	"flag"
	"fmt"
	"io"
	"ipcr/internal/clibase"
	"ipcr/internal/cliutil"
	"strings"
)

type sliceValue struct{ dst *[]string }

func (s *sliceValue) String() string {
	if s.dst == nil {
		return ""
	}
	return strings.Join(*s.dst, ",")
}
func (s *sliceValue) Set(v string) error { *s.dst = append(*s.dst, v); return nil }

type Options struct {
	clibase.Common

	// Thermo knobs
	AnnealTempC    float64
	NaSpec         string
	MgSpec         string
	PrimerConcSpec string
	AllowIndels    int

	// Oligo input
	OligoInline []string
	OligosTSV   string

	// Probe
	Probe       string
	ProbeName   string
	ProbeMaxMM  int
	ProbeWeight float64

	// Ranking/output (NOTE: score is always included in ipcr-thermo)
	Rank string

	// NEW thermoaddons knobs (thermo-only)
	ExtAlpha      float64
	LenKneeBP     int
	LenSteep      float64
	LenMaxPenC    float64
	StructHairpin bool
	StructDimer   bool
	StructScale   float64
	BindWeight    float64
	ExtWeight     float64
}

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	clibase.UsageCommon(fs, name, func(out io.Writer, _ func(string) string) {
		_, _ = fmt.Fprintln(out, "Usage:")
		_, _ = fmt.Fprintf(out, "  %s [options] --forward AAA --reverse TTT ref.fa[.gz]\n", name)
		_, _ = fmt.Fprintf(out, "  %s [options] --primers panel.tsv ref*.fa.gz\n", name)
		_, _ = fmt.Fprintf(out, "  %s [options] --oligo ID:SEQ --oligo ID2:SEQ ... ref.fa[.gz]\n", name)
		_, _ = fmt.Fprintf(out, "  %s [options] --oligos oligos.tsv ref*.fa.gz\n", name)

		_, _ = fmt.Fprintln(out, "\nOligo input:")
		_, _ = fmt.Fprintln(out, "      --oligo string         Oligo (ID:SEQ or SEQ). Repeatable.")
		_, _ = fmt.Fprintln(out, "      --oligos string        Oligo TSV (two columns: id seq)")

		_, _ = fmt.Fprintln(out, "\nThermo scoring:")
		_, _ = fmt.Fprintf(out, "      --anneal-temp float    Annealing temperature (°C) [%s]\n", "60")
		_, _ = fmt.Fprintf(out, "      --na string            Monovalent salt, e.g., 50mM [%s]\n", "50mM")
		_, _ = fmt.Fprintf(out, "      --mg string            Mg2+, e.g., 3mM [%s]\n", "3mM")
		_, _ = fmt.Fprintf(out, "      --primer-conc string   Primer concentration, e.g., 250nM [%s]\n", "250nM")
		_, _ = fmt.Fprintf(out, "      --allow-indels int     Allow up to N 1-nt gaps per primer [%s]\n", "0")
		// New clarity: mismatches are a prefilter in thermo
		_, _ = fmt.Fprintln(out, "      NOTE: In ipcr-thermo, --mismatches acts as a scanning prefilter;")
		_, _ = fmt.Fprintln(out, "            thermodynamic scoring still ranks hits.")

		_, _ = fmt.Fprintln(out, "\nProbe (optional):")
		_, _ = fmt.Fprintf(out, "      --probe string         Internal probe (5'→3') [%s]\n", "")
		_, _ = fmt.Fprintf(out, "      --probe-name string    Probe label [%s]\n", "probe")
		_, _ = fmt.Fprintf(out, "      --probe-max-mm int     Max probe mismatches allowed [%s]\n", "0")
		_, _ = fmt.Fprintf(out, "      --probe-weight float   Blend [0..1]: (1=min of margins) [%s]\n", "1.0")

		_, _ = fmt.Fprintln(out, "\nThermo extensions (scoring only; thermo binary):")
		_, _ = fmt.Fprintf(out, "      --ext-alpha float      Slope for extension prob vs margin [%s]\n", "0.45")
		_, _ = fmt.Fprintf(out, "      --length-knee-bp int   Soft-knee start (bp) for length bias [%s]\n", "550")
		_, _ = fmt.Fprintf(out, "      --length-steep float   Soft-knee steepness [%s]\n", "0.003")
		_, _ = fmt.Fprintf(out, "      --length-max-pen float Max °C-equivalent length penalty [%s]\n", "10")
		_, _ = fmt.Fprintln(out, "      --struct-hairpin       Penalize hairpins [true]")
		_, _ = fmt.Fprintln(out, "      --struct-dimer         Penalize primer-dimers [true]")
		_, _ = fmt.Fprintf(out, "      --struct-scale float   Structural penalties scale [%s]\n", "1.0")
		_, _ = fmt.Fprintf(out, "      --bind-weight float    Reserved bind weight (logit occupancy) [%s]\n", "1.0")
		_, _ = fmt.Fprintf(out, "      --ext-weight float     Weight for extension logit term [%s]\n", "1.0")

		_, _ = fmt.Fprintln(out, "\nRanking & outputs (thermo):")
		_, _ = fmt.Fprintln(out, "      score field is always included in outputs (TSV/JSON/JSONL).")
		_, _ = fmt.Fprintf(out, "      --rank string          Order by: score | coord [%s]\n", "score")
		_, _ = fmt.Fprintln(out, "      (default is score; pass --rank coord to keep coordinate order.)")
	})
	return fs
}

func ParseArgs(fs *flag.FlagSet, argv []string) (Options, error) {
	var o Options
	var help bool

	noHeader := clibase.Register(fs, &o.Common)

	oligoFlag := &sliceValue{dst: &o.OligoInline}
	fs.Var(oligoFlag, "oligo", "oligo (ID:SEQ or SEQ); repeatable")
	fs.StringVar(&o.OligosTSV, "oligos", "", "oligo TSV with 2 cols: id seq")

	fs.Float64Var(&o.AnnealTempC, "anneal-temp", 60, "annealing temperature (°C)")
	fs.StringVar(&o.NaSpec, "na", "50mM", "monovalent salt (e.g., 50mM)")
	fs.StringVar(&o.MgSpec, "mg", "3mM", "Mg2+ (e.g., 3mM)")
	fs.StringVar(&o.PrimerConcSpec, "primer-conc", "250nM", "primer concentration (e.g., 250nM)")
	fs.IntVar(&o.AllowIndels, "allow-indels", 0, "allow up to N 1-nt gaps per primer [0]")

	fs.StringVar(&o.Probe, "probe", "", "internal probe (5'→3') [optional]")
	fs.StringVar(&o.ProbeName, "probe-name", "probe", "probe label")
	fs.IntVar(&o.ProbeMaxMM, "probe-max-mm", 0, "max probe mismatches [0]")
	fs.Float64Var(&o.ProbeWeight, "probe-weight", 1.0, "blend [0..1]: 1 favors probe strongly")

	// scores flag removed: score is always on in ipcr-thermo
	fs.StringVar(&o.Rank, "rank", "score", "order by: score | coord")
	fs.BoolVar(&help, "h", false, "show this help [false]")

	// NEW thermoaddons knobs (with defaults)
	fs.Float64Var(&o.ExtAlpha, "ext-alpha", 0.45, "slope for extension prob vs margin")
	fs.IntVar(&o.LenKneeBP, "length-knee-bp", 550, "soft-knee start (bp)")
	fs.Float64Var(&o.LenSteep, "length-steep", 0.003, "soft-knee steepness")
	fs.Float64Var(&o.LenMaxPenC, "length-max-pen", 10, "max length penalty (°C)")
	fs.BoolVar(&o.StructHairpin, "struct-hairpin", true, "penalize hairpins")
	fs.BoolVar(&o.StructDimer, "struct-dimer", true, "penalize primer-dimers")
	fs.Float64Var(&o.StructScale, "struct-scale", 1.0, "scale for structural penalties")
	fs.Float64Var(&o.BindWeight, "bind-weight", 1.0, "bind weight (reserved)")
	fs.Float64Var(&o.ExtWeight, "ext-weight", 1.0, "extension weight")

	flagArgs, posArgs := cliutil.SplitFlagsAndPositionals(fs, argv)
	if err := fs.Parse(flagArgs); err != nil {
		return o, err
	}
	if help {
		return o, flag.ErrHelp
	}

	// Manual finalize (thermo): positionals → seq files; allow oligo OR primers
	o.Common.Header = !*noHeader
	if len(posArgs) > 0 {
		exp, err := cliutil.ExpandPositionals(posArgs)
		if err != nil {
			return o, err
		}
		o.Common.SeqFiles = append(o.Common.SeqFiles, exp...)
	}
	hasOligoMode := len(o.OligoInline) > 0 || o.OligosTSV != ""
	hasPairMode := o.PrimerFile != "" || (o.Fwd != "" && o.Rev != "")
	if !hasOligoMode && !hasPairMode {
		return o, fmt.Errorf("provide --oligo/--oligos OR --primers/--forward+--reverse")
	}
	if len(o.SeqFiles) == 0 {
		return o, fmt.Errorf("at least one sequence file is required (positional or --sequences)")
	}
	switch strings.ToLower(o.Rank) {
	case "coord", "score":
	default:
		return o, fmt.Errorf("--rank must be 'coord' or 'score'")
	}
	if o.AllowIndels < 0 || o.AllowIndels > 1 {
		return o, fmt.Errorf("--allow-indels must be 0 or 1")
	}
	if o.ProbeWeight < 0 || o.ProbeWeight > 1 {
		return o, fmt.Errorf("--probe-weight must be in [0,1]")
	}
	return o, nil
}
