package controllers

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"qmediasync/internal/helpers"

	"github.com/gin-gonic/gin"
)

func TestProxy115只允许115和百度网盘下载域名(t *testing.T) {
	cases := []struct {
		name    string
		target  string
		allowed bool
		host    string
	}{
		{
			name:    "允许 115 CDN 域名",
			target:  "https://cdnfhnfile.115cdn.net/example/video.mp4",
			allowed: true,
			host:    "cdnfhnfile.115cdn.net",
		},
		{
			name:    "允许 115 CDN 子域名",
			target:  "https://sub.cdnfhnfile.115cdn.net/example/video.mp4",
			allowed: true,
			host:    "sub.cdnfhnfile.115cdn.net",
		},
		{
			name:    "允许百度网盘 PCS 下载域名",
			target:  "https://d.pcs.baidu.com/file/example",
			allowed: true,
			host:    "d.pcs.baidu.com",
		},
		{
			name:    "允许百度网盘 PCS CDN 域名",
			target:  "https://thumbnail0.baidupcs.com/thumbnail/example",
			allowed: true,
			host:    "thumbnail0.baidupcs.com",
		},
		{
			name:    "拒绝伪造 115 后缀域名",
			target:  "https://evil115cdn.net/example",
			allowed: false,
			host:    "evil115cdn.net",
		},
		{
			name:    "拒绝非网盘域名",
			target:  "https://example.com/video.mp4",
			allowed: false,
			host:    "example.com",
		},
		{
			name:    "拒绝本机地址",
			target:  "http://127.0.0.1:12333/api/version",
			allowed: false,
			host:    "127.0.0.1",
		},
		{
			name:    "拒绝非 HTTP 协议",
			target:  "file:///etc/passwd",
			allowed: false,
		},
		{
			name:    "拒绝非法 URL",
			target:  "://bad-url",
			allowed: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotAllowed, gotHost := validateProxy115Target(tc.target)
			if gotAllowed != tc.allowed {
				t.Fatalf("validateProxy115Target(%q) allowed = %v, want %v", tc.target, gotAllowed, tc.allowed)
			}
			if gotHost != tc.host {
				t.Fatalf("validateProxy115Target(%q) host = %q, want %q", tc.target, gotHost, tc.host)
			}
		})
	}
}

func TestProxy115拒绝重定向到非网盘域名(t *testing.T) {
	gin.SetMode(gin.TestMode)
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}

	originalTransport := http.DefaultTransport
	defer func() {
		http.DefaultTransport = originalTransport
	}()

	requestedURLs := make([]string, 0)
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestedURLs = append(requestedURLs, req.URL.String())
		switch req.URL.Hostname() {
		case "d.pcs.baidu.com":
			return proxy115TestResponse(req, http.StatusFound, "", http.Header{
				"Location": []string{"http://127.0.0.1/private"},
			}), nil
		case "127.0.0.1":
			return proxy115TestResponse(req, http.StatusOK, "leaked", nil), nil
		default:
			t.Fatalf("收到未预期的请求地址：%s", req.URL.String())
			return nil, nil
		}
	})

	r := gin.New()
	r.GET("/proxy-115", Proxy115)

	target := "https://d.pcs.baidu.com/file/example"
	req := httptest.NewRequest(http.MethodGet, "/proxy-115?url="+url.QueryEscape(target), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("HTTP = %d, want %d, body=%s", w.Code, http.StatusBadGateway, w.Body.String())
	}
	if len(requestedURLs) != 1 {
		t.Fatalf("请求次数 = %d, want 1, urls=%v", len(requestedURLs), requestedURLs)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func proxy115TestResponse(req *http.Request, statusCode int, body string, header http.Header) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}
