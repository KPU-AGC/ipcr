// cmd/ipcr/main.go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"ipcr/internal/app"
)

func main() {
	// Derive cancellation from signals (Ctrl‑C / SIGTERM).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Stream directly to the process writers.
	code := app.RunContext(ctx, os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}
