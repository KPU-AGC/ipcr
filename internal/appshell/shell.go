// internal/appshell/shell.go  (NEW)
package appshell

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"
)

// Main wraps a RunContext-style entrypoint with Ctrl-C handling and "no args → help".
func Main(run func(context.Context, []string, io.Writer, io.Writer) int) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	argv := os.Args[1:]
	if len(argv) == 0 {
		argv = []string{"-h"}
	}

	code := run(ctx, argv, os.Stdout, os.Stderr)
	// Normalize cancellation exit code.
	if ctx.Err() != nil && code == 0 {
		code = 130
	}
	os.Exit(code)
}
