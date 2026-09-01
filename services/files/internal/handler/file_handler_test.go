package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/wingarc-ktq/speckit-workshop-server/packages/authjwt"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/handler"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

type fakeFileUsecase struct {
	uploadFn     func(context.Context, usecase.UploadInput) (*usecase.UploadOutput, error)
	listFn       func(context.Context, usecase.ListInput) (*usecase.ListOutput, error)
	getFn        func(context.Context, usecase.GetInput) (*domain.File, error)
	downloadFn   func(context.Context, usecase.DownloadInput) (*usecase.DownloadOutput, error)
	updateFn     func(context.Context, usecase.UpdateMetadataInput) (*domain.File, error)
	deleteFn     func(context.Context, usecase.DeleteInput) error
	deleteFilesFn func(context.Context, usecase.DeleteFilesInput) error
}

func (f *fakeFileUsecase) Upload(ctx context.Context, in usecase.UploadInput) (*usecase.UploadOutput, error) {
	if f.uploadFn != nil {
		return f.uploadFn(ctx, in)
	}
	return nil, nil
}

func (f *fakeFileUsecase) List(ctx context.Context, in usecase.ListInput) (*usecase.ListOutput, error) {
	if f.listFn != nil {
		return f.listFn(ctx, in)
	}
	return nil, nil
}

func (f *fakeFileUsecase) Get(ctx context.Context, in usecase.GetInput) (*domain.File, error) {
	if f.getFn != nil {
		return f.getFn(ctx, in)
	}
	return nil, nil
}

func (f *fakeFileUsecase) Download(ctx context.Context, in usecase.DownloadInput) (*usecase.DownloadOutput, error) {
	if f.downloadFn != nil {
		return f.downloadFn(ctx, in)
	}
	return nil, nil
}

func (f *fakeFileUsecase) UpdateMetadata(ctx context.Context, in usecase.UpdateMetadataInput) (*domain.File, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, in)
	}
	return nil, nil
}

func (f *fakeFileUsecase) Delete(ctx context.Context, in usecase.DeleteInput) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, in)
	}
	return nil
}

func (f *fakeFileUsecase) DeleteFiles(ctx context.Context, in usecase.DeleteFilesInput) error {
	if f.deleteFilesFn != nil {
		return f.deleteFilesFn(ctx, in)
	}
	return nil
}

func TestFileHandler_ListFiles(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	fileID := uuid.New()
	page := 1
	limit := 20

	tests := []struct {
		name       string
		params     gen.ListFilesParams
		setup      func(*fakeFileUsecase)
		wantStatus int
		wantCode   string
	}{
		{
			name: "success",
			params: gen.ListFilesParams{
				Page:  &page,
				Limit: &limit,
				Q:     strPtr("請求"),
			},
			setup: func(u *fakeFileUsecase) {
				u.listFn = func(_ context.Context, in usecase.ListInput) (*usecase.ListOutput, error) {
					if in.OwnerUserID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", in.OwnerUserID, ownerID) }
					if in.Page != 1 || in.Limit != 20 { t.Fatalf("page/limit: got %d/%d, want 1/20", in.Page, in.Limit) }
					if in.Keyword != "請求" { t.Fatalf("keyword: got %s, want %s", in.Keyword, "請求") }
					return &usecase.ListOutput{Files: []domain.File{{ID: fileID, OwnerUserID: ownerID, Name: "請求書_1.pdf", Size: 1024, MIMEType: "application/pdf"}}, Total: 1, Page: 1, Limit: 20}, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid pagination",
			params: gen.ListFilesParams{Page: intPtr(0), Limit: &limit},
			setup: func(u *fakeFileUsecase) {
				u.listFn = func(context.Context, usecase.ListInput) (*usecase.ListOutput, error) {
					return nil, domain.ErrInvalidPagination
				}
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := &fakeFileUsecase{}
			if tt.setup != nil { tt.setup(u) }
			h := handler.NewFileHandler(u)
			c, rec := newContext(http.MethodGet, "/api/v1/files", "")
			c.Set("userID", ownerID)

			if err := h.ListFiles(c, tt.params); err != nil { t.Fatal(err) }
			if rec.Code != tt.wantStatus { t.Fatalf("status: got %d, want %d", rec.Code, tt.wantStatus) }
			if tt.wantCode != "" {
				var body gen.ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil { t.Fatal(err) }
				if body.Code != tt.wantCode { t.Errorf("code: got %s, want %s", body.Code, tt.wantCode) }
				return
			}
			var body gen.FileListResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil { t.Fatal(err) }
			if len(body.Files) != 1 { t.Fatalf("files: got %d, want 1", len(body.Files)) }
		})
	}
}

func TestFileHandler_UpdateFile(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	fileID := uuid.New()
	t.Run("success", func(t *testing.T) {
		u := &fakeFileUsecase{updateFn: func(_ context.Context, in usecase.UpdateMetadataInput) (*domain.File, error) {
			if in.OwnerUserID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", in.OwnerUserID, ownerID) }
			if in.FileID != fileID { t.Fatalf("fileID: got %v, want %v", in.FileID, fileID) }
			if in.Name == nil || *in.Name != "updated.pdf" { t.Fatalf("name: got %v, want %q", in.Name, "updated.pdf") }
			if in.Description == nil || *in.Description != "new description" { t.Fatalf("description: got %v, want %q", in.Description, "new description") }
			if len(in.TagIDs) != 2 { t.Fatalf("tagIDs: got %d, want %d", len(in.TagIDs), 2) }
			return &domain.File{ID: fileID, OwnerUserID: ownerID, Name: "updated.pdf", MIMEType: "application/pdf", Description: strPtr("new description"), TagIDs: in.TagIDs, UploadedAt: time.Now()}, nil
		}}
		h := handler.NewFileHandler(u)
		jsonBody := `{"name":"updated.pdf","description":"new description","tagIds":["` + uuid.NewString() + `","` + uuid.NewString() + `"]}`
		c, rec := newContext(http.MethodPatch, "/api/v1/files/"+fileID.String(), jsonBody)
		c.Set("userID", ownerID)
		if err := h.UpdateFile(c, openapi_types.UUID(fileID)); err != nil { t.Fatal(err) }
		if rec.Code != http.StatusOK { t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK) }
	})

	t.Run("file not found", func(t *testing.T) {
		u := &fakeFileUsecase{updateFn: func(context.Context, usecase.UpdateMetadataInput) (*domain.File, error) {
			return nil, domain.ErrFileNotFound
		}}
		h := handler.NewFileHandler(u)
		c, rec := newContext(http.MethodPatch, "/api/v1/files/"+fileID.String(), `{"name":"updated.pdf"}`)
		c.Set("userID", ownerID)
		if err := h.UpdateFile(c, openapi_types.UUID(fileID)); err != nil { t.Fatal(err) }
		if rec.Code != http.StatusNotFound { t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNotFound) }
	})
}

func TestFileHandler_DeleteFile(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	fileID := uuid.New()

	t.Run("success", func(t *testing.T) {
		u := &fakeFileUsecase{deleteFn: func(_ context.Context, in usecase.DeleteInput) error {
			if in.OwnerUserID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", in.OwnerUserID, ownerID) }
			if in.FileID != fileID { t.Fatalf("fileID: got %v, want %v", in.FileID, fileID) }
			return nil
		}}
		h := handler.NewFileHandler(u)
		c, rec := newContext(http.MethodDelete, "/api/v1/files/"+fileID.String(), "")
		c.Set("userID", ownerID)
		if err := h.DeleteFile(c, openapi_types.UUID(fileID)); err != nil { t.Fatal(err) }
		if rec.Code != http.StatusNoContent { t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent) }
	})

	t.Run("file not found", func(t *testing.T) {
		u := &fakeFileUsecase{deleteFn: func(context.Context, usecase.DeleteInput) error {
			return domain.ErrFileNotFound
		}}
		h := handler.NewFileHandler(u)
		c, rec := newContext(http.MethodDelete, "/api/v1/files/"+fileID.String(), "")
		c.Set("userID", ownerID)
		if err := h.DeleteFile(c, openapi_types.UUID(fileID)); err != nil { t.Fatal(err) }
		if rec.Code != http.StatusNotFound { t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNotFound) }
	})
}

func TestFileHandler_BatchDeleteFiles(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	ids := []uuid.UUID{uuid.New(), uuid.New()}

	t.Run("success", func(t *testing.T) {
		u := &fakeFileUsecase{deleteFilesFn: func(_ context.Context, in usecase.DeleteFilesInput) error {
			if in.OwnerUserID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", in.OwnerUserID, ownerID) }
			if len(in.FileIDs) != len(ids) { t.Fatalf("len(fileIDs): got %d, want %d", len(in.FileIDs), len(ids)) }
			return nil
		}}
		h := handler.NewFileHandler(u)
		payload := `{"fileIds":["` + ids[0].String() + `","` + ids[1].String() + `"]}`
		c, rec := newContext(http.MethodPost, "/api/v1/files/batch-delete", payload)
		c.Set("userID", ownerID)
		if err := h.BatchDeleteFiles(c); err != nil { t.Fatal(err) }
		if rec.Code != http.StatusNoContent { t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent) }
	})

	t.Run("invalid request", func(t *testing.T) {
		u := &fakeFileUsecase{}
		h := handler.NewFileHandler(u)
		c, rec := newContext(http.MethodPost, "/api/v1/files/batch-delete", `{"fileIds":[]}`)
		c.Set("userID", ownerID)
		if err := h.BatchDeleteFiles(c); err != nil { t.Fatal(err) }
		if rec.Code != http.StatusBadRequest { t.Fatalf("status: got %d, want %d", rec.Code, http.StatusBadRequest) }
	})
}

func TestFileHandler_UploadFile(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	description := "2026年度の事業計画書"

	tests := []struct {
		name       string
		body       func(*multipart.Writer)
		setup      func(*fakeFileUsecase)
		wantStatus int
		wantCode   string
	}{
		{
			name: "success",
			body: func(w *multipart.Writer) {
				part, err := w.CreateFormFile("file", "report.pdf")
				if err != nil { t.Fatal(err) }
				if _, err := part.Write([]byte("pdf-content")); err != nil { t.Fatal(err) }
				if err := w.WriteField("description", description); err != nil { t.Fatal(err) }
			},
			setup: func(u *fakeFileUsecase) {
				u.uploadFn = func(_ context.Context, in usecase.UploadInput) (*usecase.UploadOutput, error) {
					if in.OwnerUserID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", in.OwnerUserID, ownerID) }
					if in.FileName != "report.pdf" { t.Fatalf("fileName: got %s, want %s", in.FileName, "report.pdf") }
					if in.Description == nil || *in.Description != description { t.Fatalf("description: got %v, want %s", in.Description, description) }
					return &usecase.UploadOutput{
						File: &domain.File{
							ID:          uuid.New(),
							OwnerUserID: ownerID,
							Name:        "report.pdf",
							Size:        11,
							MIMEType:    "application/pdf",
							Description: &description,
							UploadedAt:  time.Now(),
						},
						DownloadURL: "/api/v1/files/123/download",
					}, nil
				}
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "invalid file",
			body: func(w *multipart.Writer) {
				if err := w.WriteField("description", description); err != nil { t.Fatal(err) }
			},
			setup: func(u *fakeFileUsecase) {
				u.uploadFn = func(_ context.Context, in usecase.UploadInput) (*usecase.UploadOutput, error) {
					return nil, domain.ErrInvalidFile
				}
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name: "file too large",
			body: func(w *multipart.Writer) {
				part, err := w.CreateFormFile("file", "large.bin")
				if err != nil { t.Fatal(err) }
				if _, err := part.Write([]byte("x")); err != nil { t.Fatal(err) }
			},
			setup: func(u *fakeFileUsecase) {
				u.uploadFn = func(context.Context, usecase.UploadInput) (*usecase.UploadOutput, error) {
					return nil, domain.ErrFileTooLarge
				}
			},
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   "FILE_TOO_LARGE",
		},
		{
			name: "unexpected error",
			body: func(w *multipart.Writer) {
				part, err := w.CreateFormFile("file", "report.pdf")
				if err != nil { t.Fatal(err) }
				if _, err := part.Write([]byte("pdf-content")); err != nil { t.Fatal(err) }
			},
			setup: func(u *fakeFileUsecase) {
				u.uploadFn = func(context.Context, usecase.UploadInput) (*usecase.UploadOutput, error) {
					return nil, errors.New("boom")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := &fakeFileUsecase{}
			if tt.setup != nil { tt.setup(u) }
			h := handler.NewFileHandler(u)
			c, rec := newMultipartContext(http.MethodPost, "/api/v1/files", tt.body)
			c.Set("userID", ownerID)

			if err := h.UploadFile(c); err != nil {
				t.Fatal(err)
			}

			if rec.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantCode != "" {
				var body gen.ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Code != tt.wantCode {
					t.Errorf("code: got %s, want %s", body.Code, tt.wantCode)
				}
				return
			}
			var body gen.FileResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.File.Name != "report.pdf" {
				t.Errorf("file.name: got %s, want %s", body.File.Name, "report.pdf")
			}
		})
	}
}

func newContext(method, target, body string) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return echo.New().NewContext(req, rec), rec
}

func newMultipartContext(method, target string, builder func(*multipart.Writer)) (echo.Context, *httptest.ResponseRecorder) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	builder(w)
	if err := w.Close(); err != nil {
		panic(err)
	}

	req := httptest.NewRequest(method, target, &buf)
	req.Header.Set(echo.HeaderContentType, w.FormDataContentType())
	rec := httptest.NewRecorder()
	return echo.New().NewContext(req, rec), rec
}

func TestFileHandler_GetFile(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	fileID := uuid.New()
	desc := "請求書"

	t.Run("success", func(t *testing.T) {
		u := &fakeFileUsecase{
			getFn: func(_ context.Context, in usecase.GetInput) (*domain.File, error) {
				if in.OwnerUserID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", in.OwnerUserID, ownerID) }
				if in.FileID != fileID { t.Fatalf("fileID: got %v, want %v", in.FileID, fileID) }
				return &domain.File{ID: fileID, OwnerUserID: ownerID, Name: "請求_2026.pdf", MIMEType: "application/pdf", Description: &desc}, nil
			},
		}
		h := handler.NewFileHandler(u)
		c, rec := newContext(http.MethodGet, "/api/v1/files/"+fileID.String(), "")
		c.Set("userID", ownerID)

		if err := h.GetFile(c, openapi_types.UUID(fileID)); err != nil { t.Fatal(err) }
		if rec.Code != http.StatusOK { t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK) }
		var body gen.FileResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil { t.Fatal(err) }
		if body.File.Id != openapi_types.UUID(fileID) { t.Fatalf("file.id: got %v, want %v", body.File.Id, openapi_types.UUID(fileID)) }
	})

	t.Run("file not found", func(t *testing.T) {
		u := &fakeFileUsecase{getFn: func(context.Context, usecase.GetInput) (*domain.File, error) { return nil, domain.ErrFileNotFound }}
		h := handler.NewFileHandler(u)
		c, rec := newContext(http.MethodGet, "/api/v1/files/"+fileID.String(), "")
		c.Set("userID", ownerID)

		if err := h.GetFile(c, openapi_types.UUID(fileID)); err != nil { t.Fatal(err) }
		if rec.Code != http.StatusNotFound { t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNotFound) }
	})
}

func TestFileHandler_DownloadFile(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	fileID := uuid.New()
	desc := "請求書"
	u := &fakeFileUsecase{
		downloadFn: func(_ context.Context, in usecase.DownloadInput) (*usecase.DownloadOutput, error) {
			if in.OwnerUserID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", in.OwnerUserID, ownerID) }
			if in.FileID != fileID { t.Fatalf("fileID: got %v, want %v", in.FileID, fileID) }
			return &usecase.DownloadOutput{File: &domain.File{ID: fileID, OwnerUserID: ownerID, Name: "請求_2026.pdf", MIMEType: "application/pdf", Description: &desc}, Data: []byte("pdf-data")}, nil
		},
	}
	h := handler.NewFileHandler(u)
	c, rec := newContext(http.MethodGet, "/api/v1/files/"+fileID.String()+"/download", "")
	c.Set("userID", ownerID)

	if err := h.DownloadFile(c, openapi_types.UUID(fileID)); err != nil { t.Fatal(err) }
	if rec.Code != http.StatusOK { t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK) }
	if rec.Body.String() != "pdf-data" { t.Fatalf("body: got %q, want %q", rec.Body.String(), "pdf-data") }
	if got := rec.Header().Get("Content-Type"); got != "application/pdf" { t.Fatalf("content-type: got %q, want %q", got, "application/pdf") }
}

func TestAuthjwtContextSetMatchesHandler(t *testing.T) {
	t.Parallel()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/api/v1/files", strings.NewReader("")), httptest.NewRecorder())
	id := uuid.New()
	c.Set("userID", id)
	got, ok := authjwt.UserIDFromContext(c)
	if !ok {
		t.Fatal("userID not found in context")
	}
	if got != id {
		t.Fatalf("userID: got %v, want %v", got, id)
	}
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

var _ = time.Now
