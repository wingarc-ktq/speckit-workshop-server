package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase/mock"
)

func TestFileUsecase_UploadFile(t *testing.T) {
	t.Parallel()

	const content = "dummy pdf content"

	tests := []struct {
		name     string
		input    usecase.UploadFileInput
		setup    func(*mock.MockFileRepository, *mock.MockFileStorage)
		wantErr  error
		wantSize int64
	}{
		{
			name: "success",
			input: usecase.UploadFileInput{
				Name:        "invoice.pdf",
				MimeType:    "application/pdf",
				Description: "8月分の請求書",
				Content:     strings.NewReader(content),
			},
			setup: func(r *mock.MockFileRepository, s *mock.MockFileStorage) {
				s.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				r.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantSize: int64(len(content)),
		},
		{
			name: "file 未指定",
			input: usecase.UploadFileInput{
				Name: "empty.pdf",
			},
			setup:   func(_ *mock.MockFileRepository, _ *mock.MockFileStorage) {},
			wantErr: domain.ErrFileEmpty,
		},
		{
			name: "10MB 超過",
			input: usecase.UploadFileInput{
				Name:    "big.bin",
				Content: bytes.NewReader(make([]byte, usecase.MaxFileSize+1)),
			},
			setup:   func(_ *mock.MockFileRepository, _ *mock.MockFileStorage) {},
			wantErr: domain.ErrFileTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mock.NewMockFileRepository(ctrl)
			storage := mock.NewMockFileStorage(ctrl)
			tt.setup(repo, storage)
			uc := usecase.NewFileUsecase(repo, storage)

			file, err := uc.UploadFile(context.Background(), tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("err: got %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if file.Name != tt.input.Name {
				t.Errorf("name: got %s, want %s", file.Name, tt.input.Name)
			}
			if file.Size != tt.wantSize {
				t.Errorf("size: got %d, want %d", file.Size, tt.wantSize)
			}
			if file.TagIDs == nil {
				t.Error("tagIDs: got nil, want non-nil empty slice")
			}
			if file.StorageKey == "" {
				t.Error("storageKey: got empty, want non-empty")
			}
		})
	}
}

func TestFileUsecase_ListFiles(t *testing.T) {
	t.Parallel()

	sampleFiles := []*domain.File{
		{ID: uuid.New(), Name: "invoice.pdf"},
		{ID: uuid.New(), Name: "report.docx"},
	}
	tagID := uuid.New()

	tests := []struct {
		name      string
		params    usecase.ListFilesParams
		setup     func(*mock.MockFileRepository)
		wantFiles []*domain.File
		wantTotal int64
		wantErr   error
	}{
		{
			name:   "デフォルトページングで一覧を取得",
			params: usecase.ListFilesParams{Page: 1, Limit: 20},
			setup: func(r *mock.MockFileRepository) {
				r.EXPECT().
					List(gomock.Any(), usecase.ListFilesParams{Page: 1, Limit: 20}).
					Return(sampleFiles, int64(2), nil)
			},
			wantFiles: sampleFiles,
			wantTotal: 2,
		},
		{
			name:   "検索キーワードとタグIDsがそのまま委譲される",
			params: usecase.ListFilesParams{Page: 1, Limit: 20, Search: "請求書", TagIDs: []uuid.UUID{tagID}},
			setup: func(r *mock.MockFileRepository) {
				r.EXPECT().
					List(gomock.Any(), usecase.ListFilesParams{Page: 1, Limit: 20, Search: "請求書", TagIDs: []uuid.UUID{tagID}}).
					Return([]*domain.File{sampleFiles[0]}, int64(1), nil)
			},
			wantFiles: []*domain.File{sampleFiles[0]},
			wantTotal: 1,
		},
		{
			name:   "一致しない場合は0件",
			params: usecase.ListFilesParams{Page: 1, Limit: 20, Search: "存在しないキーワード"},
			setup: func(r *mock.MockFileRepository) {
				r.EXPECT().
					List(gomock.Any(), usecase.ListFilesParams{Page: 1, Limit: 20, Search: "存在しないキーワード"}).
					Return([]*domain.File{}, int64(0), nil)
			},
			wantFiles: []*domain.File{},
			wantTotal: 0,
		},
		{
			name:   "リポジトリのエラーはそのまま返す",
			params: usecase.ListFilesParams{Page: 1, Limit: 20},
			setup: func(r *mock.MockFileRepository) {
				r.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, int64(0), errors.New("db error"))
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mock.NewMockFileRepository(ctrl)
			storage := mock.NewMockFileStorage(ctrl)
			tt.setup(repo)
			uc := usecase.NewFileUsecase(repo, storage)

			files, total, err := uc.ListFiles(context.Background(), tt.params)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if total != tt.wantTotal {
				t.Errorf("total: got %d, want %d", total, tt.wantTotal)
			}
			if len(files) != len(tt.wantFiles) {
				t.Fatalf("files length: got %d, want %d", len(files), len(tt.wantFiles))
			}
			for i, f := range files {
				if f.ID != tt.wantFiles[i].ID {
					t.Errorf("files[%d].ID: got %v, want %v", i, f.ID, tt.wantFiles[i].ID)
				}
			}
		})
	}
}

func TestFileUsecase_GetFile(t *testing.T) {
	t.Parallel()

	existing := &domain.File{ID: uuid.New(), Name: "invoice.pdf"}

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(*mock.MockFileRepository)
		wantErr error
	}{
		{
			name: "存在するIDなら取得できる",
			id:   existing.ID,
			setup: func(r *mock.MockFileRepository) {
				r.EXPECT().FindByID(gomock.Any(), existing.ID).Return(existing, nil)
			},
		},
		{
			name: "存在しないIDはErrFileNotFound",
			id:   uuid.New(),
			setup: func(r *mock.MockFileRepository) {
				r.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(nil, domain.ErrFileNotFound)
			},
			wantErr: domain.ErrFileNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mock.NewMockFileRepository(ctrl)
			storage := mock.NewMockFileStorage(ctrl)
			tt.setup(repo)
			uc := usecase.NewFileUsecase(repo, storage)

			file, err := uc.GetFile(context.Background(), tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("err: got %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if file.ID != existing.ID {
				t.Errorf("id: got %v, want %v", file.ID, existing.ID)
			}
		})
	}
}

func TestFileUsecase_DownloadFile(t *testing.T) {
	t.Parallel()

	existing := &domain.File{ID: uuid.New(), Name: "invoice.pdf", StorageKey: "storage-key"}
	const content = "dummy pdf bytes"

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(*mock.MockFileRepository, *mock.MockFileStorage)
		wantErr error
	}{
		{
			name: "存在するIDならストレージから読み取れる",
			id:   existing.ID,
			setup: func(r *mock.MockFileRepository, s *mock.MockFileStorage) {
				r.EXPECT().FindByID(gomock.Any(), existing.ID).Return(existing, nil)
				s.EXPECT().Open(gomock.Any(), existing.StorageKey).Return(io.NopCloser(strings.NewReader(content)), nil)
			},
		},
		{
			// メタデータが無ければストレージには触らない（先に FindByID で存在確認するため）.
			name: "存在しないIDはストレージを開かずErrFileNotFound",
			id:   uuid.New(),
			setup: func(r *mock.MockFileRepository, _ *mock.MockFileStorage) {
				r.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(nil, domain.ErrFileNotFound)
			},
			wantErr: domain.ErrFileNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mock.NewMockFileRepository(ctrl)
			storage := mock.NewMockFileStorage(ctrl)
			tt.setup(repo, storage)
			uc := usecase.NewFileUsecase(repo, storage)

			out, err := uc.DownloadFile(context.Background(), tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("err: got %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer out.Content.Close()

			got, err := io.ReadAll(out.Content)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != content {
				t.Errorf("content: got %q, want %q", string(got), content)
			}
		})
	}
}
