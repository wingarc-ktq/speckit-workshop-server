package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/handler"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/storage"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

func TestHealthRoutes(t *testing.T) {
	store, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	uc := usecase.NewFileUsecase(repo.NewInMemoryFileRepository(), store, 10*1024*1024)
	h := handler.NewFilesHandler(uc)
	e, err := newEcho(h, nil, nil)
	if err != nil {
		t.Fatalf("newEcho() error = %v", err)
	}

	for _, tc := range []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "healthz", path: "/healthz", wantStatus: http.StatusOK},
		{name: "readyz", path: "/readyz", wantStatus: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			if res.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", res.Code, tc.wantStatus)
			}
		})
	}
}

func TestRunContextShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Run(ctx); err != nil {
		t.Fatalf("Run() unexpected error on canceled context: %v", err)
	}
}
