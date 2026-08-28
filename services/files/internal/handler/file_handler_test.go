package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/handler"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase/mock"
)

func TestFileHandler_UploadFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		buildBody  func(t *testing.T) *multipart.Reader
		setup      func(*mock.MockFileUsecase)
		wantStatus int
		wantCode   string
	}{
		{
			name: "success",
			buildBody: func(t *testing.T) *multipart.Reader {
				return newMultipartReader(t, "invoice.pdf", []byte("dummy content"), "8月分の請求書")
			},
			setup: func(uc *mock.MockFileUsecase) {
				uc.EXPECT().UploadFile(gomock.Any(), gomock.Any()).Return(&domain.File{
					ID:       uuid.MustParse("3fa85f64-5717-4562-b3fc-2c963f66afa6"),
					Name:     "invoice.pdf",
					Size:     13,
					MimeType: "application/octet-stream",
					TagIDs:   []uuid.UUID{},
				}, nil)
			},
			wantStatus: 201,
		},
		{
			name: "file 未指定",
			buildBody: func(t *testing.T) *multipart.Reader {
				return newMultipartReader(t, "", nil, "説明のみ")
			},
			setup: func(uc *mock.MockFileUsecase) {
				uc.EXPECT().UploadFile(gomock.Any(), gomock.Any()).Return(nil, domain.ErrFileEmpty)
			},
			wantStatus: 400,
			wantCode:   "INVALID_PARAMETER",
		},
		{
			name: "10MB 超過",
			buildBody: func(t *testing.T) *multipart.Reader {
				return newMultipartReader(t, "big.bin", []byte("dummy content"), "")
			},
			setup: func(uc *mock.MockFileUsecase) {
				uc.EXPECT().UploadFile(gomock.Any(), gomock.Any()).Return(nil, domain.ErrFileTooLarge)
			},
			wantStatus: 413,
			wantCode:   "FILE_TOO_LARGE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock.NewMockFileUsecase(ctrl)
			tt.setup(uc)
			h := handler.NewFileHandler(uc)

			resp, err := h.UploadFile(context.Background(), gen.UploadFileRequestObject{
				Body: tt.buildBody(t),
			})
			if err != nil {
				t.Fatal(err)
			}

			rec := httptest.NewRecorder()
			if err := resp.VisitUploadFileResponse(rec); err != nil {
				t.Fatal(err)
			}
			if rec.Code != tt.wantStatus {
				t.Errorf("status code: got %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantCode != "" {
				var body gen.ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Code != tt.wantCode {
					t.Errorf("code: got %s, want %s", body.Code, tt.wantCode)
				}
			}
		})
	}
}

func TestFileHandler_GetFiles(t *testing.T) {
	t.Parallel()

	tagID := uuid.New()
	sampleFiles := []*domain.File{
		{ID: uuid.New(), Name: "invoice.pdf", TagIDs: []uuid.UUID{}},
		{ID: uuid.New(), Name: "report.docx", TagIDs: []uuid.UUID{}},
	}

	tests := []struct {
		name       string
		params     gen.GetFilesParams
		setup      func(*mock.MockFileUsecase)
		wantStatus int
		wantTotal  int
		wantPage   int
		wantLimit  int
		wantFiles  int
	}{
		{
			name:   "デフォルトページングで一覧取得",
			params: gen.GetFilesParams{},
			setup: func(uc *mock.MockFileUsecase) {
				uc.EXPECT().
					ListFiles(gomock.Any(), usecase.ListFilesParams{Page: 1, Limit: 20}).
					Return(sampleFiles, int64(2), nil)
			},
			wantStatus: 200,
			wantTotal:  2,
			wantPage:   1,
			wantLimit:  20,
			wantFiles:  2,
		},
		{
			name: "検索キーワードとタグIDsが usecase に渡される",
			params: gen.GetFilesParams{
				Search: strPtr("請求書"),
				TagIds: &[]openapi_types.UUID{openapi_types.UUID(tagID)},
			},
			setup: func(uc *mock.MockFileUsecase) {
				uc.EXPECT().
					ListFiles(gomock.Any(), usecase.ListFilesParams{Page: 1, Limit: 20, Search: "請求書", TagIDs: []uuid.UUID{tagID}}).
					Return([]*domain.File{sampleFiles[0]}, int64(1), nil)
			},
			wantStatus: 200,
			wantTotal:  1,
			wantPage:   1,
			wantLimit:  20,
			wantFiles:  1,
		},
		{
			name:   "指定したページ・件数が usecase に渡される",
			params: gen.GetFilesParams{Page: intPtr(2), Limit: intPtr(5)},
			setup: func(uc *mock.MockFileUsecase) {
				uc.EXPECT().
					ListFiles(gomock.Any(), usecase.ListFilesParams{Page: 2, Limit: 5}).
					Return([]*domain.File{}, int64(0), nil)
			},
			wantStatus: 200,
			wantTotal:  0,
			wantPage:   2,
			wantLimit:  5,
			wantFiles:  0,
		},
		{
			name:   "一致しない場合は空配列とtotal 0",
			params: gen.GetFilesParams{Search: strPtr("存在しないキーワード")},
			setup: func(uc *mock.MockFileUsecase) {
				uc.EXPECT().
					ListFiles(gomock.Any(), usecase.ListFilesParams{Page: 1, Limit: 20, Search: "存在しないキーワード"}).
					Return([]*domain.File{}, int64(0), nil)
			},
			wantStatus: 200,
			wantTotal:  0,
			wantPage:   1,
			wantLimit:  20,
			wantFiles:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock.NewMockFileUsecase(ctrl)
			tt.setup(uc)
			h := handler.NewFileHandler(uc)

			resp, err := h.GetFiles(context.Background(), gen.GetFilesRequestObject{Params: tt.params})
			if err != nil {
				t.Fatal(err)
			}

			rec := httptest.NewRecorder()
			if err := resp.VisitGetFilesResponse(rec); err != nil {
				t.Fatal(err)
			}
			if rec.Code != tt.wantStatus {
				t.Errorf("status code: got %d, want %d", rec.Code, tt.wantStatus)
			}

			var body gen.FileListResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Total != tt.wantTotal {
				t.Errorf("total: got %d, want %d", body.Total, tt.wantTotal)
			}
			if body.Page != tt.wantPage {
				t.Errorf("page: got %d, want %d", body.Page, tt.wantPage)
			}
			if body.Limit != tt.wantLimit {
				t.Errorf("limit: got %d, want %d", body.Limit, tt.wantLimit)
			}
			if len(body.Files) != tt.wantFiles {
				t.Errorf("files length: got %d, want %d", len(body.Files), tt.wantFiles)
			}
			if body.Files == nil {
				t.Error("files: got nil, want non-nil (possibly empty) slice")
			}
		})
	}
}

func TestFileHandler_GetFile(t *testing.T) {
	t.Parallel()

	existing := &domain.File{
		ID:       uuid.MustParse("3fa85f64-5717-4562-b3fc-2c963f66afa6"),
		Name:     "invoice.pdf",
		MimeType: "application/pdf",
		TagIDs:   []uuid.UUID{},
	}

	tests := []struct {
		name       string
		setup      func(*mock.MockFileUsecase)
		wantStatus int
		wantCode   string
	}{
		{
			name: "存在するIDなら200",
			setup: func(uc *mock.MockFileUsecase) {
				uc.EXPECT().GetFile(gomock.Any(), existing.ID).Return(existing, nil)
			},
			wantStatus: 200,
		},
		{
			name: "存在しないIDは404",
			setup: func(uc *mock.MockFileUsecase) {
				uc.EXPECT().GetFile(gomock.Any(), existing.ID).Return(nil, domain.ErrFileNotFound)
			},
			wantStatus: 404,
			wantCode:   "FILE_NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock.NewMockFileUsecase(ctrl)
			tt.setup(uc)
			h := handler.NewFileHandler(uc)

			resp, err := h.GetFile(context.Background(), gen.GetFileRequestObject{FileId: openapi_types.UUID(existing.ID)})
			if err != nil {
				t.Fatal(err)
			}

			rec := httptest.NewRecorder()
			if err := resp.VisitGetFileResponse(rec); err != nil {
				t.Fatal(err)
			}
			if rec.Code != tt.wantStatus {
				t.Errorf("status code: got %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantCode != "" {
				var body gen.ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Code != tt.wantCode {
					t.Errorf("code: got %s, want %s", body.Code, tt.wantCode)
				}
			}
		})
	}
}

func TestFileHandler_DownloadFileContent(t *testing.T) {
	t.Parallel()

	const content = "%PDF-1.4 dummy content"
	existing := &domain.File{
		ID:       uuid.MustParse("3fa85f64-5717-4562-b3fc-2c963f66afa6"),
		Name:     "invoice.pdf",
		Size:     int64(len(content)),
		MimeType: "application/pdf",
	}

	tests := []struct {
		name       string
		setup      func(*mock.MockFileUsecase)
		wantStatus int
		wantBody   string
		wantCode   string
	}{
		{
			name: "存在するIDなら200でバイナリが返る",
			setup: func(uc *mock.MockFileUsecase) {
				uc.EXPECT().DownloadFile(gomock.Any(), existing.ID).Return(&usecase.DownloadOutput{
					File:    existing,
					Content: io.NopCloser(strings.NewReader(content)),
				}, nil)
			},
			wantStatus: 200,
			wantBody:   content,
		},
		{
			name: "存在しないIDは404",
			setup: func(uc *mock.MockFileUsecase) {
				uc.EXPECT().DownloadFile(gomock.Any(), existing.ID).Return(nil, domain.ErrFileNotFound)
			},
			wantStatus: 404,
			wantCode:   "FILE_NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock.NewMockFileUsecase(ctrl)
			tt.setup(uc)
			h := handler.NewFileHandler(uc)

			resp, err := h.DownloadFileContent(context.Background(), gen.DownloadFileContentRequestObject{FileId: openapi_types.UUID(existing.ID)})
			if err != nil {
				t.Fatal(err)
			}

			rec := httptest.NewRecorder()
			if err := resp.VisitDownloadFileContentResponse(rec); err != nil {
				t.Fatal(err)
			}
			if rec.Code != tt.wantStatus {
				t.Errorf("status code: got %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" {
				if rec.Body.String() != tt.wantBody {
					t.Errorf("body: got %q, want %q", rec.Body.String(), tt.wantBody)
				}
				// ダウンロード時にファイル名がブラウザへ伝わっているかを確認する.
				disposition := rec.Header().Get("Content-Disposition")
				if !strings.Contains(disposition, existing.Name) {
					t.Errorf("Content-Disposition: got %q, want it to contain %q", disposition, existing.Name)
				}
			}
			if tt.wantCode != "" {
				var body gen.ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Code != tt.wantCode {
					t.Errorf("code: got %s, want %s", body.Code, tt.wantCode)
				}
			}
		})
	}
}

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }

// newMultipartReader は file/description パートを含む multipart.Reader を組み立てる.
// fileName が空文字の場合は file パート自体を含めない（file 未指定を再現する）.
func newMultipartReader(t *testing.T, fileName string, content []byte, description string) *multipart.Reader {
	t.Helper()
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	if fileName != "" {
		fw, err := w.CreateFormFile("file", fileName)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if description != "" {
		if err := w.WriteField("description", description); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	return multipart.NewReader(buf, w.Boundary())
}
