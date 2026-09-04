package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	oldDB := os.Getenv("FILES_DATABASE_URL")
	oldPort := os.Getenv("FILES_SERVICE_PORT")
	oldRoot := os.Getenv("FILES_STORAGE_ROOT")
	oldMax := os.Getenv("FILES_MAX_UPLOAD_BYTES")
	defer func() {
		setenv("FILES_DATABASE_URL", oldDB)
		setenv("FILES_SERVICE_PORT", oldPort)
		setenv("FILES_STORAGE_ROOT", oldRoot)
		setenv("FILES_MAX_UPLOAD_BYTES", oldMax)
	}()

	_ = os.Unsetenv("FILES_DATABASE_URL")
	_ = os.Unsetenv("FILES_SERVICE_PORT")
	_ = os.Unsetenv("FILES_STORAGE_ROOT")
	_ = os.Unsetenv("FILES_MAX_UPLOAD_BYTES")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error when FILES_DATABASE_URL is missing: %v", err)
	}
	if cfg.DatabaseURL != "" {
		t.Fatalf("DatabaseURL = %q, want empty", cfg.DatabaseURL)
	}

	_ = os.Setenv("FILES_DATABASE_URL", "postgres://localhost:5432/files")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.Port != "8082" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "8082")
	}
	if cfg.StorageRoot != "/tmp/files-storage" {
		t.Fatalf("StorageRoot = %q, want %q", cfg.StorageRoot, "/tmp/files-storage")
	}
	if cfg.MaxUploadBytes != 10485760 {
		t.Fatalf("MaxUploadBytes = %d, want %d", cfg.MaxUploadBytes, 10485760)
	}
}

func setenv(key, value string) {
	if value == "" {
		_ = os.Unsetenv(key)
		return
	}
	_ = os.Setenv(key, value)
}
