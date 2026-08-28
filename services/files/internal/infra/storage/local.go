// Package storage はファイル本体の保存先 (usecase.FileStorage ポート) の実装を提供する.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

// Local はローカルファイルシステムを使った usecase.FileStorage の実装.
type Local struct {
	baseDir string
}

// NewLocal は baseDir 配下にファイルを保存する Local を生成する.
func NewLocal(baseDir string) *Local {
	return &Local{baseDir: baseDir}
}

// インターフェース実装の静的チェック.
var _ usecase.FileStorage = (*Local)(nil)

// Save は key（呼び出し側が生成する UUID 文字列）に対して r の内容を保存する.
func (l *Local) Save(_ context.Context, key string, r io.Reader) error {
	if err := os.MkdirAll(l.baseDir, 0o755); err != nil {
		return fmt.Errorf("mkdir storage dir: %w", err)
	}
	f, err := os.Create(filepath.Join(l.baseDir, key))
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// Open は key で保存されたファイル本体を読み取り用に開く.
func (l *Local) Open(_ context.Context, key string) (io.ReadCloser, error) {
	f, err := os.Open(filepath.Join(l.baseDir, key))
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}
