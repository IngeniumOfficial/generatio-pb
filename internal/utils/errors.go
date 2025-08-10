package utils

import (
	"net/http"

	"generatio-pb/internal/models"

	"github.com/pocketbase/pocketbase/apis"
)

// NewValidationError creates a validation error
func NewValidationError(message string) error {
	return apis.NewBadRequestError(message, nil)
}

// NewAuthError creates an authentication error
func NewAuthError(message string) error {
	return apis.NewUnauthorizedError(message, nil)
}

// NewAuthorizationError creates an authorization error
func NewAuthorizationError(message string) error {
	return apis.NewForbiddenError(message, nil)
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) error {
	return apis.NewNotFoundError(resource+" not found", nil)
}

// NewInternalError creates an internal server error
func NewInternalError() error {
	return apis.NewApiError(http.StatusInternalServerError, "internal server error", nil)
}

// NewExternalError creates an external service error
func NewExternalError(message string) error {
	return apis.NewApiError(http.StatusBadGateway, message, nil)
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError() error {
	return apis.NewTooManyRequestsError("rate limit exceeded", nil)
}

// APIResponse creates a standardized API response
func APIResponse(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"data": data,
	}
}

// ErrorResponse creates a standardized error response
func ErrorResponse(code, message string, details interface{}) *models.APIError {
	return &models.APIError{
		Code:    code,
		Message: message,
		Details: details,
	}
}