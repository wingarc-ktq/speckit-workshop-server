// Package migrations は Auth サービスのマイグレーション SQL をバイナリに埋め込み、
// 起動時マイグレーション (server.Run) から参照できるようにする.
//
// スキーマ管理をサービス自身が持つことで、compose 側に per-service の migrate を
// 並べる必要がなくなる (サービスを増やしても compose は増えない).
package migrations

import "embed"

// FS は 000001_*.sql などのマイグレーションファイルを埋め込んだ読み取り専用 FS.
//
//go:embed *.sql
var FS embed.FS
