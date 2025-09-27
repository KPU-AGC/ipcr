// ./cmd/ipcr-multiplex/main.go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"ipcr/internal/multiplexapp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	code := multiplexapp.RunContext(ctx, os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}
