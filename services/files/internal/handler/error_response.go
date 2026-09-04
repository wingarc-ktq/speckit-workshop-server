package handler

import (
	"errors"
	"net/http"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
)

// ErrorResponseFor maps a domain error to the HTTP status code and common error payload.
func ErrorResponseFor(err error) (int, gen.ErrorResponse) {
	switch {
	case err == nil:
		return http.StatusOK, gen.ErrorResponse{Code: "OK", Message: "ok"}
	case errors.Is(err, domain.ErrInvalidFileName),
		errors.Is(err, domain.ErrInvalidMIMEType),
		errors.Is(err, domain.ErrInvalidDescription),
		errors.Is(err, domain.ErrInvalidTagName),
		errors.Is(err, domain.ErrInvalidTagColor):
		return http.StatusBadRequest, gen.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}
	case errors.Is(err, domain.ErrFileTooLarge):
		return http.StatusRequestEntityTooLarge, gen.ErrorResponse{Code: "FILE_TOO_LARGE", Message: err.Error()}
	case errors.Is(err, domain.ErrTagNotFound):
		return http.StatusConflict, gen.ErrorResponse{Code: "TAG_NOT_FOUND", Message: err.Error()}
	case errors.Is(err, domain.ErrFileNotFound):
		return http.StatusNotFound, gen.ErrorResponse{Code: "FILE_NOT_FOUND", Message: err.Error()}
	default:
		return http.StatusInternalServerError, gen.ErrorResponse{Code: "INTERNAL_ERROR", Message: err.Error()}
	}
}
