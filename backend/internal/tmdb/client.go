package tmdb

import (
	"Q115-STRM/internal/helpers"
	"Q115-STRM/internal/v115open"
	"fmt"
	"time"

	"golang.org/x/time/rate"
	"resty.dev/v3"
)

const (
	// DefaultBaseURL is the default TMDB API base URL
	DefaultBaseURL = "https://api.tmdb.org"
	// 重试配置
	DEFAULT_MAX_RETRIES = 3
	DEFAULT_RETRY_DELAY = 1

	// 超时配置
	DEFAULT_TIMEOUT = 30 // 秒
)

// Client represents a TMDB API client

type Client struct {
	resty       *resty.Client
	apiKey      string
	accessToken string
	baseURL     string
	language    string
	proxyUrl    string
	rateLimiter *rate.Limiter
}

var GlobalTmdbClient *Client

// NewClient creates a new TMDB API client with the given configuration
func NewClient(apiKey, accessToken, baseUrl, language, proxyUrl string) *Client {
	if GlobalTmdbClient != nil {
		// if accessToken != "" {
		// 	GlobalTmdbClient.SetAuthToken(accessToken)
		// }
		GlobalTmdbClient.SetProxyUrl(proxyUrl)
		GlobalTmdbClient.SetBaseUrl(baseUrl)
		GlobalTmdbClient.apiKey = apiKey
		GlobalTmdbClient.language = language
		return GlobalTmdbClient
	}

	// Create resty client
	rc := resty.New()
	rc.SetHeader("User-Agent", v115open.DEFAULTUA)
	GlobalTmdbClient = &Client{
		resty:       rc,
		apiKey:      apiKey,
		accessToken: accessToken,
		baseURL:     baseUrl,
		language:    language,
		proxyUrl:    proxyUrl,
	}
	// if accessToken != "" {
	// 	GlobalTmdbClient.SetAuthToken(accessToken)
	// }
	GlobalTmdbClient.SetProxyUrl(proxyUrl)
	GlobalTmdbClient.SetBaseUrl(baseUrl)
	// 设置限流
	GlobalTmdbClient.rateLimiter = rate.NewLimiter(rate.Every(time.Second), 40) // 每秒40个请求
	return GlobalTmdbClient
}

func (c *Client) SetAuthToken(accessToken string) {
	c.accessToken = accessToken
	c.resty.SetAuthToken(c.accessToken)
}

func (c *Client) SetProxyUrl(proxyUrl string) {
	c.proxyUrl = proxyUrl
	if proxyUrl != "" {
		c.resty.SetProxy(proxyUrl)
	}
}

func (c *Client) SetBaseUrl(baseUrl string) {
	c.baseURL = baseUrl
	baseURL := fmt.Sprintf("%s/3", c.baseURL)
	c.resty.SetBaseURL(baseURL)
}

func (c *Client) TestToken() bool {
	type testResp struct {
		Success bool `json:"success"`
	}
	respResult := testResp{}
	req := c.resty.R().SetMethod("GET").SetResult(&respResult)
	// req.SetQueryParam("api_key", c.apiKey)
	resp, err := c.doRequest("/authentication", req, MakeRequestConfig(2, 5, 5))
	if err != nil {
		helpers.TMDBLog.Errorf("测试token失败:%+v", err)
		return false
	}
	if !resp.IsSuccess() {
		helpers.TMDBLog.Errorf("测试token失败:%s", resp.String())
		return false
	}
	return respResult.Success
}

type ImageConfig struct {
	BaseURL       string `json:"base_url"`
	SecureBaseURL string `json:"secure_base_url"`
}

type Configuration struct {
	Images *ImageConfig `json:"images"`
}

// GetConfiguration 获取TMDB配置,包含图片基础URL等信息
func (c *Client) GetConfiguration() (*Configuration, error) {
	respResult := Configuration{}
	req := c.resty.R().SetMethod("GET").SetResult(&respResult)
	// req.SetQueryParam("api_key", c.apiKey)
	resp, err := c.doRequest("/configuration", req, MakeRequestConfig(2, 5, 5))
	if err != nil {
		helpers.TMDBLog.Errorf("获取配置失败:%+v", err)
		return nil, err
	}
	if !resp.IsSuccess() {
		helpers.TMDBLog.Errorf("获取配置失败:%s", resp.String())
		return nil, fmt.Errorf("获取配置失败:%s", resp.String())
	}
	return &respResult, nil
}

// doRequest 执行HTTP请求
func (c *Client) doRequest(url string, req *resty.Request, options *RequestConfig) (*resty.Response, error) {
	if options == nil {
		options = DefaultRequestConfig()
	}
	// 设置超时时间
	req.SetTimeout(options.Timeout)
	var lastErr error
	for attempt := 0; attempt <= options.MaxRetries; attempt++ {
		resp, err := c.request(url, req)
		if err == nil {
			// 正常返回
			return resp, nil
		}
		lastErr = err
		// 如果是token过期错误，等待token刷新完成后重试
		if err.Error() == "token expired" {
			helpers.TMDBLog.Warn("tmdb token已失效")
			return nil, fmt.Errorf("tmdb token已失效")
		}
		// 其他错误开始重试
		if attempt < options.MaxRetries {
			// helpers.TMDBLog.Warnf("%s %s 请求失败:%+v", req.Method, req.URL, lastErr)
			helpers.TMDBLog.Warnf("%s %s 请求失败，%+v秒后重试 (第%d次尝试), 错误:%+v", req.Method, req.URL, options.RetryDelay.Seconds(), attempt+1, lastErr)
			time.Sleep(options.RetryDelay)
		}
	}
	return nil, lastErr
}

func (c *Client) request(url string, req *resty.Request) (*resty.Response, error) {
	// 等待限流令牌
	if err := c.rateLimiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("rate limiter wait error: %w", err)
	}
	req.SetHeader("Accept", "application/json")
	if c.apiKey != "" {
		req.SetQueryParam("api_key", c.apiKey)
	}
	var response *resty.Response
	var err error
	switch req.Method {
	case "GET":
		response, err = req.Get(url)
	case "POST":
		response, err = req.Post(url)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", req.Method)
	}
	helpers.TMDBLog.Infof("%s %s 请求", req.Method, req.URL)
	if err != nil {
		return response, err
	}
	return response, nil
}
