// Package migrations は Files サービスのマイグレーション SQL をバイナリに埋め込み、
// 起動時マイグレーション (server.Run) から参照できるようにする.
package migrations

import "embed"

// FS は 000001_*.sql などのマイグレーションファイルを埋め込んだ読み取り専用 FS.
//
//go:embed *.sql
var FS embed.FS
