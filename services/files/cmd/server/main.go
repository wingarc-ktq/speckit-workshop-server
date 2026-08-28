// Package main は Files Service の薄いエントリポイント.
//
// 配線・起動・終了は internal/server に集約している。main は OS レベルの関心事
// （シグナル購読・プロセス終了コード）だけを担い、薄く保つ（auth の
// cmd/server/main.go と同じ方針。Constitution III）.
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
	// SIGINT / SIGTERM を受けるとキャンセルされる context（graceful shutdown の起点）.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "files: %v\n", err)
		os.Exit(1)
	}
}
