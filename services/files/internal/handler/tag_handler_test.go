package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/handler"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

type fakeTagUsecase struct {
	createFn func(context.Context, usecase.CreateTagInput) (*domain.Tag, error)
	listFn   func(context.Context) ([]domain.Tag, error)
	updateFn func(context.Context, usecase.UpdateTagInput) (*domain.Tag, error)
	deleteFn func(context.Context, uuid.UUID) error
}

func (f *fakeTagUsecase) Create(ctx context.Context, in usecase.CreateTagInput) (*domain.Tag, error) {
	if f.createFn != nil {
		return f.createFn(ctx, in)
	}
	return nil, nil
}

func (f *fakeTagUsecase) List(ctx context.Context) ([]domain.Tag, error) {
	if f.listFn != nil {
		return f.listFn(ctx)
	}
	return nil, nil
}

func (f *fakeTagUsecase) Update(ctx context.Context, in usecase.UpdateTagInput) (*domain.Tag, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, in)
	}
	return nil, nil
}

func (f *fakeTagUsecase) Delete(ctx context.Context, id uuid.UUID) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func TestTagHandler_CreateListUpdateDelete(t *testing.T) {
	t.Parallel()

	tagID := uuid.New()
	t.Run("create", func(t *testing.T) {
		u := &fakeTagUsecase{createFn: func(_ context.Context, in usecase.CreateTagInput) (*domain.Tag, error) {
			if in.Name != "重要" {
				t.Fatalf("name: got %s, want %s", in.Name, "重要")
			}
			if in.Color != string(domain.TagColorRed) {
				t.Fatalf("color: got %s, want %s", in.Color, string(domain.TagColorRed))
			}
			return &domain.Tag{ID: tagID, Name: in.Name, Color: domain.TagColorRed, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		}}
		h := handler.NewTagHandler(u)
		body := `{"name":"重要","color":"red"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tags", bytes.NewReader([]byte(body)))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c := e.NewContext(req, httptest.NewRecorder())
		c.Set("userID", uuid.New())
		if err := h.CreateTag(c); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("list", func(t *testing.T) {
		u := &fakeTagUsecase{listFn: func(context.Context) ([]domain.Tag, error) {
			return []domain.Tag{{ID: tagID, Name: "重要", Color: domain.TagColorRed, CreatedAt: time.Now(), UpdatedAt: time.Now()}}, nil
		}}
		h := handler.NewTagHandler(u)
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
		c := e.NewContext(req, httptest.NewRecorder())
		c.Set("userID", uuid.New())
		if err := h.ListTags(c); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("update", func(t *testing.T) {
		u := &fakeTagUsecase{updateFn: func(_ context.Context, in usecase.UpdateTagInput) (*domain.Tag, error) {
			if in.TagID != tagID {
				t.Fatalf("id: got %v, want %v", in.TagID, tagID)
			}
			if in.Name == nil || *in.Name != "緊急" {
				t.Fatalf("name: got %v, want %s", in.Name, "緊急")
			}
			return &domain.Tag{ID: tagID, Name: *in.Name, Color: domain.TagColorOrange, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		}}
		h := handler.NewTagHandler(u)
		body := `{"name":"緊急","color":"orange"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/tags/"+tagID.String(), bytes.NewReader([]byte(body)))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c := e.NewContext(req, httptest.NewRecorder())
		c.Set("userID", uuid.New())
		if err := h.UpdateTag(c, openapi_types.UUID(tagID)); err != nil {
			t.Fatal(err)
		}
		rec := c.Response().Writer.(*httptest.ResponseRecorder)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("delete", func(t *testing.T) {
		u := &fakeTagUsecase{deleteFn: func(_ context.Context, gotID uuid.UUID) error {
			if gotID != tagID {
				t.Fatalf("id: got %v, want %v", gotID, tagID)
			}
			return nil
		}}
		h := handler.NewTagHandler(u)
		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/tags/"+tagID.String(), nil)
		c := e.NewContext(req, httptest.NewRecorder())
		c.Set("userID", uuid.New())
		if err := h.DeleteTag(c, openapi_types.UUID(tagID)); err != nil {
			t.Fatal(err)
		}
		rec := c.Response().Writer.(*httptest.ResponseRecorder)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent)
		}
	})
}

func TestTagHandler_ErrorMapping(t *testing.T) {
	t.Parallel()
	u := &fakeTagUsecase{createFn: func(context.Context, usecase.CreateTagInput) (*domain.Tag, error) {
		return nil, domain.ErrDuplicateTagName
	}}
	h := handler.NewTagHandler(u)
	e := echo.New()
	body := `{"name":"重要","color":"red"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags", bytes.NewReader([]byte(body)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c := e.NewContext(req, httptest.NewRecorder())
	c.Set("userID", uuid.New())
	if err := h.CreateTag(c); err != nil {
		t.Fatal(err)
	}
	rec := c.Response().Writer.(*httptest.ResponseRecorder)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusConflict)
	}
	var out gen.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Code != "TAG_ALREADY_EXISTS" {
		t.Fatalf("code: got %s, want %s", out.Code, "TAG_ALREADY_EXISTS")
	}
}
