package storage_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/storage"
)

func TestLocal_SaveAndOpen(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := storage.NewLocal(dir)
	ctx := context.Background()

	const key = "test-key"
	const content = "hello, files service"

	if err := s.Save(ctx, key, strings.NewReader(content)); err != nil {
		t.Fatal(err)
	}

	rc, err := s.Open(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Errorf("content: got %q, want %q", string(got), content)
	}

	if _, err := os.Stat(filepath.Join(dir, key)); err != nil {
		t.Errorf("expected file at %s: %v", filepath.Join(dir, key), err)
	}
}

func TestLocal_Open_NotFound(t *testing.T) {
	t.Parallel()
	s := storage.NewLocal(t.TempDir())

	if _, err := s.Open(context.Background(), "does-not-exist"); err == nil {
		t.Error("expected error for missing key, got nil")
	}
}
