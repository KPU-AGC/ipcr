// internal/clibase/common.go
package clibase

import (
	"errors"
	"flag"
	"fmt"

	"ipcr/internal/cliutil"
)

// Common holds CLI fields shared by ipcr and ipcr-probe.
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
	Mode           string

	// Performance
	Threads    int
	ChunkSize  int
	SeedLength int
	Circular   bool

	// Output
	Output          string // text|json|jsonl|fasta
	Products        bool
	Pretty          bool
	Sort            bool
	Header          bool
	NoMatchExitCode int

	// Misc
	Quiet   bool
	Version bool
}

// sliceValue appends each value to a *[]string (for --sequences/-s)
type sliceValue struct{ dst *[]string }

func (s *sliceValue) String() string {
	if s.dst == nil { return "" }
	return fmt.Sprint(*s.dst)
}
func (s *sliceValue) Set(v string) error {
	*s.dst = append(*s.dst, v)
	return nil
}

// Register wires shared flags onto fs and returns a pointer to the “no-header” bool
// that the caller can use to set Common.Header = !noHeader after parsing.
func Register(fs *flag.FlagSet, c *Common) *bool {
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
	fs.IntVar(&c.TerminalWindow, "terminal-window", -1, "3' terminal window (0=allow, -1=auto) [-1]")
	fs.StringVar(&c.Mode, "mode", "realistic", "matching mode: realistic | debug")
	fs.IntVar(&c.Mismatches, "m", 0, "alias of --mismatches")

	// Performance
	fs.IntVar(&c.Threads, "threads", 0, "worker threads (0=all CPUs) [0]")
	fs.IntVar(&c.ChunkSize, "chunk-size", 0, "split sequences into N-bp windows (0=no chunking) [0]")
	fs.IntVar(&c.SeedLength, "seed-length", 12, "seed length for multi-pattern scan (0=auto) [12]")
	fs.IntVar(&c.Threads, "t", 0, "alias of --threads")
	fs.BoolVar(&c.Circular, "circular", false, "treat each FASTA record as circular [false]")
	fs.BoolVar(&c.Circular, "c", false, "alias of --circular")

	// Output
	fs.StringVar(&c.Output, "output", "text", "output: text | json | jsonl | fasta [text]")
	fs.StringVar(&c.Output, "o", "text", "alias of --output")
	fs.BoolVar(&c.Products, "products", false, "emit product sequences [false]")
	fs.BoolVar(&c.Pretty, "pretty", false, "pretty ASCII alignment block (text) [false]")
	fs.BoolVar(&c.Sort, "sort", false, "sort outputs deterministically [false]")
	noHeader := false
	fs.BoolVar(&noHeader, "no-header", false, "suppress header line [false]")
	fs.IntVar(&c.NoMatchExitCode, "no-match-exit-code", 1, "exit code when no amplicons found [1]")

	// Misc
	fs.BoolVar(&c.Quiet, "quiet", false, "suppress non-essential warnings [false]")
	fs.BoolVar(&c.Quiet, "q", false, "alias of --quiet")
	fs.BoolVar(&c.Version, "v", false, "print version and exit [false]")
	fs.BoolVar(&c.Version, "version", false, "print version and exit [false]")

	return &noHeader
}

// AfterParse finalizes header and expands positionals, then runs shared validation.
func AfterParse(fs *flag.FlagSet, c *Common, noHeader *bool, posArgs []string) error {
	c.Header = !*noHeader

	if len(posArgs) > 0 {
		exp, err := cliutil.ExpandPositionals(posArgs)
		if err != nil {
			return err
		}
		c.SeqFiles = append(c.SeqFiles, exp...)
	}
	return Validate(c)
}

// Validate applies shared CLI invariants used by all tools.
func Validate(c *Common) error {
	usingFile := c.PrimerFile != ""
	usingInline := c.Fwd != "" || c.Rev != ""
	switch {
	case usingFile && usingInline:
		return errors.New("--primers conflicts with --forward/--reverse")
	case usingInline && (c.Fwd == "" || c.Rev == ""):
		return errors.New("--forward and --reverse must be supplied together")
	case !usingFile && !usingInline:
		return errors.New("provide --primers or --forward/--reverse")
	}
	if len(c.SeqFiles) == 0 {
		return errors.New("at least one sequence file is required")
	}
	if c.Threads < 0 {
		return errors.New("--threads must be ≥ 0")
	}
	if c.ChunkSize < 0 {
		return errors.New("--chunk-size must be ≥ 0")
	}
	if c.HitCap < 0 {
		return errors.New("--hit-cap must be ≥ 0")
	}
	switch c.Output {
	case "text", "json", "jsonl", "fasta":
	default:
		return fmt.Errorf("invalid --output %q", c.Output)
	}
	if c.TerminalWindow < -1 {
		return errors.New("--terminal-window must be ≥ -1")
	}
	if c.NoMatchExitCode < 0 || c.NoMatchExitCode > 255 {
		return errors.New("--no-match-exit-code must be between 0 and 255")
	}
	return nil
}
