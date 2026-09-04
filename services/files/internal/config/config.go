package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Config は Files サービスの設定値を保持する.
type Config struct {
	Port           string
	DatabaseURL    string
	StorageRoot    string
	MaxUploadBytes int64
	JWTPublicKey   []byte
}

func Load() (*Config, error) {
	port := getenv("FILES_SERVICE_PORT", "8082")
	dbURL := os.Getenv("FILES_DATABASE_URL")
	storageRoot := getenv("FILES_STORAGE_ROOT", "/tmp/files-storage")
	maxBytesStr := getenv("FILES_MAX_UPLOAD_BYTES", "10485760")
	maxBytes, err := strconv.ParseInt(maxBytesStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("FILES_MAX_UPLOAD_BYTES must be integer: %w", err)
	}
	if maxBytes <= 0 {
		return nil, errors.New("FILES_MAX_UPLOAD_BYTES must be positive")
	}

	var pubKey []byte
	if keyPath := os.Getenv("FILES_JWT_PUBLIC_KEY_PATH"); keyPath != "" {
		pubKey, err = os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read FILES_JWT_PUBLIC_KEY_PATH (%s): %w", keyPath, err)
		}
	}

	return &Config{
		Port:           port,
		DatabaseURL:    dbURL,
		StorageRoot:    storageRoot,
		MaxUploadBytes: maxBytes,
		JWTPublicKey:   pubKey,
	}, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
