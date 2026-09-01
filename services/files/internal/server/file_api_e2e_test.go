//go:build integration

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/wingarc-ktq/speckit-workshop-server/packages/authjwt"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
)

func TestFileAPI_EndToEndJWTFlow(t *testing.T) {
	pool, cleanup := setupFilesServerTestDB(t)
	defer cleanup()

	privPEM, pubPEM := genTestPEM(t)
	verifier, err := authjwt.NewVerifier(pubPEM)
	if err != nil {
		t.Fatal(err)
	}

	e, err := newEcho(pool, verifier, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	userID := uuid.New()
	token := signRS256(t, privPEM, jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	// 1) upload file via real HTTP path
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("hello from e2e")); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("description", "integration e2e"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("upload status: got %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var uploaded gen.FileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &uploaded); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if uploaded.File.Id == (gen.File{}.Id) {
		t.Fatal("uploaded file id is empty")
	}
	fileID := uuid.UUID(uploaded.File.Id)

	// 2) list files includes uploaded file
	req = httptest.NewRequest(http.MethodGet, "/api/v1/files?page=1&limit=20", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status: got %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var listed gen.FileListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listed.Total != 1 {
		t.Fatalf("list total: got %d, want 1", listed.Total)
	}
	if len(listed.Files) != 1 || listed.Files[0].Id != uploaded.File.Id {
		t.Fatalf("list entries: got %+v, want file id %s", listed.Files, uploaded.File.Id)
	}

	// 3) get detail works
	req = httptest.NewRequest(http.MethodGet, "/api/v1/files/"+fileID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status: got %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got gen.FileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if got.File.Id != uploaded.File.Id {
		t.Fatalf("detail id: got %s, want %s", got.File.Id, uploaded.File.Id)
	}

	// 4) download contents works
	req = httptest.NewRequest(http.MethodGet, "/api/v1/files/"+fileID.String()+"/download", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("download status: got %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Body.String() != "hello from e2e" {
		t.Fatalf("download body: got %q, want %q", rec.Body.String(), "hello from e2e")
	}

	// 5) delete file
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/files/"+fileID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status: got %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	// 6) deleted file is no longer readable
	req = httptest.NewRequest(http.MethodGet, "/api/v1/files/"+fileID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("after delete status: got %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func setupFilesServerTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("files"),
		postgres.WithUsername("workshop"),
		postgres.WithPassword("workshop"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}

	for _, migration := range []string{
		"../../migrations/000001_create_files_table.up.sql",
		"../../migrations/000002_create_tags_table.up.sql",
	} {
		ddl, err := os.ReadFile(migration)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(ddl)); err != nil {
			t.Fatal(err)
		}
	}

	cleanup := func() {
		pool.Close()
		_ = container.Terminate(ctx)
	}
	return pool, cleanup
}
