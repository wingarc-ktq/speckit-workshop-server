package domain

import (
	"strings"

	"github.com/google/uuid"
)

// Tag はファイルに付与できるメタタグを表す.
type Tag struct {
	ID    uuid.UUID
	Name  string
	Color string
}

// NewTag はタグの入力値を検証して新しいタグを生成する.
func NewTag(name string, color string) (Tag, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Tag{}, ErrInvalidTagName
	}
	if len(trimmedName) > 50 {
		return Tag{}, ErrInvalidTagName
	}
	trimmedColor := strings.TrimSpace(color)
	if trimmedColor == "" {
		trimmedColor = "blue"
	}
	if len(trimmedColor) > 32 {
		return Tag{}, ErrInvalidTagColor
	}
	return Tag{
		ID:    uuid.New(),
		Name:  trimmedName,
		Color: trimmedColor,
	}, nil
}
