package appshell

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"
)

func Main(run func(context.Context, []string, io.Writer, io.Writer) int) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	argv := os.Args[1:]
	if len(argv) == 0 {
		argv = []string{"-h"}
	}

	code := run(ctx, argv, os.Stdout, os.Stderr)
	// Normalize cancellation exit code.
	if ctx.Err() != nil && code == 0 {
		code = 130
	}

	stop()
	os.Exit(code)
}
