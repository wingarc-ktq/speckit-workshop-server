// Package config は Auth サービスの環境変数ベースの設定読み込みを提供する.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config は Auth サービスの設定.
type Config struct {
	Port        string
	DatabaseURL string
	// JWTPrivateKey は RS256 署名用の秘密鍵 (PEM).
	JWTPrivateKey []byte
	// JWTPublicKey は RS256 検証用の公開鍵 (PEM).
	JWTPublicKey []byte
	JWTTTL       time.Duration
}

// Load は環境変数から Config を読み込む.
func Load() (*Config, error) {
	port := getenv("AUTH_SERVICE_PORT", "8081")

	dbURL := os.Getenv("AUTH_DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("AUTH_DATABASE_URL is required")
	}

	privKey, err := readKey("AUTH_JWT_PRIVATE_KEY_PATH")
	if err != nil {
		return nil, err
	}
	pubKey, err := readKey("AUTH_JWT_PUBLIC_KEY_PATH")
	if err != nil {
		return nil, err
	}

	ttlStr := getenv("AUTH_JWT_TTL_SECONDS", "3600")
	ttlSec, err := strconv.Atoi(ttlStr)
	if err != nil {
		return nil, errors.New("AUTH_JWT_TTL_SECONDS must be integer")
	}

	return &Config{
		Port:          port,
		DatabaseURL:   dbURL,
		JWTPrivateKey: privKey,
		JWTPublicKey:  pubKey,
		JWTTTL:        time.Duration(ttlSec) * time.Second,
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
