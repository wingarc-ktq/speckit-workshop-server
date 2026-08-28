// Package config は Files サービスの環境変数ベースの設定読み込みを提供する.
package config

import (
	"errors"
	"fmt"
	"os"
)

// Config は Files サービスの設定.
type Config struct {
	Port        string
	DatabaseURL string
	// JWTPublicKey は RS256 検証用の公開鍵 (PEM). Files は検証のみ行い、秘密鍵は持たない (Constitution VII).
	JWTPublicKey []byte
	// StorageDir はアップロードされたファイル本体を保存するローカルディレクトリ.
	StorageDir string
}

// Load は環境変数から Config を読み込む.
func Load() (*Config, error) {
	port := getenv("FILES_SERVICE_PORT", "8082")

	dbURL := os.Getenv("FILES_DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("FILES_DATABASE_URL is required")
	}

	pubKey, err := readKey("FILES_JWT_PUBLIC_KEY_PATH")
	if err != nil {
		return nil, err
	}

	storageDir := getenv("FILES_STORAGE_DIR", "./data")

	return &Config{
		Port:         port,
		DatabaseURL:  dbURL,
		JWTPublicKey: pubKey,
		StorageDir:   storageDir,
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
