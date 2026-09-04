package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

type FilesHandler struct {
	usecase *usecase.FileUsecase
}

func NewFilesHandler(usecase *usecase.FileUsecase) *FilesHandler {
	return &FilesHandler{usecase: usecase}
}

var _ gen.ServerInterface = (*FilesHandler)(nil)

func (h *FilesHandler) GetFiles(c echo.Context, params gen.GetFilesParams) error {
	if h.usecase == nil {
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Code: "INTERNAL_ERROR", Message: "file usecase not configured"})
	}

	page := 1
	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}
	limit := 20
	if params.Limit != nil && *params.Limit > 0 {
		limit = *params.Limit
	}
	name := ""
	if params.Name != nil {
		name = *params.Name
	}

	tagIDs, err := parseTagIDsFromQuery(params.TagIds)
	if err != nil {
		return c.JSON(http.StatusBadRequest, gen.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()})
	}

	files, total, err := h.usecase.ListFiles(c.Request().Context(), usecase.ListFilesInput{
		Page:   page,
		Limit:  limit,
		Name:   name,
		TagIDs: tagIDs,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Code: "INTERNAL_ERROR", Message: err.Error()})
	}
	return c.JSON(http.StatusOK, fileListFromDomain(files, int(total), page, limit))
}

func (h *FilesHandler) UploadFile(c echo.Context) error {
	if h.usecase == nil {
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Code: "INTERNAL_ERROR", Message: "file usecase not configured"})
	}

	if err := c.Request().ParseMultipartForm(10 << 20); err != nil {
		return c.JSON(http.StatusBadRequest, gen.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()})
	}
	form := c.Request().MultipartForm
	files, ok := form.File["file"]
	if !ok || len(files) == 0 {
		return c.JSON(http.StatusBadRequest, gen.ErrorResponse{Code: "VALIDATION_ERROR", Message: "file is required"})
	}
	fh := files[0]
	if fh.Size > domain.MaxFileSize {
		return c.JSON(http.StatusRequestEntityTooLarge, gen.ErrorResponse{Code: "FILE_TOO_LARGE", Message: "file exceeds 10MiB limit"})
	}

	tagIDs, err := parseTagIDs(form.Value["tagIds"])
	if err != nil {
		return c.JSON(http.StatusBadRequest, gen.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()})
	}

	f, err := fh.Open()
	if err != nil {
		return c.JSON(http.StatusBadRequest, gen.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()})
	}
	defer f.Close()

	payload, err := io.ReadAll(io.LimitReader(f, domain.MaxFileSize+1))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Code: "INTERNAL_ERROR", Message: err.Error()})
	}
	if int64(len(payload)) > domain.MaxFileSize {
		return c.JSON(http.StatusRequestEntityTooLarge, gen.ErrorResponse{Code: "FILE_TOO_LARGE", Message: "file exceeds 10MiB limit"})
	}
	mimeType := fh.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(payload)
	}

	file, err := h.usecase.UploadFile(c.Request().Context(), usecase.UploadInput{
		FileName:    fh.Filename,
		MIMEType:    mimeType,
		Description: strings.Join(form.Value["description"], " "),
		Size:        int64(len(payload)),
		TagIDs:      tagIDs,
		Reader:      bytes.NewReader(payload),
	})
	if err != nil {
		if errors.Is(err, domain.ErrFileTooLarge) {
			return c.JSON(http.StatusRequestEntityTooLarge, gen.ErrorResponse{Code: "FILE_TOO_LARGE", Message: err.Error()})
		}
		if errors.Is(err, domain.ErrInvalidFileName) || errors.Is(err, domain.ErrInvalidMIMEType) || errors.Is(err, domain.ErrInvalidDescription) {
			return c.JSON(http.StatusBadRequest, gen.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()})
		}
		if errors.Is(err, domain.ErrTagNotFound) {
			return c.JSON(http.StatusConflict, gen.ErrorResponse{Code: "TAG_NOT_FOUND", Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Code: "INTERNAL_ERROR", Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, fileInfoFromDomain(file))
}

func (h *FilesHandler) GetFileById(c echo.Context, fileId openapi_types.UUID) error {
	if h.usecase == nil {
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Code: "INTERNAL_ERROR", Message: "file usecase not configured"})
	}
	file, err := h.usecase.GetFileByID(c.Request().Context(), uuid.UUID(fileId))
	if err != nil {
		if errors.Is(err, domain.ErrFileNotFound) {
			return c.JSON(http.StatusNotFound, gen.ErrorResponse{Code: "FILE_NOT_FOUND", Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Code: "INTERNAL_ERROR", Message: err.Error()})
	}
	return c.JSON(http.StatusOK, fileInfoFromDomain(file))
}

func (h *FilesHandler) DownloadFileById(c echo.Context, fileId openapi_types.UUID) error {
	if h.usecase == nil {
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Code: "INTERNAL_ERROR", Message: "file usecase not configured"})
	}
	file, reader, err := h.usecase.DownloadFile(c.Request().Context(), uuid.UUID(fileId))
	if err != nil {
		if errors.Is(err, domain.ErrFileNotFound) {
			return c.JSON(http.StatusNotFound, gen.ErrorResponse{Code: "FILE_NOT_FOUND", Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Code: "INTERNAL_ERROR", Message: err.Error()})
	}
	defer reader.Close()
	c.Response().Header().Set("Content-Type", file.MIMEType)
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", file.Size))
	_, err = io.Copy(c.Response(), reader)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, gen.ErrorResponse{Code: "INTERNAL_ERROR", Message: err.Error()})
	}
	return nil
}

func parseTagIDs(values []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(values))
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			id, err := uuid.Parse(trimmed)
			if err != nil {
				return nil, fmt.Errorf("invalid tag id %q: %w", trimmed, err)
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func parseTagIDsFromQuery(values *[]openapi_types.UUID) ([]uuid.UUID, error) {
	if values == nil {
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(*values))
	for _, value := range *values {
		ids = append(ids, uuid.UUID(value))
	}
	return ids, nil
}

func (h *FilesHandler) withContext(ctx context.Context) context.Context { return ctx }
