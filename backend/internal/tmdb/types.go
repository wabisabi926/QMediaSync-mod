package tmdb

import "time"

// RequestConfig 请求配置
type RequestConfig struct {
	MaxRetries int           `json:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay"`
	Timeout    time.Duration `json:"timeout"`
}

// DefaultRequestConfig 默认请求配置
func DefaultRequestConfig() *RequestConfig {
	return &RequestConfig{
		MaxRetries: DEFAULT_MAX_RETRIES,
		RetryDelay: DEFAULT_RETRY_DELAY * time.Second,
		Timeout:    DEFAULT_TIMEOUT * time.Second,
	}
}

func MakeRequestConfig(maxRetries int, retryDelay time.Duration, timeout time.Duration) *RequestConfig {
	config := DefaultRequestConfig()
	config.MaxRetries = maxRetries
	if retryDelay > 0 {
		config.RetryDelay = retryDelay * time.Second
	}
	if timeout > 0 {
		config.Timeout = timeout * time.Second
	}
	return config
}
