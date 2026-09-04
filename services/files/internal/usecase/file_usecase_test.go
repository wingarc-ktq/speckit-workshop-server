package usecase

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	repo "github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo"
	storage "github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/storage"
)

func TestUploadFileSuccess(t *testing.T) {
	repoImpl := repo.NewInMemoryFileRepository()
	storageImpl, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	uc := NewFileUsecase(repoImpl, storageImpl, domain.MaxFileSize)
	tagID := uuid.New()

	file, err := uc.UploadFile(context.Background(), UploadInput{
		FileName:    "report.pdf",
		MIMEType:    "application/pdf",
		Description: "monthly report",
		Size:        int64(len("hello world")),
		TagIDs:      []uuid.UUID{tagID},
		Reader:      bytes.NewReader([]byte("hello world")),
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if file.Name != "report.pdf" {
		t.Fatalf("file name = %q, want %q", file.Name, "report.pdf")
	}
	if file.StorageKey == "" {
		t.Fatal("storage key should be set")
	}
	if len(file.TagIDs) != 1 || file.TagIDs[0] != tagID {
		t.Fatalf("tag ids = %v, want [%v]", file.TagIDs, tagID)
	}
	if _, err := repoImpl.GetByID(context.Background(), file.ID); err != nil {
		t.Fatal("repository should store uploaded file")
	}
}

func TestUploadFileRejectsOversizedPayload(t *testing.T) {
	repoImpl := repo.NewInMemoryFileRepository()
	storageImpl, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	uc := NewFileUsecase(repoImpl, storageImpl, domain.MaxFileSize)

	_, err = uc.UploadFile(context.Background(), UploadInput{
		FileName: "big.bin",
		MIMEType: "application/octet-stream",
		Size:     domain.MaxFileSize + 1,
		Reader:   bytes.NewReader(make([]byte, domain.MaxFileSize+1)),
	})
	if err == nil {
		t.Fatal("UploadFile() expected oversized error")
	}
	if err.Error() == "" {
		t.Fatal("expected non-empty error")
	}
}

func TestListFilesAndDownloadFile(t *testing.T) {
	repoImpl := repo.NewInMemoryFileRepository()
	storageImpl, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	uc := NewFileUsecase(repoImpl, storageImpl, domain.MaxFileSize)
	tagID := uuid.New()

	file, err := uc.UploadFile(context.Background(), UploadInput{
		FileName:    "report.pdf",
		MIMEType:    "application/pdf",
		Description: "monthly report",
		Size:        int64(len("hello world")),
		TagIDs:      []uuid.UUID{tagID},
		Reader:      bytes.NewReader([]byte("hello world")),
	})
	if err != nil {
		t.Fatal(err)
	}

	items, total, err := uc.ListFiles(context.Background(), ListFilesInput{Page: 1, Limit: 20, Name: "report", TagIDs: []uuid.UUID{tagID}})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(items) != 1 || items[0].ID != file.ID {
		t.Fatalf("items = %#v, want one file with id %s", items, file.ID)
	}

	got, reader, err := uc.DownloadFile(context.Background(), file.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	payload, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != file.ID {
		t.Fatalf("download file id = %s, want %s", got.ID, file.ID)
	}
	if string(payload) != "hello world" {
		t.Fatalf("download payload = %q, want %q", payload, "hello world")
	}
}
