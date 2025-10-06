// internal/clibase/examples.go
package clibase

import (
	"errors"
	"fmt"
	"io"
)

// ErrPrintedAndExitOK is returned by ParseArgs when the caller requested examples.
// Apps should catch this and exit 0 after printing examples.
var ErrPrintedAndExitOK = errors.New("examples requested")

// PrintExamples prints a small quickstart header and body, followed by a
// one-line tip to discover full help.
func PrintExamples(out io.Writer, name string, body func(io.Writer)) {
	if out == nil {
		return
	}
	_, _ = fmt.Fprintf(out, "%s — quickstart\n\n", name)
	if body != nil {
		body(out)
	}
	_, _ = fmt.Fprintln(out, "\nTip: run with --help for all flags.")
}
