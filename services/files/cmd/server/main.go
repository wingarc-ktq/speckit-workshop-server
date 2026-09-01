package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "files: %v\n", err)
		os.Exit(1)
	}
}
