// Package domain は Files サービスのタグ関連ドメインモデルとエラーを定義する.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// TagColor は許容されるタグ色の列挙値.
type TagColor string

const (
	TagColorBlue   TagColor = "blue"
	TagColorRed    TagColor = "red"
	TagColorYellow TagColor = "yellow"
	TagColorGreen  TagColor = "green"
	TagColorPurple TagColor = "purple"
	TagColorOrange TagColor = "orange"
	TagColorGray   TagColor = "gray"
)

// Tag はファイル整理用のタグ.
type Tag struct {
	ID        uuid.UUID
	Name      string
	Color     TagColor
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TagRepository はタグの永続化抽象.
type TagRepository interface {
	Create(ctx context.Context, tag *Tag) (*Tag, error)
	List(ctx context.Context) ([]Tag, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Tag, error)
	Update(ctx context.Context, id uuid.UUID, name string, color TagColor) (*Tag, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

var (
	ErrTagNotFound      = errors.New("tag not found")
	ErrDuplicateTagName = errors.New("duplicate tag name")
	ErrInvalidTag       = errors.New("invalid tag")
)
