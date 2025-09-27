// ./cmd/ipcr-nested/main.go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"ipcr/internal/nestedapp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	code := nestedapp.RunContext(ctx, os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}
