package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
)

type LocalStorage struct {
	root string
}

func NewLocalStorage(root string) (*LocalStorage, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	return &LocalStorage{root: root}, nil
}

func (s *LocalStorage) Save(ctx context.Context, fileName string, reader io.Reader, maxBytes int64) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if maxBytes <= 0 {
		maxBytes = domain.MaxFileSize
	}

	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return "", domain.ErrInvalidFileName
	}
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return "", domain.ErrInvalidFileName
	}

	payload, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read bytes: %w", err)
	}
	if int64(len(payload)) > maxBytes {
		return "", domain.ErrFileTooLarge
	}

	storageKey := uuid.NewString()
	path, err := safePath(s.root, storageKey)
	if err != nil {
		return "", fmt.Errorf("safe path: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return storageKey, nil
}

func (s *LocalStorage) Open(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	path, err := safePath(s.root, storageKey)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open storage file: %w", err)
	}
	return f, nil
}

func (s *LocalStorage) Delete(ctx context.Context, storageKey string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if storageKey == "" {
		return nil
	}
	path, err := safePath(s.root, storageKey)
	if err != nil {
		return nil
	}
	return os.Remove(path)
}
