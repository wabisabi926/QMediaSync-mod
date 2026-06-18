package open123

import "fmt"

type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error: code=%d, message=%s", e.Code, e.Message)
}

func NewAPIError(code int, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

const (
	ErrCodeSuccess            = 0
	ErrCodeInvalidParams      = 400
	ErrCodeUnauthorized       = 401
	ErrCodeForbidden          = 403
	ErrCodeNotFound           = 404
	ErrCodeRateLimit          = 429
	ErrCodeInternalError      = 500
	ErrCodeServiceUnavailable = 503
)

func IsTokenExpired(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Code == ErrCodeUnauthorized
	}
	return false
}

func IsRateLimited(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Code == ErrCodeRateLimit
	}
	return false
}
