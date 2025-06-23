package cli

import "flag"

// NewFlagSet returns a clean FlagSet with ContinueOnError.
func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {}
	return fs
}
