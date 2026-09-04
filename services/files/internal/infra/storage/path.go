package storage

import (
	"fmt"
	"path/filepath"
	"strings"
)

// safePath は root 配下の安全なファイルパスのみを許可する.
func safePath(root string, storageKey string) (string, error) {
	root = filepath.Clean(root)
	storageKey = strings.TrimSpace(storageKey)
	if storageKey == "" {
		return "", fmt.Errorf("storage key is required")
	}
	if strings.Contains(storageKey, "..") || strings.Contains(storageKey, "/") || strings.Contains(storageKey, "\\") {
		return "", fmt.Errorf("unsafe storage key: %s", storageKey)
	}
	return filepath.Join(root, storageKey), nil
}
