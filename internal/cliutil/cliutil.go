// internal/cliutil/cliutil.go
package cliutil

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

// BoolFlags returns names of flags that don't require a value.
func BoolFlags(fs *flag.FlagSet) map[string]bool {
	m := map[string]bool{}
	fs.VisitAll(func(f *flag.Flag) {
		if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
			m[f.Name] = true
		}
	})
	return m
}

// SplitFlagsAndPositionals separates flag-like args from positionals,
// preserving '-','--','--x=y' semantics. Use before fs.Parse(flagArgs).
func SplitFlagsAndPositionals(fs *flag.FlagSet, argv []string) (flagArgs, posArgs []string) {
	boolFlags := BoolFlags(fs)
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if arg == "--" {
			posArgs = append(posArgs, argv[i+1:]...)
			break
		}
		if arg == "-" {
			posArgs = append(posArgs, arg)
			continue
		}
		if strings.HasPrefix(arg, "-") {
			if strings.Contains(arg, "=") {
				flagArgs = append(flagArgs, arg)
				continue
			}
			name := strings.TrimLeft(arg, "-")
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				name = name[:eq]
			}
			needsVal := !boolFlags[name]
			flagArgs = append(flagArgs, arg)
			if needsVal && i+1 < len(argv) {
				flagArgs = append(flagArgs, argv[i+1])
				i++
			}
			continue
		}
		posArgs = append(posArgs, arg)
	}
	return
}

func hasGlobMeta(s string) bool { return strings.ContainsAny(s, "*?[") }

// ExpandPositionals expands any globs among path-like positionals.
func ExpandPositionals(posArgs []string) ([]string, error) {
	var out []string
	for _, a := range posArgs {
		if a == "-" {
			out = append(out, a)
			continue
		}
		if hasGlobMeta(a) {
			m, err := filepath.Glob(a)
			if err != nil {
				return nil, fmt.Errorf("bad glob %q: %v", a, err)
			}
			if len(m) == 0 {
				return nil, fmt.Errorf("no input matched %q", a)
			}
			out = append(out, m...)
		} else {
			out = append(out, a)
		}
	}
	return out, nil
}
