package github

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestInitManager(t *testing.T) {
	// 重置默认管理器
	defaultManager = nil

	InitManager("http://proxy.example.com:8080")

	if defaultManager == nil {
		t.Error("InitManager应创建非nil的管理器")
	}

	if defaultManager.testTimeout != 3*time.Second {
		t.Errorf("期望超时时间为3秒，实际为%v", defaultManager.testTimeout)
	}

	if defaultManager.cacheValid != 10*time.Minute {
		t.Errorf("期望缓存有效期为10分钟，实际为%v", defaultManager.cacheValid)
	}

	if defaultManager.httpProxy != "http://proxy.example.com:8080" {
		t.Errorf("期望HTTP代理为http://proxy.example.com:8080，实际为%v", defaultManager.httpProxy)
	}

	// 验证GithubProxyURL常量
	if GithubProxyURL != "https://gh.llkk.cc" {
		t.Errorf("期望GithubProxyURL为https://gh.llkk.cc，实际为%v", GithubProxyURL)
	}
}

func TestGetManager(t *testing.T) {
	// 重置默认管理器
	defaultManager = nil

	manager := GetManager()

	if manager == nil {
		t.Error("GetManager应返回非nil的管理器")
	}

	// 再次调用应返回同一个实例
	manager2 := GetManager()
	if manager != manager2 {
		t.Error("GetManager应返回单例")
	}
}

func TestManager_TestConnection(t *testing.T) {
	manager := &Manager{
		testTimeout: 3 * time.Second,
	}

	tests := []struct {
		name     string
		connType ConnectionType
		proxyURL string
	}{
		{
			name:     "测试直连",
			connType: ConnectionTypeDirect,
			proxyURL: "",
		},
		{
			name:     "测试无效代理",
			connType: ConnectionTypeProxy,
			proxyURL: "http://invalid-proxy:8080",
		},
		{
			name:     "测试无效GitHub代理",
			connType: ConnectionTypeGitHubProxy,
			proxyURL: "http://invalid-github-proxy:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这里只验证不会panic，结果可能因环境而异
			got := manager.TestConnection(tt.connType, tt.proxyURL)
			t.Logf("连接测试结果: %v (连接类型: %s, 代理: %s)", got, tt.connType, tt.proxyURL)
		})
	}
}

func TestManager_TestConnection_InvalidProxyURL(t *testing.T) {
	manager := &Manager{
		testTimeout: 3 * time.Second,
	}

	// 测试无效的代理URL
	got := manager.TestConnection(ConnectionTypeProxy, "://invalid-url")
	if got {
		t.Error("无效URL应返回false")
	}
}

func TestManager_GetBestConnection(t *testing.T) {
	manager := &Manager{
		testTimeout: 3 * time.Second,
		cacheValid:  10 * time.Minute,
	}

	// 测试获取最佳连接
	access, err := manager.GetBestConnection()
	if err != nil {
		t.Logf("获取GitHub连接失败（可能网络不可达）: %v", err)
		return
	}

	if access == nil {
		t.Error("access不应为nil")
	}

	if access.Client == nil {
		t.Error("Client不应为nil")
	}

	if access.LastTested.IsZero() {
		t.Error("LastTested不应为零值")
	}

	// 验证缓存
	access2, err := manager.GetBestConnection()
	if err != nil {
		t.Error("第二次调用不应失败")
	}

	if access2.Type != access.Type {
		t.Errorf("缓存失败: 第一次 %s, 第二次 %s", access.Type, access2.Type)
	}

	// Cached字段表示是否从缓存读取，第二次调用应该是true
	if !access2.Cached {
		t.Error("第二次调用应使用缓存（Cached应为true）")
	}
}

func TestManager_GetClient(t *testing.T) {
	manager := &Manager{
		testTimeout: 3 * time.Second,
		cacheValid:  10 * time.Minute,
	}

	client, err := manager.GetClient()
	if err != nil {
		t.Logf("获取GitHub客户端失败（可能网络不可达）: %v", err)
		return
	}

	if client == nil {
		t.Error("Client不应为nil")
	}

	if client.Timeout != 30*time.Second {
		t.Errorf("Client超时应为30秒，实际为%v", client.Timeout)
	}
}

func TestManager_GetClientWithCache_NoCache(t *testing.T) {
	manager := &Manager{
		testTimeout: 3 * time.Second,
		cacheValid:  10 * time.Minute,
	}

	// 没有缓存时调用
	_, err := manager.GetClientWithCache()
	if err == nil {
		t.Error("没有缓存时应返回错误")
	}
}

func TestManager_GetClientWithCache_WithCache(t *testing.T) {
	manager := &Manager{
		testTimeout: 3 * time.Second,
		cacheValid:  10 * time.Minute,
		current: &GitHubAccess{
			Type:   ConnectionTypeDirect,
			Client: &http.Client{Timeout: 30 * time.Second},
		},
	}

	client, err := manager.GetClientWithCache()
	if err != nil {
		t.Errorf("有缓存时不应返回错误: %v", err)
	}

	if client == nil {
		t.Error("Client不应为nil")
	}
}

func TestManager_ClearCache(t *testing.T) {
	manager := &Manager{
		testTimeout: 3 * time.Second,
		cacheValid:  10 * time.Minute,
	}

	// 设置缓存
	manager.current = &GitHubAccess{
		Type:       ConnectionTypeDirect,
		Client:     &http.Client{Timeout: 30 * time.Second},
		LastTested: time.Now(),
		Cached:     true,
	}

	// 清除缓存
	manager.ClearCache()

	if manager.current != nil {
		t.Error("缓存应被清除")
	}
}

func TestManager_ConnectionPriority(t *testing.T) {
	// 这个测试模拟有缓存的场景
	manager := &Manager{
		testTimeout: 3 * time.Second,
		cacheValid:  10 * time.Minute,
	}

	// 先设置一个代理缓存
	proxyURL := "http://proxy.example.com:8080"
	proxy, _ := url.Parse(proxyURL)

	manager.current = &GitHubAccess{
		Type: ConnectionTypeProxy,
		Client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: &http.Transport{Proxy: http.ProxyURL(proxy)},
		},
		ProxyURL:   proxyURL,
		LastTested: time.Now(),
		Cached:     true,
	}

	// 应该立即返回缓存的代理连接
	access, err := manager.GetBestConnection()
	if err != nil {
		t.Errorf("应返回缓存的连接: %v", err)
	}

	if access.Type != ConnectionTypeProxy {
		t.Errorf("应返回代理连接，实际为%v", access.Type)
	}

	if !access.Cached {
		t.Error("应标记为缓存")
	}
}

func TestConnectionType_String(t *testing.T) {
	tests := []struct {
		name string
		ct   ConnectionType
		want string
	}{
		{
			name: "直连",
			ct:   ConnectionTypeDirect,
			want: "direct",
		},
		{
			name: "代理",
			ct:   ConnectionTypeProxy,
			want: "proxy",
		},
		{
			name: "GitHub代理",
			ct:   ConnectionTypeGitHubProxy,
			want: "github_proxy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.ct) != tt.want {
				t.Errorf("期望%q, 实际%q", tt.want, tt.ct)
			}
		})
	}
}
