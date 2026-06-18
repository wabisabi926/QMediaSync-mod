package helpers

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// TestHttpProxy 测试HTTP代理连接
func TestHttpProxy(proxyURL string) (bool, error) {
	if proxyURL == "" {
		return false, fmt.Errorf("代理URL不能为空")
	}

	// 验证代理URL格式
	parsedProxy, err := url.Parse(proxyURL)
	if err != nil {
		return false, fmt.Errorf("代理URL格式无效: %v", err)
	}

	// 检查协议
	if parsedProxy.Scheme != "http" && parsedProxy.Scheme != "https" {
		return false, fmt.Errorf("不支持的代理协议: %s，仅支持HTTP/HTTPS协议", parsedProxy.Scheme)
	}

	// 创建HTTP客户端，使用HTTP代理
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(parsedProxy),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
		Timeout: 30 * time.Second,
	}

	// 测试URL列表，按优先级排序
	testURLs := []string{
		"https://api.github.com",  // GitHub API，稳定可靠
		"https://www.google.com",  // Google首页
		"http://www.baidu.com",    // 百度首页，国内访问
		"https://httpstat.us/200", // HTTP状态测试服务
	}

	var lastError error

	for _, testURL := range testURLs {
		AppLogger.Infof("使用代理 %s 测试连接到 %s", proxyURL, testURL)

		req, err := http.NewRequest("GET", testURL, nil)
		if err != nil {
			lastError = fmt.Errorf("创建请求失败: %v", err)
			continue
		}

		// 设置请求头，模拟正常浏览器请求
		req.Header.Set("User-Agent", "Q115-STRM/1.0 (Proxy Test)")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		req.Header.Set("Accept-Encoding", "gzip, deflate")
		req.Header.Set("Connection", "keep-alive")

		resp, err := client.Do(req)
		if err != nil {
			lastError = fmt.Errorf("请求失败 [%s]: %v", testURL, err)
			AppLogger.Warnf("代理测试失败 [%s]: %v", testURL, err)
			continue
		}
		defer resp.Body.Close()

		// 检查响应状态
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			AppLogger.Infof("代理连接测试成功 [%s]: HTTP %d", testURL, resp.StatusCode)
			return true, nil
		} else {
			lastError = fmt.Errorf("HTTP响应异常 [%s]: %d %s", testURL, resp.StatusCode, resp.Status)
			AppLogger.Warnf("代理测试响应异常 [%s]: %d", testURL, resp.StatusCode)
		}
	}

	// 所有测试URL都失败了
	if lastError != nil {
		return false, fmt.Errorf("代理连接测试失败: %v", lastError)
	}

	return false, fmt.Errorf("代理连接测试失败: 所有测试URL都无法访问")
}

// TestHttpProxyAdvanced 高级代理测试，返回更详细的信息
func TestHttpProxyAdvanced(proxyURL string) (*ProxyTestResult, error) {
	result := &ProxyTestResult{
		ProxyURL:    proxyURL,
		TestTime:    time.Now(),
		TestResults: make([]TestURLResult, 0),
	}

	if proxyURL == "" {
		result.Success = false
		result.ErrorMessage = "代理URL不能为空"
		return result, fmt.Errorf("代理URL不能为空")
	}

	// 验证代理URL格式
	parsedProxy, err := url.Parse(proxyURL)
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("代理URL格式无效: %v", err)
		return result, err
	}

	result.ProxyScheme = parsedProxy.Scheme
	result.ProxyHost = parsedProxy.Host

	// 检查协议
	if parsedProxy.Scheme != "http" && parsedProxy.Scheme != "https" {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("不支持的代理协议: %s，仅支持HTTP/HTTPS协议", parsedProxy.Scheme)
		return result, fmt.Errorf("不支持的代理协议: %s", parsedProxy.Scheme)
	}

	// 创建HTTP客户端，使用HTTP代理
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(parsedProxy),
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
		Timeout: 30 * time.Second,
	}

	// 测试URL列表
	testURLs := []TestURL{
		{URL: "http://httpbin.org/ip", Description: "IP检测服务"},
		{URL: "https://api.github.com", Description: "GitHub API"},
		{URL: "https://www.google.com", Description: "Google首页"},
		{URL: "http://www.baidu.com", Description: "百度首页"},
		{URL: "https://httpstat.us/200", Description: "HTTP状态测试"},
	}

	successCount := 0

	for _, testURL := range testURLs {
		testResult := TestURLResult{
			URL:         testURL.URL,
			Description: testURL.Description,
			StartTime:   time.Now(),
		}

		req, err := http.NewRequest("GET", testURL.URL, nil)
		if err != nil {
			testResult.Success = false
			testResult.ErrorMessage = fmt.Sprintf("创建请求失败: %v", err)
			testResult.Duration = time.Since(testResult.StartTime)
			result.TestResults = append(result.TestResults, testResult)
			continue
		}

		// 设置请求头
		req.Header.Set("User-Agent", "Q115-STRM/1.0 (Proxy Test)")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

		resp, err := client.Do(req)
		testResult.Duration = time.Since(testResult.StartTime)

		if err != nil {
			testResult.Success = false
			testResult.ErrorMessage = fmt.Sprintf("请求失败: %v", err)
		} else {
			defer resp.Body.Close()
			testResult.StatusCode = resp.StatusCode
			testResult.StatusText = resp.Status

			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				testResult.Success = true
				successCount++
			} else {
				testResult.Success = false
				testResult.ErrorMessage = fmt.Sprintf("HTTP响应异常: %d %s", resp.StatusCode, resp.Status)
			}
		}

		result.TestResults = append(result.TestResults, testResult)
	}

	// 如果至少有一个测试成功，则认为代理可用
	if successCount > 0 {
		result.Success = true
		result.SuccessCount = successCount
		result.TotalCount = len(testURLs)
	} else {
		result.Success = false
		result.ErrorMessage = "所有测试URL都无法通过代理访问"
	}

	return result, nil
}

// createProxyTransport 创建代理传输
func createProxyTransport(proxyURL string) (*http.Transport, error) {
	if proxyURL == "" {
		return &http.Transport{
			// 默认传输配置
			TLSHandshakeTimeout:   30 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			// 自定义Dialer
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		}, nil
	}

	// 解析代理URL
	parsedProxy, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("代理URL格式无效: %v", err)
	}

	// 检查是否为HTTP代理
	if parsedProxy.Scheme != "http" && parsedProxy.Scheme != "https" {
		return nil, fmt.Errorf("不支持的代理协议: %s，仅支持HTTP/HTTPS协议", parsedProxy.Scheme)
	}

	// 创建HTTP代理传输配置
	transport := &http.Transport{
		Proxy: http.ProxyURL(parsedProxy),
		// TLS设置
		TLSHandshakeTimeout: 60 * time.Second, // 增加TLS握手超时
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false, // 保持证书验证
		},
		// HTTP设置
		ResponseHeaderTimeout: 60 * time.Second, // 增加响应头超时
		ExpectContinueTimeout: 5 * time.Second,  // 增加ExpectContinue超时
		// 连接池设置
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		// 启用HTTP/2
		ForceAttemptHTTP2: true,
		// 自定义Dialer以支持更好的网络控制
		DialContext: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	return transport, nil
}

// GetProxyTransport 获取代理传输配置的便捷函数
func GetProxyTransport(proxyURL string) *http.Transport {
	transport, err := createProxyTransport(proxyURL)
	if err != nil {
		AppLogger.Warnf("创建代理传输失败: %v", err)
		return &http.Transport{} // 返回默认传输
	}
	return transport
}

// ProxyTestResult 代理测试结果
type ProxyTestResult struct {
	ProxyURL     string          `json:"proxy_url"`
	ProxyScheme  string          `json:"proxy_scheme"`
	ProxyHost    string          `json:"proxy_host"`
	Success      bool            `json:"success"`
	SuccessCount int             `json:"success_count"`
	TotalCount   int             `json:"total_count"`
	ErrorMessage string          `json:"error_message,omitempty"`
	TestTime     time.Time       `json:"test_time"`
	TestResults  []TestURLResult `json:"test_results"`
}

// TestURL 测试URL
type TestURL struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// TestURLResult 单个URL测试结果
type TestURLResult struct {
	URL          string        `json:"url"`
	Description  string        `json:"description"`
	Success      bool          `json:"success"`
	StatusCode   int           `json:"status_code,omitempty"`
	StatusText   string        `json:"status_text,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
	StartTime    time.Time     `json:"start_time"`
	Duration     time.Duration `json:"duration"`
}
