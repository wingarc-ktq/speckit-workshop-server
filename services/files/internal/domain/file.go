package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const MaxFileSize = 10 * 1024 * 1024

type File struct {
	ID          uuid.UUID
	Name        string
	Size        int64
	MIMEType    string
	Description string
	StorageKey  string
	UploadedAt  time.Time
	TagIDs      []uuid.UUID
}

func NewFile(name string, mimeType string, description string, storageKey string, size int64, tagIDs []uuid.UUID) (File, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return File{}, ErrInvalidFileName
	}
	if mimeType == "" {
		return File{}, ErrInvalidMIMEType
	}
	if description != "" && (len(description) > 500 || !utf8.ValidString(description) || containsControlCharacter(description)) {
		return File{}, ErrInvalidDescription
	}
	if size < 0 || size > MaxFileSize {
		return File{}, ErrFileTooLarge
	}

	copyIDs := append([]uuid.UUID(nil), tagIDs...)
	return File{
		ID:          uuid.New(),
		Name:        name,
		Size:        size,
		MIMEType:    mimeType,
		Description: description,
		StorageKey:  storageKey,
		UploadedAt:  time.Now().UTC(),
		TagIDs:      copyIDs,
	}, nil
}

func containsControlCharacter(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}
