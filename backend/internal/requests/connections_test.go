package requests

import "testing"

func TestConnectionRequestValidate(t *testing.T) {
	t.Run("保存 HTTP 代理允许空", func(t *testing.T) {
		req := HTTPProxyRequest{}
		if err := req.ValidateSave(); err != nil {
			t.Fatalf("ValidateSave() error = %v", err)
		}
	})

	t.Run("测试 HTTP 代理必须非空", func(t *testing.T) {
		req := HTTPProxyRequest{}
		if err := req.ValidateTest(); err == nil {
			t.Fatal("ValidateTest() error = nil, want error")
		}
	})

	t.Run("代理 detailed 枚举错误失败", func(t *testing.T) {
		req := HTTPProxyRequest{HTTPProxy: "http://127.0.0.1:7890", Detailed: 2}
		if err := req.ValidateTest(); err == nil {
			t.Fatal("ValidateTest() error = nil, want error")
		}
	})

	t.Run("账号 ID 通过", func(t *testing.T) {
		req := AccountIDRequest{AccountID: 1}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})
}

func TestOAuthRequestValidate(t *testing.T) {
	t.Run("OAuth URL 请求通过", func(t *testing.T) {
		req := OAuthURLRequest{AccountID: 1, RedirectURL: "https://example.com/callback"}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("OAuth URL redirect 非法失败", func(t *testing.T) {
		req := OAuthURLRequest{AccountID: 1, RedirectURL: "ftp://example.com"}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})

	t.Run("OAuth Confirm 缺少 data 和 payload 失败", func(t *testing.T) {
		req := OAuthConfirmRequest{AccountID: 1}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})

	t.Run("OAuth Status 通过", func(t *testing.T) {
		req := OAuthStatusRequest{AccountID: 1, State: "state"}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})
}

func TestRemoteFileURLRequestValidate(t *testing.T) {
	t.Run("PickCode 查询通过", func(t *testing.T) {
		req := RemoteFileURLRequest{PickCode: "abc", Force: 1}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("PickCode 为空失败", func(t *testing.T) {
		req := RemoteFileURLRequest{}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})

	t.Run("Force 枚举错误失败", func(t *testing.T) {
		req := RemoteFileURLRequest{PickCode: "abc", Force: 2}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})
}

func TestProxy115RequestValidate(t *testing.T) {
	t.Run("115 CDN URL 通过", func(t *testing.T) {
		req := Proxy115Request{URL: "https://cdn.115cdn.net/file.mp4"}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("未知域名失败", func(t *testing.T) {
		req := Proxy115Request{URL: "https://example.com/file.mp4"}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})
}

func TestConnectionAuxRequestValidate(t *testing.T) {
	t.Run("队列限流通过", func(t *testing.T) {
		req := QueueRateLimitRequest{QPS: 1, QPM: 60, QPH: 3600}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("清理统计天数超限失败", func(t *testing.T) {
		req := CleanRequestStatsRequest{Days: 366}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})

	t.Run("OpenList 直链请求通过", func(t *testing.T) {
		req := OpenListFileURLRequest{AccountID: 1, Path: "/movie.mp4"}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})
}
