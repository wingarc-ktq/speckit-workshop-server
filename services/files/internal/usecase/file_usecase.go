// Package usecase はファイル管理に関するアプリケーションロジックを実装する.
package usecase

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
)

// MaxFileSize は 1 ファイルあたりの最大サイズ（FR-003）.
const MaxFileSize = 10 * 1024 * 1024 // 10MB

// FileUsecase はファイル管理に関するアプリケーションの入力ポート（ユースケース契約）.
// handler などの外側アダプタはこの interface に依存する（依存は内向き）.
//
//go:generate mockgen -source=file_usecase.go -destination=mock/file_usecase_mock.go -package=mock
type FileUsecase interface {
	UploadFile(ctx context.Context, input UploadFileInput) (*domain.File, error)
	// ListFiles は検索・ページネーション条件に一致するファイル一覧と総件数を返す.
	// Page/Limit のデフォルト適用は呼び出し側（handler）の責務とし、ここでは
	// 受け取った条件をそのまま FileRepository.List に委譲する.
	ListFiles(ctx context.Context, params ListFilesParams) (files []*domain.File, total int64, err error)
	// GetFile は指定した ID のファイルメタデータを取得する.
	// 該当するファイルが無い場合は domain.ErrFileNotFound を返す
	// （出典: spec.md FR-010「存在しないファイル ID に対して存在しないエラーを返さなければならない」）。
	GetFile(ctx context.Context, id uuid.UUID) (*domain.File, error)
	// DownloadFile はファイル本体（バイナリ）をストレージから取得する.
	// メタデータとファイル本体の両方を返すのは、handler が Content-Disposition
	// ヘッダーに必要なファイル名や MIME タイプをメタデータから組み立てるため.
	DownloadFile(ctx context.Context, id uuid.UUID) (*DownloadOutput, error)
}

// fileUsecase は FileUsecase の実装.
type fileUsecase struct {
	repo    FileRepository
	storage FileStorage
}

// NewFileUsecase は FileUsecase を生成する.
func NewFileUsecase(repo FileRepository, storage FileStorage) FileUsecase {
	return &fileUsecase{repo: repo, storage: storage}
}

// UploadFileInput はファイルアップロードの入力.
// Content は転送方式 (multipart など) に依存しない汎用リーダーとして受け取る.
type UploadFileInput struct {
	Name        string
	MimeType    string
	Description string
	TagIDs      []uuid.UUID
	Content     io.Reader
}

// UploadFile はファイル本体をストレージへ保存し、メタデータを永続化する.
func (u *fileUsecase) UploadFile(ctx context.Context, input UploadFileInput) (*domain.File, error) {
	if input.Content == nil {
		return nil, domain.ErrFileEmpty
	}

	// MaxFileSize+1 まで読み、ストレージへ書き切る前にサイズ超過を判定する
	// (research.md Decision 4).
	buf, err := io.ReadAll(io.LimitReader(input.Content, MaxFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("read upload content: %w", err)
	}
	if len(buf) > MaxFileSize {
		return nil, domain.ErrFileTooLarge
	}

	tagIDs := input.TagIDs
	if tagIDs == nil {
		tagIDs = []uuid.UUID{}
	}

	file := &domain.File{
		ID:          uuid.New(),
		Name:        input.Name,
		Size:        int64(len(buf)),
		MimeType:    input.MimeType,
		Description: input.Description,
		TagIDs:      tagIDs,
	}
	file.StorageKey = file.ID.String()

	if err := u.storage.Save(ctx, file.StorageKey, bytes.NewReader(buf)); err != nil {
		return nil, fmt.Errorf("save file content: %w", err)
	}
	if err := u.repo.Create(ctx, file); err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	return file, nil
}

// ListFiles は FileRepository.List に委譲する.
func (u *fileUsecase) ListFiles(ctx context.Context, params ListFilesParams) ([]*domain.File, int64, error) {
	return u.repo.List(ctx, params)
}

// GetFile は FileRepository.FindByID に委譲する.
// エラーの変換（pgx.ErrNoRows → domain.ErrFileNotFound）は infra 層（file_repository.go）の
// 責務とし、usecase はドメインエラーをそのまま呼び出し元へ伝播させるだけでよい.
func (u *fileUsecase) GetFile(ctx context.Context, id uuid.UUID) (*domain.File, error) {
	return u.repo.FindByID(ctx, id)
}

// DownloadOutput はファイルダウンロードの結果.
type DownloadOutput struct {
	// File はダウンロード対象のメタデータ（ファイル名・MIME タイプなど）.
	File *domain.File
	// Content はストレージから開いた読み取り用ストリーム.
	// ここでは Close せずに呼び出し元（handler）へそのまま渡す。
	// レスポンスの書き込みが終わるまで開いたままにする必要があり、
	// usecase の中で早々に閉じてしまうと handler 側で読めなくなるため.
	Content io.ReadCloser
}

// DownloadFile はファイル本体をストレージから取得する.
// 先にメタデータ（FindByID）を確認してから storage.Open するのは、
// 万が一 DB とストレージの状態がズレていても「存在しないファイル ID」を
// 一貫して 404 として扱えるようにするため（spec.md FR-010）.
func (u *fileUsecase) DownloadFile(ctx context.Context, id uuid.UUID) (*DownloadOutput, error) {
	file, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	content, err := u.storage.Open(ctx, file.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("open file content: %w", err)
	}

	return &DownloadOutput{File: file, Content: content}, nil
}
