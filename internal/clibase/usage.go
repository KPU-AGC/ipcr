// internal/clibase/usage.go
package clibase

import (
	"flag"
	"fmt"
	"io"
	"ipcr/internal/version"
)

func UsageCommon(fs *flag.FlagSet, name string, extra func(out io.Writer, def func(string) string)) {
	fs.Usage = func() {
		out := fs.Output()
		def := func(flagName string) string {
			if f := fs.Lookup(flagName); f != nil {
				return f.DefValue
			}
			return ""
		}

		_, _ = fmt.Fprintf(out, "%s – in-silico PCR toolkit\n\n", name)
		_, _ = fmt.Fprintln(out, "Author:  Erick Samera (erick.samera@kpu.ca)")
		_, _ = fmt.Fprintln(out, "License: MIT")
		_, _ = fmt.Fprintf(out, "Version: %s\n\n", version.Version)

		if extra != nil {
			extra(out, def)
		}

		_, _ = fmt.Fprintln(out, "\nInput:")
		_, _ = fmt.Fprintln(out, "  -f, --forward string        Forward primer sequence (5'→3') [*]")
		_, _ = fmt.Fprintln(out, "  -r, --reverse string        Reverse primer sequence (5'→3') [*]")
		_, _ = fmt.Fprintln(out, "  -p, --primers string        Primer TSV (id fwd rev [min] [max])")
		_, _ = fmt.Fprintln(out, "  -s, --sequences file        FASTA file(s) (repeatable) or '-' for STDIN")

		_, _ = fmt.Fprintln(out, "\nPCR:")
		_, _ = fmt.Fprintf(out, "  -m, --mismatches int        Max mismatches allowed per primer [%s]\n", def("mismatches"))
		_, _ = fmt.Fprintf(out, "      --min-length int        Minimum product length [%s]\n", def("min-length"))
		_, _ = fmt.Fprintf(out, "      --max-length int        Maximum product length [%s]\n", def("max-length"))
		_, _ = fmt.Fprintf(out, "      --hit-cap int           Max matches stored per primer/window (0=unlimited) [%s]\n", def("hit-cap"))
		_, _ = fmt.Fprintf(out, "      --terminal-window int   3' terminal window (0=allow, -1=auto) [%s]\n", def("terminal-window"))
		_, _ = fmt.Fprintf(out, "      --mode string           Matching mode: realistic | debug [%s]\n", def("mode"))
		_, _ = fmt.Fprintf(out, "      --self                  Allow single-oligo amplification (A×rc(A), B×rc(B)) [%s]\n", def("self"))

		_, _ = fmt.Fprintln(out, "\nPerformance:")
		_, _ = fmt.Fprintf(out, "  -t, --threads int           Worker threads (0=all CPUs) [%s]\n", def("threads"))
		_, _ = fmt.Fprintf(out, "      --chunk-size int        Split sequences into N-bp windows (0=no chunking) [%s]\n", def("chunk-size"))
		_, _ = fmt.Fprintf(out, "      --seed-length int       Seed length for multi-pattern scan (0=auto) [%s]\n", def("seed-length"))
		_, _ = fmt.Fprintf(out, "  -c, --circular              Treat each FASTA record as circular [%s]\n", def("circular"))

		_, _ = fmt.Fprintln(out, "\nOutput:")
		_, _ = fmt.Fprintf(out, "  -o, --output string         Output: text | json | jsonl | fasta [%s]\n", def("output"))
		_, _ = fmt.Fprintf(out, "      --products              Emit product sequences [%s]\n", def("products"))
		_, _ = fmt.Fprintf(out, "      --pretty                Pretty ASCII alignment block (text) [%s]\n", def("pretty"))
		_, _ = fmt.Fprintf(out, "      --sort                  Sort outputs deterministically [%s]\n", def("sort"))
		_, _ = fmt.Fprintf(out, "      --no-header             Suppress header line [%s]\n", def("no-header"))
		_, _ = fmt.Fprintf(out, "      --no-match-exit-code int  Exit code when no amplicons found [%s]\n", def("no-match-exit-code"))

		_, _ = fmt.Fprintln(out, "\nMiscellaneous:")
		_, _ = fmt.Fprintf(out, "  -q, --quiet                 Suppress non-essential warnings [%s]\n", def("quiet"))
		_, _ = fmt.Fprintln(out, "  -v, --version               Print version and exit")
		_, _ = fmt.Fprintln(out, "  -h, --help                  Show this help and exit")
	}
}
