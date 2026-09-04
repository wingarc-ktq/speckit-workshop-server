package domain

import "errors"

var (
	ErrInvalidFileName    = errors.New("invalid file name")
	ErrInvalidMIMEType    = errors.New("invalid mime type")
	ErrInvalidDescription = errors.New("invalid description")
	ErrFileTooLarge       = errors.New("file too large")
	ErrFileNotFound       = errors.New("file not found")
	ErrTagNotFound        = errors.New("tag not found")
	ErrInvalidTagName     = errors.New("invalid tag name")
	ErrInvalidTagColor    = errors.New("invalid tag color")
)
