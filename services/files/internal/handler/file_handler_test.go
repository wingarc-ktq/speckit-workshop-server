package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	repo "github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo"
	storage "github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/storage"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

func TestUploadFileHandler(t *testing.T) {
	fileRepo := repo.NewInMemoryFileRepository()
	store, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	uc := usecase.NewFileUsecase(fileRepo, store, domain.MaxFileSize)
	h := NewFilesHandler(uc)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "report.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("hello world")); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("description", "monthly report"); err != nil {
		t.Fatal(err)
	}
	tagID := uuid.NewString()
	if err := writer.WriteField("tagIds", tagID); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/files", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	if err := h.UploadFile(ctx); err != nil {
		t.Fatalf("UploadFile() returned error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if got := rec.Body.String(); got == "" || !bytes.Contains([]byte(got), []byte("report.pdf")) {
		t.Fatalf("response body = %q, want report.pdf", got)
	}
}

func TestUploadFileHandlerRejectsMissingFile(t *testing.T) {
	fileRepo := repo.NewInMemoryFileRepository()
	store, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	uc := usecase.NewFileUsecase(fileRepo, store, domain.MaxFileSize)
	h := NewFilesHandler(uc)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("description", "test"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/files", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	if err := h.UploadFile(ctx); err != nil {
		t.Fatalf("UploadFile() returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
