// internal/clibase/usage.go  (NEW FILE)
package clibase

import (
	"flag"
	"fmt"
	"io"

	"ipcr/internal/version"
)

// UsageCommon installs a shared Usage() handler on fs.
// extra prints tool-specific sections (usage examples, probe/inner blocks, etc.).
func UsageCommon(fs *flag.FlagSet, name string, extra func(out io.Writer, def func(string) string)) {
	fs.Usage = func() {
		out := fs.Output()
		def := func(flagName string) string {
			if f := fs.Lookup(flagName); f != nil {
				return f.DefValue
			}
			return ""
		}

		// Header
		fmt.Fprintf(out, "%s – in-silico PCR toolkit\n\n", name)
		fmt.Fprintln(out, "Author:  Erick Samera (erick.samera@kpu.ca)")
		fmt.Fprintln(out, "License: MIT")
		fmt.Fprintf(out, "Version: %s\n\n", version.Version)

		// Tool-specific additions (usage examples, extra sections)
		if extra != nil {
			extra(out, def)
		}

		// Shared blocks
		fmt.Fprintln(out, "\nInput:")
		fmt.Fprintln(out, "  -f, --forward string        Forward primer sequence (5'→3') [*]")
		fmt.Fprintln(out, "  -r, --reverse string        Reverse primer sequence (5'→3') [*]")
		fmt.Fprintln(out, "  -p, --primers string        Primer TSV (id fwd rev [min] [max])")
		fmt.Fprintln(out, "  -s, --sequences file        FASTA file(s) (repeatable) or '-' for STDIN")

		fmt.Fprintln(out, "\nPCR:")
		fmt.Fprintf(out, "  -m, --mismatches int        Max mismatches allowed per primer [%s]\n", def("mismatches"))
		fmt.Fprintf(out, "      --min-length int        Minimum product length [%s]\n", def("min-length"))
		fmt.Fprintf(out, "      --max-length int        Maximum product length [%s]\n", def("max-length"))
		fmt.Fprintf(out, "      --hit-cap int           Max matches stored per primer/window (0=unlimited) [%s]\n", def("hit-cap"))
		fmt.Fprintf(out, "      --terminal-window int   3' terminal window (0=allow, -1=auto) [%s]\n", def("terminal-window"))
		fmt.Fprintf(out, "      --mode string           Matching mode: realistic | debug [%s]\n", def("mode"))

		fmt.Fprintln(out, "\nPerformance:")
		fmt.Fprintf(out, "  -t, --threads int           Worker threads (0=all CPUs) [%s]\n", def("threads"))
		fmt.Fprintf(out, "      --chunk-size int        Split sequences into N-bp windows (0=no chunking) [%s]\n", def("chunk-size"))
		fmt.Fprintf(out, "      --seed-length int       Seed length for multi-pattern scan (0=auto) [%s]\n", def("seed-length"))
		fmt.Fprintf(out, "  -c, --circular              Treat each FASTA record as circular [%s]\n", def("circular"))

		fmt.Fprintln(out, "\nOutput:")
		fmt.Fprintf(out, "  -o, --output string         Output: text | json | jsonl | fasta [%s]\n", def("output"))
		fmt.Fprintf(out, "      --products              Emit product sequences [%s]\n", def("products"))
		fmt.Fprintf(out, "      --pretty                Pretty ASCII alignment block (text) [%s]\n", def("pretty"))
		fmt.Fprintf(out, "      --sort                  Sort outputs deterministically [%s]\n", def("sort"))
		fmt.Fprintf(out, "      --no-header             Suppress header line [%s]\n", def("no-header"))
		fmt.Fprintf(out, "      --no-match-exit-code int  Exit code when no amplicons found [%s]\n", def("no-match-exit-code"))

		fmt.Fprintln(out, "\nMiscellaneous:")
		fmt.Fprintf(out, "  -q, --quiet                 Suppress non-essential warnings [%s]\n", def("quiet"))
		fmt.Fprintln(out, "  -v, --version               Print version and exit")
		fmt.Fprintln(out, "  -h, --help                  Show this help and exit")
	}
}
