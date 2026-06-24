package v115open

import "fmt"

// OpenAPIError 保留 115 开放平台返回的原始错误信息。
type OpenAPIError struct {
	Code    int
	Message string
}

func NewOpenAPIError(code int, message string) *OpenAPIError {
	if message == "" {
		message = "未知错误"
	}
	return &OpenAPIError{Code: code, Message: message}
}

func NewOpenAPIResponseError(code int, errno int, message string, errorText string, fallback string) error {
	if code == 0 {
		code = errno
	}
	if message == "" {
		message = errorText
	}
	if code == 0 && message == "" {
		return fmt.Errorf("%s", fallback)
	}
	return NewOpenAPIError(code, message)
}

func (e *OpenAPIError) Error() string {
	if e.Code == 0 {
		return fmt.Sprintf("115 接口错误：%s", e.Message)
	}
	return fmt.Sprintf("115 接口错误（%d）：%s", e.Code, e.Message)
}
