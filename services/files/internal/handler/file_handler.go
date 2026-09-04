package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/wingarc-ktq/speckit-workshop-server/packages/authjwt"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

type FileHandler struct {
	uc    usecase.FileUsecase
	tagUC usecase.TagUsecase
}

func NewFileHandler(uc usecase.FileUsecase, tagUC ...usecase.TagUsecase) *FileHandler {
	var selected usecase.TagUsecase
	if len(tagUC) > 0 {
		selected = tagUC[0]
	}
	return &FileHandler{uc: uc, tagUC: selected}
}

func (h *FileHandler) SetTagUsecase(uc usecase.TagUsecase) {
	h.tagUC = uc
}

func (h *FileHandler) ListFiles(ctx echo.Context, params gen.ListFilesParams) error {
	userID, ok := authjwt.UserIDFromContext(ctx)
	if !ok {
		return h.errorResponse(ctx, domain.ErrInvalidFile)
	}

	page := 1
	if params.Page != nil {
		page = *params.Page
	}
	limit := 20
	if params.Limit != nil {
		limit = *params.Limit
	}
	keyword := ""
	if params.Q != nil {
		keyword = *params.Q
	}

	out, err := h.uc.List(ctx.Request().Context(), usecase.ListInput{
		OwnerUserID: userID,
		Keyword:     keyword,
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		return h.errorResponse(ctx, err)
	}

	resp := gen.FileListResponse{
		Files: make([]gen.File, 0, len(out.Files)),
		Total: out.Total,
		Page:  out.Page,
		Limit: out.Limit,
	}
	for _, f := range out.Files {
		resp.Files = append(resp.Files, toGenFile(&f, fmt.Sprintf("/api/v1/files/%s/download", f.ID.String())))
	}
	return ctx.JSON(http.StatusOK, resp)
}

func (h *FileHandler) UploadFile(ctx echo.Context) error {
	userID, ok := authjwt.UserIDFromContext(ctx)
	if !ok {
		return h.errorResponse(ctx, domain.ErrInvalidFile)
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		return validationError(ctx, "multipart/form-data が不正です")
	}

	files := form.File["file"]
	if len(files) == 0 {
		return validationError(ctx, "file は必須です")
	}
	fileHeader := files[0]
	if fileHeader == nil {
		return validationError(ctx, "file は必須です")
	}

	fh, err := fileHeader.Open()
	if err != nil {
		return validationError(ctx, "file を読み込めませんでした")
	}
	defer fh.Close()

	content, err := io.ReadAll(fh)
	if err != nil {
		return validationError(ctx, "file を読み込めませんでした")
	}

	description := ptrFromForm(form.Value["description"])
	if description != nil && utf8.RuneCountInString(*description) > 500 {
		return validationError(ctx, "description は500文字以内で指定してください")
	}
	out, err := h.uc.Upload(ctx.Request().Context(), usecase.UploadInput{
		OwnerUserID: userID,
		FileName:    fileHeader.Filename,
		MIMEType:    detectMIMEType(fileHeader.Filename, content),
		Description: description,
		Data:        content,
	})
	if err != nil {
		return h.errorResponse(ctx, err)
	}

	return ctx.JSON(http.StatusCreated, gen.FileResponse{File: toGenFile(out.File, out.DownloadURL)})
}

func (h *FileHandler) BatchDeleteFiles(ctx echo.Context) error {
	userID, ok := authjwt.UserIDFromContext(ctx)
	if !ok {
		return h.errorResponse(ctx, domain.ErrInvalidFile)
	}

	var req gen.BatchDeleteRequest
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
		return validationError(ctx, "JSON が不正です")
	}
	if len(req.FileIds) == 0 {
		return validationError(ctx, "削除するファイルIDを少なくとも1つ指定してください")
	}

	ids := make([]uuid.UUID, 0, len(req.FileIds))
	for _, id := range req.FileIds {
		ids = append(ids, uuidFromOpenAPI(id))
	}
	if err := h.uc.DeleteFiles(ctx.Request().Context(), usecase.DeleteFilesInput{OwnerUserID: userID, FileIDs: ids}); err != nil {
		return h.errorResponse(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (h *FileHandler) DeleteFile(ctx echo.Context, fileId openapi_types.UUID) error {
	userID, ok := authjwt.UserIDFromContext(ctx)
	if !ok {
		return h.errorResponse(ctx, domain.ErrInvalidFile)
	}
	if err := h.uc.Delete(ctx.Request().Context(), usecase.DeleteInput{OwnerUserID: userID, FileID: uuidFromOpenAPI(fileId)}); err != nil {
		return h.errorResponse(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (h *FileHandler) GetFile(ctx echo.Context, fileId openapi_types.UUID) error {
	userID, ok := authjwt.UserIDFromContext(ctx)
	if !ok {
		return h.errorResponse(ctx, domain.ErrInvalidFile)
	}

	file, err := h.uc.Get(ctx.Request().Context(), usecase.GetInput{OwnerUserID: userID, FileID: uuidFromOpenAPI(fileId)})
	if err != nil {
		return h.errorResponse(ctx, err)
	}
	return ctx.JSON(http.StatusOK, gen.FileResponse{File: toGenFile(file, fmt.Sprintf("/api/v1/files/%s/download", file.ID.String()))})
}

func (h *FileHandler) UpdateFile(ctx echo.Context, fileId openapi_types.UUID) error {
	userID, ok := authjwt.UserIDFromContext(ctx)
	if !ok {
		return h.errorResponse(ctx, domain.ErrInvalidFile)
	}

	var req gen.UpdateFileRequest
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
		return validationError(ctx, "JSON が不正です")
	}

	input := usecase.UpdateMetadataInput{
		OwnerUserID: userID,
		FileID:      uuidFromOpenAPI(fileId),
		Name:        req.Name,
		Description: req.Description,
	}
	if req.TagIds != nil {
		input.TagIDs = make([]uuid.UUID, 0, len(*req.TagIds))
		for _, id := range *req.TagIds {
			input.TagIDs = append(input.TagIDs, uuidFromOpenAPI(id))
		}
	}

	updated, err := h.uc.UpdateMetadata(ctx.Request().Context(), input)
	if err != nil {
		return h.errorResponse(ctx, err)
	}
	return ctx.JSON(http.StatusOK, gen.FileResponse{File: toGenFile(updated, fmt.Sprintf("/api/v1/files/%s/download", updated.ID.String()))})
}

func (h *FileHandler) DownloadFile(ctx echo.Context, fileId openapi_types.UUID) error {
	userID, ok := authjwt.UserIDFromContext(ctx)
	if !ok {
		return h.errorResponse(ctx, domain.ErrInvalidFile)
	}

	out, err := h.uc.Download(ctx.Request().Context(), usecase.DownloadInput{OwnerUserID: userID, FileID: uuidFromOpenAPI(fileId)})
	if err != nil {
		return h.errorResponse(ctx, err)
	}
	if out == nil || out.File == nil {
		return h.errorResponse(ctx, domain.ErrFileNotFound)
	}
	ctx.Response().Header().Set("Content-Type", "application/octet-stream")
	ctx.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", out.File.Name))
	ctx.Response().Header().Set("Content-Length", fmt.Sprintf("%d", len(out.Data)))
	return ctx.Blob(http.StatusOK, "application/octet-stream", out.Data)
}

func (h *FileHandler) ListTags(ctx echo.Context) error {
	if h.tagUC == nil {
		return notImplemented(ctx, "listTags")
	}
	return NewTagHandler(h.tagUC).ListTags(ctx)
}

func (h *FileHandler) CreateTag(ctx echo.Context) error {
	if h.tagUC == nil {
		return notImplemented(ctx, "createTag")
	}
	return NewTagHandler(h.tagUC).CreateTag(ctx)
}

func (h *FileHandler) DeleteTag(ctx echo.Context, tagId openapi_types.UUID) error {
	if h.tagUC == nil {
		return notImplemented(ctx, "deleteTag")
	}
	return NewTagHandler(h.tagUC).DeleteTag(ctx, tagId)
}

func (h *FileHandler) UpdateTag(ctx echo.Context, tagId openapi_types.UUID) error {
	if h.tagUC == nil {
		return notImplemented(ctx, "updateTag")
	}
	return NewTagHandler(h.tagUC).UpdateTag(ctx, tagId)
}

func notImplemented(ctx echo.Context, endpoint string) error {
	return ctx.JSON(http.StatusNotImplemented, gen.ErrorResponse{
		Code:    "NOT_IMPLEMENTED",
		Message: endpoint + " is not implemented yet",
	})
}

func validationError(ctx echo.Context, msg string) error {
	return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{
		Code:    "VALIDATION_ERROR",
		Message: msg,
	})
}

func (h *FileHandler) errorResponse(ctx echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidFile), errors.Is(err, domain.ErrInvalidPagination):
		return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "リクエストが不正です",
		})
	case errors.Is(err, domain.ErrFileTooLarge):
		return ctx.JSON(http.StatusRequestEntityTooLarge, gen.ErrorResponse{
			Code:    "FILE_TOO_LARGE",
			Message: "ファイルサイズが上限を超えています",
		})
	case errors.Is(err, domain.ErrFileNotFound):
		return ctx.JSON(http.StatusNotFound, gen.ErrorResponse{
			Code:    "FILE_NOT_FOUND",
			Message: "ファイルが見つかりません",
		})
	default:
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "内部エラーが発生しました",
		})
	}
}

func uuidFromOpenAPI(id openapi_types.UUID) uuid.UUID {
	return uuid.UUID(id)
}

func ptrFromForm(values []string) *string {
	if len(values) == 0 {
		return nil
	}
	v := strings.TrimSpace(values[0])
	if v == "" {
		return nil
	}
	return &v
}

func detectMIMEType(fileName string, data []byte) string {
	if mimeType := http.DetectContentType(data); mimeType != "application/octet-stream" {
		return mimeType
	}
	if ext := filepath.Ext(fileName); ext != "" {
		if mt := mime.TypeByExtension(ext); mt != "" {
			return mt
		}
	}
	return "application/octet-stream"
}

func toGenFile(f *domain.File, downloadURL string) gen.File {
	if f == nil {
		return gen.File{}
	}
	tagIDs := make([]openapi_types.UUID, 0, len(f.TagIDs))
	for _, tagID := range f.TagIDs {
		tagIDs = append(tagIDs, openapi_types.UUID(tagID))
	}
	file := gen.File{
		Id:          openapi_types.UUID(f.ID),
		Name:        f.Name,
		Size:        f.Size,
		MimeType:    f.MIMEType,
		DownloadUrl: downloadURL,
		TagIds:      tagIDs,
		UploadedAt:  f.UploadedAt,
	}
	if f.Description != nil {
		v := *f.Description
		file.Description = &v
	}
	return file
}

var _ gen.ServerInterface = (*FileHandler)(nil)
