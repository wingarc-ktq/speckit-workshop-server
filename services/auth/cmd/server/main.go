// Package main は Auth Service の薄いエントリポイント.
//
// この main は Day 2 のリファレンス実装の入口です。
// 受講者が files サービスを実装する際の参考にしてください。
//
// 配線・起動・終了は internal/server に集約しています。main は OS レベルの関心事
// （シグナル購読・プロセス終了コード）だけを担い、薄く保ちます。
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/server"
)

func main() {
	// SIGINT / SIGTERM を受けるとキャンセルされる context（graceful shutdown の起点）.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "auth: %v\n", err)
		os.Exit(1)
	}
}
