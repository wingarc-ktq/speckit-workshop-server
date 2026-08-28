// Package handler は OpenAPI で生成された gen.StrictServerInterface を実装する
// HTTP ハンドラ層を提供する.
//
// リクエストの形式・制約の検証は cmd/server で組み込む OpenAPI 検証ミドルウェア
// (OapiRequestValidator) が担うため、ハンドラはビジネス的な変換・整形に専念する.
package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

// filesBasePath は OpenAPI のサーバー URL と一致させる API のベースパス.
// downloadUrl の組み立てに使う（DB には保存しない）.
const filesBasePath = "/api/v1"

// FileHandler はファイル管理 API の HTTP ハンドラ.
// gen.StrictServerInterface を実装する.
type FileHandler struct {
	uc usecase.FileUsecase
}

// NewFileHandler は FileHandler を生成する.
func NewFileHandler(uc usecase.FileUsecase) *FileHandler {
	return &FileHandler{uc: uc}
}

// インターフェース実装の静的チェック.
// GetFiles/UploadFile/GetFile/DownloadFileContent がすべて揃った（Phase 3〜5）ことで
// gen.StrictServerInterface を満たすようになったため、ここで宣言できるようになった.
var _ gen.StrictServerInterface = (*FileHandler)(nil)

// UploadFile はファイルアップロード (POST /files).
func (h *FileHandler) UploadFile(ctx context.Context, request gen.UploadFileRequestObject) (gen.UploadFileResponseObject, error) {
	input, err := parseUploadFileRequest(request.Body)
	if err != nil {
		return gen.UploadFile400JSONResponse{
			Code:    "INVALID_PARAMETER",
			Message: "リクエストの解析に失敗しました",
		}, nil
	}

	file, err := h.uc.UploadFile(ctx, input)
	switch {
	case err == nil:
		return gen.UploadFile201JSONResponse{File: toGenFileInfo(file)}, nil
	case errors.Is(err, domain.ErrFileEmpty):
		return gen.UploadFile400JSONResponse{
			Code:    "INVALID_PARAMETER",
			Message: "ファイルが指定されていません",
		}, nil
	case errors.Is(err, domain.ErrFileTooLarge):
		return gen.UploadFile413JSONResponse{
			Code:    "FILE_TOO_LARGE",
			Message: "ファイルサイズが上限を超えています（最大 10MB）",
		}, nil
	default:
		// 想定外のエラー（ストレージ/DB 障害等）はここでは整形せず、
		// server.go 側の共通エラーハンドラに委ねる.
		return nil, err
	}
}

// parseUploadFileRequest は multipart.Reader から usecase.UploadFileInput を組み立てる.
// multipart.Reader は前方読み取り専用（NextPart するとそれまでのパートは破棄される）ため、
// file パートは遭遇した時点で（MaxFileSize+1 まで）読み切る.
func parseUploadFileRequest(mr *multipart.Reader) (usecase.UploadFileInput, error) {
	var input usecase.UploadFileInput

	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return usecase.UploadFileInput{}, err
		}

		switch part.FormName() {
		case "file":
			buf, err := io.ReadAll(io.LimitReader(part, usecase.MaxFileSize+1))
			if err != nil {
				return usecase.UploadFileInput{}, err
			}
			input.Name = part.FileName()
			input.MimeType = part.Header.Get("Content-Type")
			input.Content = bytes.NewReader(buf)
		case "description":
			b, err := io.ReadAll(part)
			if err != nil {
				return usecase.UploadFileInput{}, err
			}
			input.Description = string(b)
		case "tagIds":
			b, err := io.ReadAll(part)
			if err != nil {
				return usecase.UploadFileInput{}, err
			}
			id, err := uuid.Parse(string(b))
			if err != nil {
				return usecase.UploadFileInput{}, err
			}
			input.TagIDs = append(input.TagIDs, id)
		}
	}

	// file パートが無かった場合 input.Content は nil のままとなり、
	// usecase.UploadFile が domain.ErrFileEmpty を返す.
	return input, nil
}

// toGenFileInfo は domain.File を API レスポンス用の gen.FileInfo に変換する.
func toGenFileInfo(f *domain.File) gen.FileInfo {
	var description *string
	if f.Description != "" {
		description = &f.Description
	}

	tagIDs := make([]openapi_types.UUID, len(f.TagIDs))
	for i, id := range f.TagIDs {
		tagIDs[i] = openapi_types.UUID(id)
	}

	return gen.FileInfo{
		Id:          openapi_types.UUID(f.ID),
		Name:        f.Name,
		Size:        f.Size,
		MimeType:    f.MimeType,
		Description: description,
		UploadedAt:  f.UploadedAt,
		DownloadUrl: filesBasePath + "/files/" + f.ID.String() + "/content",
		TagIds:      tagIDs,
	}
}

// GetFiles はファイル一覧取得 (GET /files).
// ページネーション・検索・タグフィルタの条件を usecase.ListFilesParams に変換する.
// page/limit のデフォルト（1/20）はここで適用し、レスポンスにもその値をそのまま使う.
func (h *FileHandler) GetFiles(ctx context.Context, request gen.GetFilesRequestObject) (gen.GetFilesResponseObject, error) {
	page := 1
	if request.Params.Page != nil {
		page = *request.Params.Page
	}
	limit := 20
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}
	search := ""
	if request.Params.Search != nil {
		search = *request.Params.Search
	}
	var tagIDs []uuid.UUID
	if request.Params.TagIds != nil {
		tagIDs = make([]uuid.UUID, len(*request.Params.TagIds))
		for i, id := range *request.Params.TagIds {
			tagIDs[i] = uuid.UUID(id)
		}
	}

	files, total, err := h.uc.ListFiles(ctx, usecase.ListFilesParams{
		Page:   page,
		Limit:  limit,
		Search: search,
		TagIDs: tagIDs,
	})
	if err != nil {
		return nil, err
	}

	infos := make([]gen.FileInfo, len(files))
	for i, f := range files {
		infos[i] = toGenFileInfo(f)
	}

	return gen.GetFiles200JSONResponse{
		Files: infos,
		Total: int(total),
		Page:  page,
		Limit: limit,
	}, nil
}

// GetFile はファイル詳細取得 (GET /files/{fileId}).
func (h *FileHandler) GetFile(ctx context.Context, request gen.GetFileRequestObject) (gen.GetFileResponseObject, error) {
	file, err := h.uc.GetFile(ctx, uuid.UUID(request.FileId))
	switch {
	case err == nil:
		return gen.GetFile200JSONResponse{File: toGenFileInfo(file)}, nil
	case errors.Is(err, domain.ErrFileNotFound):
		// 出典: schema/files/openapi.yaml の GetFile 404 レスポンス例
		// （spec.md FR-010「存在しないファイル ID に対して存在しないエラーを返す」）。
		return gen.GetFile404JSONResponse{
			Code:    "FILE_NOT_FOUND",
			Message: "ファイルが見つかりません",
		}, nil
	default:
		return nil, err
	}
}

// DownloadFileContent はファイル本体のダウンロード (GET /files/{fileId}/content).
func (h *FileHandler) DownloadFileContent(ctx context.Context, request gen.DownloadFileContentRequestObject) (gen.DownloadFileContentResponseObject, error) {
	out, err := h.uc.DownloadFile(ctx, uuid.UUID(request.FileId))
	switch {
	case err == nil:
		// filename にはアップロード時の元のファイル名を使う。
		// ブラウザ側の「名前を付けて保存」で表示される名前がこれになる.
		disposition := fmt.Sprintf(`attachment; filename="%s"`, out.File.Name)

		// out.Content（io.ReadCloser）はここでは閉じない。
		// oapi-codegen の生成コード（VisitDownloadFileContentResponse）が
		// レスポンス書き込み後に自動で Close するため、ハンドラ側で Close すると
		// 二重クローズになったり、書き込み前に閉じてしまう恐れがある.
		return gen.DownloadFileContent200ApplicationoctetStreamResponse{
			Body: out.Content,
			Headers: gen.DownloadFileContent200ResponseHeaders{
				ContentDisposition: &disposition,
			},
			ContentLength: out.File.Size,
		}, nil
	case errors.Is(err, domain.ErrFileNotFound):
		return gen.DownloadFileContent404JSONResponse{
			Code:    "FILE_NOT_FOUND",
			Message: "ファイルが見つかりません",
		}, nil
	default:
		return nil, err
	}
}
