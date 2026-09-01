// Package storage は Files サービスのローカルファイルストレージ実装を提供する.
package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
)

// LocalStorage はローカルファイルシステムにファイルを保存する具象実装.
type LocalStorage struct {
	basePath string
}

// NewLocalStorage は保存先ディレクトリを指定して LocalStorage を生成する.
func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{basePath: basePath}
}

// インターフェース実装の静的チェック.
var _ domain.FileStorage = (*LocalStorage)(nil)

// Save は入力ファイルをストレージに保存し、参照情報を返す.
func (s *LocalStorage) Save(ctx context.Context, content domain.FileContent) (*domain.StoredFile, error) {
	if strings.TrimSpace(content.OriginalName) == "" {
		return nil, domain.ErrInvalidFile
	}
	if content.Size <= 0 || int64(len(content.Data)) != content.Size {
		return nil, domain.ErrInvalidFile
	}
	if err := s.ensureDir(); err != nil {
		return nil, err
	}

	fileID := uuid.New()
	storedName := buildStoredName(fileID, content.OriginalName)
	path := filepath.Join(s.basePath, storedName)

	if err := os.WriteFile(path, content.Data, 0o600); err != nil {
		return nil, fmt.Errorf("write storage file: %w", err)
	}

	return &domain.StoredFile{
		ID:       fileID,
		Name:     content.OriginalName,
		Path:     path,
		Size:     content.Size,
		MIMEType: content.MIMEType,
	}, nil
}

// Open は保存済みファイルの内容を読み出す.
func (s *LocalStorage) Open(ctx context.Context, storedFile *domain.StoredFile) ([]byte, error) {
	if storedFile == nil || storedFile.Path == "" {
		return nil, domain.ErrInvalidFile
	}
	data, err := os.ReadFile(storedFile.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, domain.ErrFileNotFound
		}
		return nil, fmt.Errorf("read storage file: %w", err)
	}
	return data, nil
}

// OpenByFile はメタデータから保存時の物理パスを再構成し、ファイル内容を読み出す.
func (s *LocalStorage) OpenByFile(ctx context.Context, file *domain.File) ([]byte, error) {
	if file == nil {
		return nil, domain.ErrInvalidFile
	}
	storedPath := filepath.Join(s.basePath, buildStoredName(file.ID, file.Name))
	data, err := os.ReadFile(storedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, domain.ErrFileNotFound
		}
		return nil, fmt.Errorf("read storage file by file: %w", err)
	}
	return data, nil
}

// Delete は保存済みファイルを削除する.
func (s *LocalStorage) Delete(ctx context.Context, storedFile *domain.StoredFile) error {
	if storedFile == nil || storedFile.Path == "" {
		return domain.ErrInvalidFile
	}
	if err := os.Remove(storedFile.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete storage file: %w", err)
	}
	return nil
}

// DeleteByFile はメタデータから物理ファイルを削除する.
func (s *LocalStorage) DeleteByFile(ctx context.Context, file *domain.File) error {
	if file == nil {
		return domain.ErrInvalidFile
	}
	path := filepath.Join(s.basePath, buildStoredName(file.ID, file.Name))
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete storage file by file: %w", err)
	}
	return nil
}

func (s *LocalStorage) ensureDir() error {
	if err := os.MkdirAll(s.basePath, 0o755); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}
	return nil
}

func buildStoredName(id uuid.UUID, originalName string) string {
	base := filepath.Base(originalName)
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == "/" {
		base = "file"
	}
	return fmt.Sprintf("%s_%s", id.String(), sanitizeName(base))
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "..", "_")
	return name
}
