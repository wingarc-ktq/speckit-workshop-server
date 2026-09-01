// Package config は Files サービスの環境変数ベースの設定読み込みを提供する.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config は Files サービスの設定.
type Config struct {
	Port        string
	DatabaseURL string
	StoragePath string
	JWTPublicKey []byte
	JWTTTL      time.Duration
}

// Load は環境変数から Config を読み込む.
func Load() (*Config, error) {
	port := getenv("FILES_SERVICE_PORT", "8082")

	databaseURL := os.Getenv("FILES_DATABASE_URL")
	if databaseURL == "" {
		return nil, errors.New("FILES_DATABASE_URL is required")
	}

	storagePath := os.Getenv("FILES_STORAGE_PATH")
	if storagePath == "" {
		return nil, errors.New("FILES_STORAGE_PATH is required")
	}

	pubKey, err := readKey("FILES_JWT_PUBLIC_KEY_PATH")
	if err != nil {
		return nil, err
	}

	ttlStr := getenv("FILES_JWT_TTL_SECONDS", "3600")
	ttlSec, err := strconv.Atoi(ttlStr)
	if err != nil {
		return nil, errors.New("FILES_JWT_TTL_SECONDS must be integer")
	}

	return &Config{
		Port:         port,
		DatabaseURL:  databaseURL,
		StoragePath:  storagePath,
		JWTPublicKey: pubKey,
		JWTTTL:       time.Duration(ttlSec) * time.Second,
	}, nil
}

// readKey は環境変数で指定されたパスから PEM 鍵ファイルを読み込む.
func readKey(envKey string) ([]byte, error) {
	path := os.Getenv(envKey)
	if path == "" {
		return nil, fmt.Errorf("%s is required", envKey)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s (%s): %w", envKey, path, err)
	}
	return data, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
