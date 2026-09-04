package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/labstack/echo/v4"
	oapimiddleware "github.com/oapi-codegen/echo-middleware"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
)

func newOpenAPIValidator() (echo.MiddlewareFunc, error) {
	spec, err := gen.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}
	spec.Servers = openapi3.Servers{{URL: basePath}}

	return oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
		SilenceServersWarning: true,
		Skipper: func(c echo.Context) bool {
			return !strings.HasPrefix(c.Request().URL.Path, basePath)
		},
		Options: openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
		ErrorHandler: func(c echo.Context, err *echo.HTTPError) error {
			code := "VALIDATION_ERROR"
			status := err.Code
			msg := fmt.Sprintf("%v", err.Message)
			switch {
			case err.Code == http.StatusNotFound && msg == routers.ErrMethodNotAllowed.Error():
				status = http.StatusMethodNotAllowed
				code = "METHOD_NOT_ALLOWED"
				if allow, ok := c.Get(echo.ContextKeyHeaderAllow).(string); ok && allow != "" {
					c.Response().Header().Set(echo.HeaderAllow, allow)
				}
			case err.Code == http.StatusNotFound:
				code = "NOT_FOUND"
			}
			return c.JSON(status, gen.ErrorResponse{Code: code, Message: msg})
		},
	}), nil
}
