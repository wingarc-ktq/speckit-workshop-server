package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/wingarc-ktq/speckit-workshop-server/packages/authjwt"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

// TagHandler はタグ API のハンドラ実装.
type TagHandler struct {
	uc usecase.TagUsecase
}

// NewTagHandler はタグハンドラを生成する.
func NewTagHandler(uc usecase.TagUsecase) *TagHandler {
	return &TagHandler{uc: uc}
}

func (h *TagHandler) ListTags(ctx echo.Context) error {
	if _, ok := authjwt.UserIDFromContext(ctx); !ok {
		return h.errorResponse(ctx, domain.ErrInvalidTag)
	}
	items, err := h.uc.List(ctx.Request().Context())
	if err != nil {
		return h.errorResponse(ctx, err)
	}
	resp := gen.TagListResponse{Tags: make([]gen.Tag, 0, len(items))}
	for _, item := range items {
		resp.Tags = append(resp.Tags, toGenTag(item))
	}
	return ctx.JSON(http.StatusOK, resp)
}

func (h *TagHandler) CreateTag(ctx echo.Context) error {
	if _, ok := authjwt.UserIDFromContext(ctx); !ok {
		return h.errorResponse(ctx, domain.ErrInvalidTag)
	}
	var req gen.CreateTagRequest
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
		return validationError(ctx, "JSON が不正です")
	}
	created, err := h.uc.Create(ctx.Request().Context(), usecase.CreateTagInput{
		Name:  strings.TrimSpace(req.Name),
		Color: string(req.Color),
	})
	if err != nil {
		return h.errorResponse(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, gen.TagResponse{Tag: toGenTag(*created)})
}

func (h *TagHandler) DeleteTag(ctx echo.Context, tagId openapi_types.UUID) error {
	if _, ok := authjwt.UserIDFromContext(ctx); !ok {
		return h.errorResponse(ctx, domain.ErrInvalidTag)
	}
	if err := h.uc.Delete(ctx.Request().Context(), uuid.UUID(tagId)); err != nil {
		return h.errorResponse(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (h *TagHandler) UpdateTag(ctx echo.Context, tagId openapi_types.UUID) error {
	if _, ok := authjwt.UserIDFromContext(ctx); !ok {
		return h.errorResponse(ctx, domain.ErrInvalidTag)
	}
	var req gen.UpdateTagRequest
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
		return validationError(ctx, "JSON が不正です")
	}
	updated, err := h.uc.Update(ctx.Request().Context(), usecase.UpdateTagInput{
		TagID: uuid.UUID(tagId),
		Name:  req.Name,
		Color: updateTagColorPtr(req.Color),
	})
	if err != nil {
		return h.errorResponse(ctx, err)
	}
	return ctx.JSON(http.StatusOK, gen.TagResponse{Tag: toGenTag(*updated)})
}

func updateTagColorPtr(color *gen.UpdateTagRequestColor) *string {
	if color == nil {
		return nil
	}
	v := string(*color)
	return &v
}

func (h *TagHandler) errorResponse(ctx echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidTag):
		return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{Code: "VALIDATION_ERROR", Message: "タグの入力が不正です"})
	case errors.Is(err, domain.ErrDuplicateTagName):
		return ctx.JSON(http.StatusConflict, gen.ErrorResponse{Code: "TAG_ALREADY_EXISTS", Message: "同名のタグが既に存在します"})
	case errors.Is(err, domain.ErrTagNotFound):
		return ctx.JSON(http.StatusNotFound, gen.ErrorResponse{Code: "TAG_NOT_FOUND", Message: "タグが見つかりません"})
	default:
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{Code: "INTERNAL_ERROR", Message: "内部エラーが発生しました"})
	}
}

func toGenTag(tag domain.Tag) gen.Tag {
	return gen.Tag{
		Id:        openapi_types.UUID(tag.ID),
		Name:      tag.Name,
		Color:     gen.TagColor(tag.Color),
		CreatedAt: tag.CreatedAt,
		UpdatedAt: tag.UpdatedAt,
	}
}
