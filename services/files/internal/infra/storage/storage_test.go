package storage

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
)

func TestLocalStorageSaveAndOpen(t *testing.T) {
	root := t.TempDir()
	store, err := NewLocalStorage(root)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	key, err := store.Save(context.Background(), "hello.txt", bytes.NewReader([]byte("hello world")), domain.MaxFileSize)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if key == "" {
		t.Fatal("Save() returned empty key")
	}
	f, err := store.Open(context.Background(), key)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer f.Close()
	payload, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(payload) != "hello world" {
		t.Fatalf("payload = %q, want %q", string(payload), "hello world")
	}
}

func TestLocalStorageRejectsTraversalKey(t *testing.T) {
	root := t.TempDir()
	store, err := NewLocalStorage(root)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	if _, err := store.Open(context.Background(), "../etc/passwd"); err == nil {
		t.Fatal("Open() expected traversal rejection")
	}
}
