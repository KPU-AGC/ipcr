// internal/clibase/common.go
package clibase

import (
	"errors"
	"flag"
	"fmt"
	"ipcr/internal/cliutil"
	"ipcr/internal/output"
	"strings"
)

// Common holds CLI fields shared by ipcr/ipcr-probe/ipcr-nested/ipcr-multiplex.
type Common struct {
	// Input
	PrimerFile string
	Fwd        string
	Rev        string
	SeqFiles   []string

	// PCR
	Mismatches     int
	MinLen         int
	MaxLen         int
	HitCap         int
	TerminalWindow int
	AllowSoftmask  bool
	Self           bool

	// Performance
	Threads    int
	ChunkSize  int
	SeedLength int
	Circular   bool
	DedupeCap  int

	// Output
	Output          string
	Products        bool
	Pretty          bool
	Sort            bool
	Header          bool // IMPORTANT: expected by cli tests and thermo cli
	NoMatchExitCode int

	// Misc
	Quiet   bool
	Version bool

	// Derived / internal
	StdinSeq bool
}

// sliceValue is a flag.Value that appends string occurrences to a slice.
type sliceValue struct{ dst *[]string }

func (s *sliceValue) String() string {
	if s.dst == nil {
		return ""
	}
	return fmt.Sprint(*s.dst)
}
func (s *sliceValue) Set(v string) error { *s.dst = append(*s.dst, v); return nil }

// Register registers common flags. Returns pointer to the no-header bool for AfterParse.
func Register(fs *flag.FlagSet, c *Common) *bool {
	// Defaults
	c.Header = true

	// Inputs
	fs.StringVar(&c.PrimerFile, "primers", "", "TSV primer file")
	fs.StringVar(&c.Fwd, "forward", "", "forward primer (5'→3')")
	fs.StringVar(&c.Rev, "reverse", "", "reverse primer (5'→3')")
	fs.StringVar(&c.PrimerFile, "p", "", "alias of --primers")
	fs.StringVar(&c.Fwd, "f", "", "alias of --forward")
	fs.StringVar(&c.Rev, "r", "", "alias of --reverse")

	seqVal := &sliceValue{dst: &c.SeqFiles}
	fs.Var(seqVal, "sequences", "FASTA file(s) (repeatable) or '-'")
	fs.Var(seqVal, "s", "alias of --sequences")

	// PCR
	fs.IntVar(&c.Mismatches, "mismatches", 0, "max mismatches per primer [0]")
	fs.IntVar(&c.MinLen, "min-length", 0, "minimum product length [0]")
	fs.IntVar(&c.MaxLen, "max-length", 2000, "maximum product length [2000]")
	fs.IntVar(&c.HitCap, "hit-cap", 10000, "max matches stored per primer/window (0=unlimited) [10000]")
	fs.IntVar(&c.TerminalWindow, "terminal-window", 3, "3' terminal window (N<1 disables) [3]")

	// NEW FLAG: allow matching in soft-masked (lowercase) reference regions
	fs.BoolVar(&c.AllowSoftmask, "allow-softmask", false,
		"allow matching within soft-masked (lowercase) reference regions; default rejects any match overlapping lowercase bases [false]")

	fs.BoolVar(&c.Self, "self", true, "allow single-oligo amplification (A×rc(A), B×rc(B)) [true]")
	fs.IntVar(&c.Mismatches, "m", 0, "alias of --mismatches")

	// Performance
	fs.IntVar(&c.Threads, "threads", 0, "worker threads (0=all CPUs) [0]")
	fs.IntVar(&c.ChunkSize, "chunk-size", 0, "split sequences into N-bp windows (0=no chunking) [0]")
	fs.IntVar(&c.SeedLength, "seed-length", 12, "seed length for multi-pattern scan (0=auto) [12]")
	fs.IntVar(&c.Threads, "t", 0, "alias of --threads")
	fs.BoolVar(&c.Circular, "circular", false, "treat each FASTA record as circular [false]")
	fs.BoolVar(&c.Circular, "c", false, "alias of --circular")
	fs.IntVar(&c.DedupeCap, "dedupe-cap", 0, "LRU size for de-duplicating cross-chunk hits (0=default) [0]")

	// Output
	fs.StringVar(&c.Output, "output", output.FormatText, "output: text | json | jsonl | fasta [text]")
	fs.StringVar(&c.Output, "o", output.FormatText, "alias of --output")
	fs.BoolVar(&c.Products, "products", false, "emit product sequences [false]")
	fs.BoolVar(&c.Pretty, "pretty", false, "pretty alignment block (text) [false]")
	fs.BoolVar(&c.Sort, "sort", false, "sort outputs deterministically [false]")
	noHeader := fs.Bool("no-header", false, "suppress header line [false]")
	fs.IntVar(&c.NoMatchExitCode, "no-match-exit-code", 0, "exit code when no amplicons found [0]")

	// Misc
	fs.BoolVar(&c.Quiet, "quiet", false, "suppress non-essential warnings [false]")
	fs.BoolVar(&c.Quiet, "q", false, "alias of --quiet")
	fs.BoolVar(&c.Version, "version", false, "print version and exit")
	fs.BoolVar(&c.Version, "v", false, "alias of --version")

	return noHeader
}

// AfterParse applies derived values and validates.
func AfterParse(fs *flag.FlagSet, c *Common, noHeader *bool, posArgs []string) error {
	// Expand positional globs and append.
	if len(posArgs) > 0 {
		exp, err := cliutil.ExpandPositionals(posArgs)
		if err != nil {
			return err
		}
		c.SeqFiles = append(c.SeqFiles, exp...)
	}

	if c.PrimerFile == "" && (c.Fwd == "" || c.Rev == "") {
		return errors.New("need --forward/--reverse or --primers")
	}
	if c.PrimerFile != "" && (c.Fwd != "" || c.Rev != "") {
		return errors.New("--primers conflicts with --forward/--reverse")
	}
	if len(c.SeqFiles) == 0 {
		return errors.New("need --sequences or positional FASTA files")
	}

	for _, s := range c.SeqFiles {
		if s == "-" {
			c.StdinSeq = true
			break
		}
	}

	// Header behavior (repo expects Common.Header; tests assert it)
	if noHeader != nil && *noHeader {
		c.Header = false
	} else {
		c.Header = true
	}

	// normalize/validate output
	c.Output = strings.ToLower(strings.TrimSpace(c.Output))
	switch c.Output {
	case output.FormatText, output.FormatJSON, output.FormatJSONL, output.FormatFASTA:
	default:
		return fmt.Errorf("unknown --output %q", c.Output)
	}
	if c.Output == output.FormatFASTA {
		// FASTA output requires sequences emitted.
		c.Products = true
	}

	_ = fs
	return nil
}
