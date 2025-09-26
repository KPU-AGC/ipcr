// cmd/ipcr-probe/main.go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"ipcr/internal/probeapp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	code := probeapp.RunContext(ctx, os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}
