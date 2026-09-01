package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

var errStorageFailure = errors.New("storage failure")

type fakeFileRepository struct {
	createFn    func(context.Context, *domain.File) (*domain.File, error)
	listFn      func(context.Context, uuid.UUID, string, int, int) ([]domain.File, error)
	countFn     func(context.Context, uuid.UUID, string) (int, error)
	getByIDFn   func(context.Context, uuid.UUID, uuid.UUID) (*domain.File, error)
	updateFn    func(context.Context, uuid.UUID, uuid.UUID, string, *string) (*domain.File, error)
	deleteFn    func(context.Context, uuid.UUID, uuid.UUID) error
	deleteByIDs func(context.Context, uuid.UUID, []uuid.UUID) error
}

func (f *fakeFileRepository) Create(ctx context.Context, file *domain.File) (*domain.File, error) {
	return f.createFn(ctx, file)
}

func (f *fakeFileRepository) List(ctx context.Context, ownerUserID uuid.UUID, keyword string, offset, limit int) ([]domain.File, error) {
	if f.listFn != nil {
		return f.listFn(ctx, ownerUserID, keyword, offset, limit)
	}
	return nil, nil
}

func (f *fakeFileRepository) Count(ctx context.Context, ownerUserID uuid.UUID, keyword string) (int, error) {
	if f.countFn != nil {
		return f.countFn(ctx, ownerUserID, keyword)
	}
	return 0, nil
}

func (f *fakeFileRepository) GetByID(ctx context.Context, ownerUserID, fileID uuid.UUID) (*domain.File, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, ownerUserID, fileID)
	}
	return nil, nil
}

func (f *fakeFileRepository) UpdateMetadata(ctx context.Context, ownerUserID, fileID uuid.UUID, name string, description *string, tagIDs []uuid.UUID) (*domain.File, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, ownerUserID, fileID, name, description)
	}
	return nil, nil
}

func (f *fakeFileRepository) Delete(ctx context.Context, ownerUserID, fileID uuid.UUID) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, ownerUserID, fileID)
	}
	return nil
}

func (f *fakeFileRepository) DeleteByIDs(ctx context.Context, ownerUserID uuid.UUID, fileIDs []uuid.UUID) error {
	if f.deleteByIDs != nil {
		return f.deleteByIDs(ctx, ownerUserID, fileIDs)
	}
	return nil
}

type fakeFileStorage struct {
	saveFn       func(context.Context, domain.FileContent) (*domain.StoredFile, error)
	openByFileFn func(context.Context, *domain.File) ([]byte, error)
	deleteFn     func(context.Context, *domain.StoredFile) error
	deleteByFile func(context.Context, *domain.File) error
}

func (f *fakeFileStorage) Save(ctx context.Context, content domain.FileContent) (*domain.StoredFile, error) {
	if f.saveFn != nil {
		return f.saveFn(ctx, content)
	}
	return &domain.StoredFile{
		ID:       uuid.New(),
		Name:     content.OriginalName,
		Path:     "/tmp/test.bin",
		Size:     content.Size,
		MIMEType: content.MIMEType,
		CreatedAt: time.Now(),
	}, nil
}

func (f *fakeFileStorage) Open(ctx context.Context, storedFile *domain.StoredFile) ([]byte, error) {
	return nil, nil
}

func (f *fakeFileStorage) OpenByFile(ctx context.Context, file *domain.File) ([]byte, error) {
	if f.openByFileFn != nil {
		return f.openByFileFn(ctx, file)
	}
	return nil, nil
}

func (f *fakeFileStorage) Delete(ctx context.Context, storedFile *domain.StoredFile) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, storedFile)
	}
	return nil
}

func (f *fakeFileStorage) DeleteByFile(ctx context.Context, file *domain.File) error {
	if f.deleteByFile != nil {
		return f.deleteByFile(ctx, file)
	}
	return nil
}

func TestFileUsecase_List(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	files := []domain.File{
		{ID: uuid.New(), OwnerUserID: ownerID, Name: "請求書_1.pdf", Size: 1024, MIMEType: "application/pdf"},
		{ID: uuid.New(), OwnerUserID: ownerID, Name: "請求書_2.pdf", Size: 2048, MIMEType: "application/pdf"},
	}

	tests := []struct {
		name    string
		input   usecase.ListInput
		repo    func(*fakeFileRepository)
		wantErr error
		wantLen int
		wantPage int
		wantLimit int
	}{
		{
			name: "success",
			input: usecase.ListInput{OwnerUserID: ownerID, Page: 1, Limit: 20, Keyword: "請求"},
			repo: func(r *fakeFileRepository) {
				r.listFn = func(_ context.Context, gotOwnerID uuid.UUID, keyword string, offset, limit int) ([]domain.File, error) {
					if gotOwnerID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", gotOwnerID, ownerID) }
					if keyword != "請求" { t.Fatalf("keyword: got %s, want %s", keyword, "請求") }
					if offset != 0 || limit != 20 { t.Fatalf("offset/limit: got %d/%d, want 0/20", offset, limit) }
					return files, nil
				}
				r.countFn = func(_ context.Context, gotOwnerID uuid.UUID, keyword string) (int, error) {
					if gotOwnerID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", gotOwnerID, ownerID) }
					if keyword != "請求" { t.Fatalf("keyword: got %s, want %s", keyword, "請求") }
					return len(files), nil
				}
			},
			wantLen: 2,
			wantPage: 1,
			wantLimit: 20,
		},
		{
			name: "invalid pagination",
			input: usecase.ListInput{OwnerUserID: ownerID, Page: 0, Limit: 20},
			wantErr: domain.ErrInvalidPagination,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &fakeFileRepository{}
			if tt.repo != nil { tt.repo(repo) }

			uc := usecase.NewFileUsecase(repo, nil, 5*1024*1024)
			out, err := uc.List(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil || !errors.Is(err, tt.wantErr) {
					t.Fatalf("err: got %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil { t.Fatal(err) }
			if out == nil { t.Fatal("output is nil") }
			if len(out.Files) != tt.wantLen { t.Fatalf("len(files): got %d, want %d", len(out.Files), tt.wantLen) }
			if out.Page != tt.wantPage { t.Fatalf("page: got %d, want %d", out.Page, tt.wantPage) }
			if out.Limit != tt.wantLimit { t.Fatalf("limit: got %d, want %d", out.Limit, tt.wantLimit) }
		})
	}
}

func TestFileUsecase_GetAndDownload(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	fileID := uuid.New()
	desc := "請求書"
	file := &domain.File{
		ID:          fileID,
		OwnerUserID: ownerID,
		Name:        "請求_2026.pdf",
		Size:        1024,
		MIMEType:    "application/pdf",
		Description: &desc,
		UploadedAt:  time.Now(),
	}

	t.Run("get success", func(t *testing.T) {
		repo := &fakeFileRepository{getByIDFn: func(_ context.Context, gotOwnerID, gotFileID uuid.UUID) (*domain.File, error) {
			if gotOwnerID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", gotOwnerID, ownerID) }
			if gotFileID != fileID { t.Fatalf("fileID: got %v, want %v", gotFileID, fileID) }
			return file, nil
		}}
		storage := &fakeFileStorage{}
		uc := usecase.NewFileUsecase(repo, storage, 5*1024*1024)
		got, err := uc.Get(context.Background(), usecase.GetInput{OwnerUserID: ownerID, FileID: fileID})
		if err != nil { t.Fatal(err) }
		if got == nil || got.ID != fileID { t.Fatalf("file: got %+v, want id %v", got, fileID) }
	})

	t.Run("download success", func(t *testing.T) {
		repo := &fakeFileRepository{getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.File, error) { return file, nil }}
		storage := &fakeFileStorage{openByFileFn: func(_ context.Context, got *domain.File) ([]byte, error) {
			if got == nil || got.ID != fileID { t.Fatalf("file: got %+v, want id %v", got, fileID) }
			return []byte("pdf-data"), nil
		}}
		uc := usecase.NewFileUsecase(repo, storage, 5*1024*1024)
		out, err := uc.Download(context.Background(), usecase.DownloadInput{OwnerUserID: ownerID, FileID: fileID})
		if err != nil { t.Fatal(err) }
		if out == nil || out.File == nil || out.File.ID != fileID { t.Fatalf("output: got %+v, want file id %v", out, fileID) }
		if string(out.Data) != "pdf-data" { t.Fatalf("data: got %q, want %q", string(out.Data), "pdf-data") }
	})

	t.Run("file not found", func(t *testing.T) {
		repo := &fakeFileRepository{getByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*domain.File, error) {
			return nil, domain.ErrFileNotFound
		}}
		storage := &fakeFileStorage{}
		uc := usecase.NewFileUsecase(repo, storage, 5*1024*1024)
		_, err := uc.Get(context.Background(), usecase.GetInput{OwnerUserID: ownerID, FileID: fileID})
		if !errors.Is(err, domain.ErrFileNotFound) { t.Fatalf("err: got %v, want %v", err, domain.ErrFileNotFound) }
	})
}

func TestFileUsecase_UpdateMetadata(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	fileID := uuid.New()
	current := &domain.File{
		ID:          fileID,
		OwnerUserID: ownerID,
		Name:        "old.pdf",
		MIMEType:    "application/pdf",
		Description: strPtr("old desc"),
		TagIDs:      []uuid.UUID{uuid.New()},
		UploadedAt:  time.Now(),
	}
	newName := "renewed.pdf"
	newDescription := "new desc"
	newTagIDs := []uuid.UUID{uuid.New(), uuid.New()}

	t.Run("success", func(t *testing.T) {
		repo := &fakeFileRepository{getByIDFn: func(_ context.Context, gotOwnerID, gotFileID uuid.UUID) (*domain.File, error) {
			if gotOwnerID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", gotOwnerID, ownerID) }
			if gotFileID != fileID { t.Fatalf("fileID: got %v, want %v", gotFileID, fileID) }
			return current, nil
		}, updateFn: func(_ context.Context, gotOwnerID, gotFileID uuid.UUID, name string, description *string) (*domain.File, error) {
			if gotOwnerID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", gotOwnerID, ownerID) }
			if gotFileID != fileID { t.Fatalf("fileID: got %v, want %v", gotFileID, fileID) }
			if name != newName { t.Fatalf("name: got %s, want %s", name, newName) }
			if description == nil || *description != newDescription { t.Fatalf("description: got %v, want %s", description, newDescription) }
			return &domain.File{ID: fileID, OwnerUserID: ownerID, Name: newName, MIMEType: "application/pdf", Description: &newDescription, TagIDs: newTagIDs, UploadedAt: time.Now()}, nil
		}}
		storage := &fakeFileStorage{}
		uc := usecase.NewFileUsecase(repo, storage, 5*1024*1024)
		got, err := uc.UpdateMetadata(context.Background(), usecase.UpdateMetadataInput{
			OwnerUserID: ownerID,
			FileID:      fileID,
			Name:        &newName,
			Description: &newDescription,
			TagIDs:      newTagIDs,
		})
		if err != nil { t.Fatal(err) }
		if got == nil || got.Name != newName { t.Fatalf("returned name: got %q, want %q", got.Name, newName) }
		if got.Description == nil || *got.Description != newDescription { t.Fatalf("returned description: got %v, want %q", got.Description, newDescription) }
		if len(got.TagIDs) != len(newTagIDs) { t.Fatalf("tag count: got %d, want %d", len(got.TagIDs), len(newTagIDs)) }
	})

	t.Run("file not found", func(t *testing.T) {
		repo := &fakeFileRepository{getByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*domain.File, error) {
			return nil, domain.ErrFileNotFound
		}}
		storage := &fakeFileStorage{}
		uc := usecase.NewFileUsecase(repo, storage, 5*1024*1024)
		_, err := uc.UpdateMetadata(context.Background(), usecase.UpdateMetadataInput{
			OwnerUserID: ownerID,
			FileID:      fileID,
			Name:        &newName,
		})
		if !errors.Is(err, domain.ErrFileNotFound) { t.Fatalf("err: got %v, want %v", err, domain.ErrFileNotFound) }
	})
}

func strPtr(s string) *string {
	return &s
}

func TestFileUsecase_Delete(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	fileID := uuid.New()
	file := &domain.File{ID: fileID, OwnerUserID: ownerID, Name: "report.pdf", MIMEType: "application/pdf"}

	t.Run("single delete success", func(t *testing.T) {
		repo := &fakeFileRepository{
			getByIDFn: func(_ context.Context, gotOwnerID, gotFileID uuid.UUID) (*domain.File, error) {
				if gotOwnerID != ownerID || gotFileID != fileID { t.Fatalf("unexpected lookup: owner=%v file=%v", gotOwnerID, gotFileID) }
				return file, nil
			},
			deleteFn: func(_ context.Context, gotOwnerID, gotFileID uuid.UUID) error {
				if gotOwnerID != ownerID || gotFileID != fileID { t.Fatalf("unexpected delete: owner=%v file=%v", gotOwnerID, gotFileID) }
				return nil
			},
		}
		storage := &fakeFileStorage{deleteByFile: func(_ context.Context, got *domain.File) error {
			if got == nil || got.ID != fileID { t.Fatalf("storage file: got %+v, want id %v", got, fileID) }
			return nil
		}}
		uc := usecase.NewFileUsecase(repo, storage, 5*1024*1024)
		if err := uc.Delete(context.Background(), usecase.DeleteInput{OwnerUserID: ownerID, FileID: fileID}); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
	})

	t.Run("batch delete success", func(t *testing.T) {
		ids := []uuid.UUID{uuid.New(), uuid.New()}
		repo := &fakeFileRepository{
			getByIDFn: func(_ context.Context, gotOwnerID, gotFileID uuid.UUID) (*domain.File, error) {
				if gotOwnerID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", gotOwnerID, ownerID) }
				return &domain.File{ID: gotFileID, OwnerUserID: ownerID, Name: "report.pdf", MIMEType: "application/pdf"}, nil
			},
			deleteByIDs: func(_ context.Context, gotOwnerID uuid.UUID, gotIDs []uuid.UUID) error {
				if gotOwnerID != ownerID { t.Fatalf("ownerUserID: got %v, want %v", gotOwnerID, ownerID) }
				if len(gotIDs) != len(ids) { t.Fatalf("len(ids): got %d, want %d", len(gotIDs), len(ids)) }
				return nil
			},
		}
		storage := &fakeFileStorage{deleteByFile: func(_ context.Context, got *domain.File) error { return nil }}
		uc := usecase.NewFileUsecase(repo, storage, 5*1024*1024)
		if err := uc.DeleteFiles(context.Background(), usecase.DeleteFilesInput{OwnerUserID: ownerID, FileIDs: ids}); err != nil {
			t.Fatalf("DeleteFiles() error = %v", err)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		repo := &fakeFileRepository{getByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*domain.File, error) {
			return nil, domain.ErrFileNotFound
		}}
		storage := &fakeFileStorage{}
		uc := usecase.NewFileUsecase(repo, storage, 5*1024*1024)
		if err := uc.Delete(context.Background(), usecase.DeleteInput{OwnerUserID: ownerID, FileID: fileID}); !errors.Is(err, domain.ErrFileNotFound) {
			t.Fatalf("err: got %v, want %v", err, domain.ErrFileNotFound)
		}
	})
}

func TestFileUsecase_Upload(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	const maxBytes int64 = 5 * 1024 * 1024
	description := "2026年度の事業計画書"

	tests := []struct {
		name        string
		input       usecase.UploadInput
		repo        func(*fakeFileRepository)
		storage     func(*fakeFileStorage)
		wantErr     error
		wantPath    string
		wantDesc    *string
		wantSize    int64
		wantMime    string
	}{
		{
			name: "upload success",
			input: usecase.UploadInput{
				OwnerUserID: ownerID,
				FileName:    "report.pdf",
				MIMEType:    "application/pdf",
				Description: &description,
				Data:        []byte("test pdf content"),
			},
			repo: func(r *fakeFileRepository) {
				r.createFn = func(_ context.Context, file *domain.File) (*domain.File, error) {
					return file, nil
				}
			},
			storage: func(s *fakeFileStorage) {
				s.saveFn = func(_ context.Context, content domain.FileContent) (*domain.StoredFile, error) {
					return &domain.StoredFile{
						ID:        uuid.New(),
						Name:      content.OriginalName,
						Path:      "/tmp/report.pdf",
						Size:      content.Size,
						MIMEType:  content.MIMEType,
						CreatedAt: time.Now(),
					}, nil
				}
			},
			wantSize: 16,
			wantMime: "application/pdf",
			wantDesc: &description,
		},
		{
			name: "file too large",
			input: usecase.UploadInput{
				OwnerUserID: ownerID,
				FileName:    "large.bin",
				MIMEType:    "application/octet-stream",
				Data:        make([]byte, maxBytes+1),
			},
			wantErr: domain.ErrFileTooLarge,
		},
		{
			name: "invalid empty file name",
			input: usecase.UploadInput{
				OwnerUserID: ownerID,
				MIMEType:    "application/pdf",
				Data:        []byte("hello"),
			},
			wantErr: domain.ErrInvalidFile,
		},
		{
			name: "storage failure",
			input: usecase.UploadInput{
				OwnerUserID: ownerID,
				FileName:    "report.pdf",
				MIMEType:    "application/pdf",
				Data:        []byte("hello"),
			},
			storage: func(s *fakeFileStorage) {
				s.saveFn = func(context.Context, domain.FileContent) (*domain.StoredFile, error) {
					return nil, errStorageFailure
				}
			},
			wantErr: errStorageFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &fakeFileRepository{}
			storage := &fakeFileStorage{}
			if tt.repo != nil { tt.repo(repo) }
			if tt.storage != nil { tt.storage(storage) }

			uc := usecase.NewFileUsecase(repo, storage, maxBytes)
			out, err := uc.Upload(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil || !errors.Is(err, tt.wantErr) {
					t.Fatalf("err: got %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if out == nil || out.File == nil {
				t.Fatal("output file is nil")
			}
			if out.File.OwnerUserID != ownerID {
				t.Errorf("ownerUserID: got %v, want %v", out.File.OwnerUserID, ownerID)
			}
			if out.File.Size != tt.wantSize {
				t.Errorf("size: got %d, want %d", out.File.Size, tt.wantSize)
			}
			if out.File.MIMEType != tt.wantMime {
				t.Errorf("mimeType: got %s, want %s", out.File.MIMEType, tt.wantMime)
			}
			if out.DownloadURL == "" {
				t.Error("downloadURL: got empty, want non-empty")
			}
			if tt.wantDesc != nil && (out.File.Description == nil || *out.File.Description != *tt.wantDesc) {
				t.Errorf("description: got %v, want %v", out.File.Description, tt.wantDesc)
			}
		})
	}
}

func TestFileUsecase_AdditionalInvalidBranches(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	fileID := uuid.New()
	const maxBytes int64 = 5 * 1024 * 1024

	t.Run("upload invalid owner and empty content", func(t *testing.T) {
		repo := &fakeFileRepository{}
		storage := &fakeFileStorage{}
		uc := usecase.NewFileUsecase(repo, storage, maxBytes)

		if _, err := uc.Upload(context.Background(), usecase.UploadInput{OwnerUserID: uuid.Nil, FileName: "a.pdf", Data: []byte("x")}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("nil owner: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if _, err := uc.Upload(context.Background(), usecase.UploadInput{OwnerUserID: ownerID, FileName: "", Data: []byte("x")}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("empty name: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if _, err := uc.Upload(context.Background(), usecase.UploadInput{OwnerUserID: ownerID, FileName: "a.pdf", Data: nil}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("empty data: got %v, want %v", err, domain.ErrInvalidFile)
		}
	})

	t.Run("upload cleanup on repo create failure", func(t *testing.T) {
		repoErr := errors.New("db create failed")
		stored := &domain.StoredFile{ID: uuid.New(), Name: "a.pdf", Path: "/tmp/a.pdf", Size: 3, MIMEType: "application/pdf", CreatedAt: time.Now()}
		repo := &fakeFileRepository{createFn: func(context.Context, *domain.File) (*domain.File, error) { return nil, repoErr }}
		storage := &fakeFileStorage{
			saveFn: func(context.Context, domain.FileContent) (*domain.StoredFile, error) { return stored, nil },
			deleteFn: func(_ context.Context, got *domain.StoredFile) error {
				if got == nil || got.ID != stored.ID { t.Fatalf("cleanup stored file: got %+v, want id %v", got, stored.ID) }
				return nil
			},
		}
		uc := usecase.NewFileUsecase(repo, storage, maxBytes)
		_, err := uc.Upload(context.Background(), usecase.UploadInput{OwnerUserID: ownerID, FileName: "a.pdf", MIMEType: "application/pdf", Data: []byte("abc")})
		if !errors.Is(err, repoErr) { t.Fatalf("err: got %v, want %v", err, repoErr) }
	})

	t.Run("list get download update delete invalid branches", func(t *testing.T) {
		repo := &fakeFileRepository{}
		storage := &fakeFileStorage{}
		uc := usecase.NewFileUsecase(repo, storage, maxBytes)

		if _, err := uc.List(context.Background(), usecase.ListInput{OwnerUserID: uuid.Nil, Page: 1, Limit: 10}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("list nil owner: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if _, err := uc.List(context.Background(), usecase.ListInput{OwnerUserID: ownerID, Page: 0, Limit: 10}); !errors.Is(err, domain.ErrInvalidPagination) {
			t.Fatalf("list pagination: got %v, want %v", err, domain.ErrInvalidPagination)
		}
		if _, err := uc.Get(context.Background(), usecase.GetInput{OwnerUserID: uuid.Nil, FileID: fileID}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("get nil owner: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if _, err := uc.Get(context.Background(), usecase.GetInput{OwnerUserID: ownerID, FileID: uuid.Nil}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("get nil file: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if _, err := uc.Download(context.Background(), usecase.DownloadInput{OwnerUserID: uuid.Nil, FileID: fileID}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("download nil owner: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if _, err := uc.Download(context.Background(), usecase.DownloadInput{OwnerUserID: ownerID, FileID: uuid.Nil}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("download nil file: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if _, err := uc.UpdateMetadata(context.Background(), usecase.UpdateMetadataInput{OwnerUserID: uuid.Nil, FileID: fileID}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("update nil owner: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if _, err := uc.UpdateMetadata(context.Background(), usecase.UpdateMetadataInput{OwnerUserID: ownerID, FileID: uuid.Nil}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("update nil file: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if _, err := uc.UpdateMetadata(context.Background(), usecase.UpdateMetadataInput{OwnerUserID: ownerID, FileID: fileID}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("update all nil: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if err := uc.Delete(context.Background(), usecase.DeleteInput{OwnerUserID: uuid.Nil, FileID: fileID}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("delete nil owner: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if err := uc.Delete(context.Background(), usecase.DeleteInput{OwnerUserID: ownerID, FileID: uuid.Nil}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("delete nil file: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if err := uc.DeleteFiles(context.Background(), usecase.DeleteFilesInput{OwnerUserID: uuid.Nil, FileIDs: []uuid.UUID{fileID}}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("delete files nil owner: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if err := uc.DeleteFiles(context.Background(), usecase.DeleteFilesInput{OwnerUserID: ownerID, FileIDs: nil}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("delete files empty: got %v, want %v", err, domain.ErrInvalidFile)
		}
		if err := uc.DeleteFiles(context.Background(), usecase.DeleteFilesInput{OwnerUserID: ownerID, FileIDs: []uuid.UUID{uuid.Nil}}); !errors.Is(err, domain.ErrInvalidFile) {
			t.Fatalf("delete files nil id: got %v, want %v", err, domain.ErrInvalidFile)
		}
	})
}
